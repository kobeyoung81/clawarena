package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/clawarena/clawarena/internal/game"
	"github.com/clawarena/clawarena/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// RoomHub manages SSE subscriber channels per room.
type RoomHub struct {
	mu   sync.RWMutex
	subs map[uint][]chan []byte
}

func NewRoomHub() *RoomHub {
	return &RoomHub{subs: map[uint][]chan []byte{}}
}

func (h *RoomHub) Subscribe(roomID uint) chan []byte {
	ch := make(chan []byte, 32)
	h.mu.Lock()
	h.subs[roomID] = append(h.subs[roomID], ch)
	h.mu.Unlock()
	return ch
}

func (h *RoomHub) Unsubscribe(roomID uint, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	subs := h.subs[roomID]
	for i, s := range subs {
		if s == ch {
			h.subs[roomID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

func (h *RoomHub) Broadcast(roomID uint, data []byte) {
	h.mu.RLock()
	subs := h.subs[roomID]
	h.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- data:
		default:
			// subscriber too slow; skip
		}
	}
}

func (h *RoomHub) CloseRoom(roomID uint) {
	h.mu.Lock()
	subs := h.subs[roomID]
	delete(h.subs, roomID)
	h.mu.Unlock()
	for _, ch := range subs {
		close(ch)
	}
}

// WatchHandler handles SSE streams.
type WatchHandler struct {
	db  *gorm.DB
	hub *RoomHub
}

func NewWatchHandler(db *gorm.DB, hub *RoomHub) *WatchHandler {
	return &WatchHandler{db: db, hub: hub}
}

func (h *WatchHandler) Watch(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id", "INVALID_REQUEST")
		return
	}

	var room models.Room
	if err := h.db.Preload("GameType").Preload("Agents.Agent").First(&room, roomID).Error; err != nil {
		writeError(w, http.StatusNotFound, "room not found", "NOT_FOUND")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	eng := game.Registry[room.GameType.Name]

	agents := roomAgentsInfo(room.Agents)

	// buildSpectatorSnapshot builds the enriched fields (spectator state, pending action, phase)
	// from a raw game state. Returns spectatorView, pendingAction, currentAgentID, phase.
	buildSpectatorSnapshot := func(rawState json.RawMessage) (json.RawMessage, *map[string]any, *uint, string) {
		stateView := rawState
		if eng != nil {
			if sv, err := eng.GetSpectatorView(rawState); err == nil {
				stateView = sv
			}
		}

		var pendingAction *map[string]any
		var currentAgentID *uint
		if eng != nil {
			if pending, err := eng.GetPendingActions(rawState); err == nil && len(pending) > 0 {
				id := pending[0].PlayerID
				currentAgentID = &id
				pa := map[string]any{
					"player_id":   pending[0].PlayerID,
					"action_type": pending[0].ActionType,
					"prompt":      pending[0].Prompt,
				}
				pendingAction = &pa
			}
		}

		var phase string
		var stateMap map[string]any
		if json.Unmarshal(stateView, &stateMap) == nil {
			if p, ok := stateMap["phase"].(string); ok {
				phase = p
			}
		}
		return stateView, pendingAction, currentAgentID, phase
	}

	// enrichEvent injects room_id, turn, and status into every SSE event.
	enrichEvent := func(raw []byte, turnNum uint, fallbackStatus string) []byte {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return raw
		}
		m["room_id"] = roomID
		m["turn"] = turnNum
		if _, ok := m["status"]; !ok {
			if gameOver, _ := m["game_over"].(bool); gameOver {
				m["status"] = "finished"
			} else {
				m["status"] = fallbackStatus
			}
		}
		out, err := json.Marshal(m)
		if err != nil {
			return raw
		}
		return out
	}

	// Send initial full state immediately on connect
	{
		var gs models.GameState
		q := h.db.Where("room_id = ?", roomID)
		if room.CurrentGameID != nil {
			q = q.Where("game_id = ?", *room.CurrentGameID)
		}
		if q.Order("turn DESC").First(&gs).Error == nil {
			stateView, pendingAction, currentAgentID, phase := buildSpectatorSnapshot(json.RawMessage(gs.State))
			initEvent := map[string]any{
				"turn":             gs.Turn,
				"state":            stateView,
				"agents":           agents,
				"pending_action":   pendingAction,
				"current_agent_id": currentAgentID,
				"phase":            phase,
				"game_over":        false,
				"status":           string(room.Status),
				"room_id":          roomID,
				"game_type":        room.GameType.Name,
			}
			data, _ := json.Marshal(initEvent)
			fmt.Fprintf(w, "id: %d\ndata: %s\n\n", gs.Turn, data)
			flusher.Flush()
		}
	}

	// Replay missed events via Last-Event-ID
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		if lastTurn, err := strconv.ParseUint(lastEventID, 10, 64); err == nil {
			// Load game states for replayed turns to build full snapshots
			var states []models.GameState
			qStates := h.db.Where("room_id = ? AND turn > ?", roomID, lastTurn)
			if room.CurrentGameID != nil {
				qStates = qStates.Where("game_id = ?", *room.CurrentGameID)
			}
			qStates.Order("turn ASC").Find(&states)
			stateByTurn := map[uint]models.GameState{}
			for _, gs := range states {
				stateByTurn[gs.Turn] = gs
			}

			var actions []models.GameAction
			qActions := h.db.Preload("Agent").Where("room_id = ? AND turn > ?", roomID, lastTurn)
			if room.CurrentGameID != nil {
				qActions = qActions.Where("game_id = ?", *room.CurrentGameID)
			}
			qActions.Order("turn ASC").Find(&actions)
			for _, act := range actions {
				replayData := map[string]any{
					"turn":     act.Turn,
					"agent":    act.Agent.Name,
					"action":   json.RawMessage(act.Action),
					"replayed": true,
					"agents":   agents,
				}
				// Enrich with state snapshot if available
				if gs, ok := stateByTurn[act.Turn]; ok {
					stateView, pendingAction, currentAgentID, phase := buildSpectatorSnapshot(json.RawMessage(gs.State))
					replayData["state"] = stateView
					replayData["pending_action"] = pendingAction
					replayData["current_agent_id"] = currentAgentID
					replayData["phase"] = phase
				}
				raw, _ := json.Marshal(replayData)
				data := enrichEvent(raw, uint(act.Turn), "playing")
				fmt.Fprintf(w, "id: %d\ndata: %s\n\n", act.Turn, data)
			}
			flusher.Flush()
		}
	}

	// If game is already in a terminal state, send a final event and close
	if room.Status == models.RoomFinished || room.Status == models.RoomCancelled || room.Status == models.RoomDead {
		raw, _ := json.Marshal(map[string]any{
			"type":      "game_over",
			"status":    string(room.Status),
			"game_over": true,
			"agents":    agents,
		})
		data := enrichEvent(raw, 0, string(room.Status))
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	ch := h.hub.Subscribe(uint(roomID))
	defer h.hub.Unsubscribe(uint(roomID), ch)

	// Send a keep-alive comment every 15 seconds
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	var turn uint
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		case msg, open := <-ch:
			if !open {
				turn++
				raw, _ := json.Marshal(map[string]any{"type": "room_closed", "game_over": true})
				data := enrichEvent(raw, turn, "cancelled")
				fmt.Fprintf(w, "id: %d\ndata: %s\n\n", turn, data)
				flusher.Flush()
				return
			}
			turn++
			data := enrichEvent(msg, turn, "playing")
			fmt.Fprintf(w, "id: %d\ndata: %s\n\n", turn, data)
			flusher.Flush()
		}
	}
}

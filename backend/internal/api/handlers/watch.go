package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/clawarena/clawarena/internal/api/dto"
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

// WatchHandler handles SSE streams for spectators.
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

	eng := game.GetEngine(room.GameType.Name)
	agents := roomAgentsInfo(room.Agents)

	// buildSpectatorEvent builds an SSE event payload from a stored event record.
	buildSpectatorEvent := func(rec *game.BaseGameEvent, eng game.GameEngine, roomID uint, gameType string, agents []dto.RoomAgentInfo, status string) dto.SSEEventPayload {
		evt := rec.ToGameEvent()

		stateView := evt.StateAfter
		if eng != nil {
			if sv, err := eng.GetSpectatorView(evt.StateAfter); err == nil {
				stateView = sv
			}
		}

		var pendingDTO *dto.PendingActionDTO
		var currentAgentID *uint
		if eng != nil {
			if pending, err := eng.GetPendingActions(evt.StateAfter); err == nil && len(pending) > 0 {
				id := pending[0].PlayerID
				currentAgentID = &id
				pendingDTO = &dto.PendingActionDTO{
					PlayerID:   pending[0].PlayerID,
					ActionType: pending[0].ActionType,
					Prompt:     pending[0].Prompt,
				}
			}
		}

		var resultDTO *dto.GameResultDTO
		if evt.Result != nil {
			resultDTO = &dto.GameResultDTO{
				WinnerIDs:  evt.Result.WinnerIDs,
				WinnerTeam: evt.Result.WinnerTeam,
				Scores:     evt.Result.Scores,
			}
		}

		payload := dto.SSEEventPayload{
			Seq:            rec.Seq,
			GameID:         rec.GameID,
			RoomID:         roomID,
			Source:         evt.Source,
			EventType:      evt.EventType,
			Actor:          evt.Actor,
			Target:         evt.Target,
			Details:        evt.Details,
			State:          stateView,
			Visibility:     evt.Visibility,
			PendingAction:  pendingDTO,
			CurrentAgentID: currentAgentID,
			Agents:         agents,
			GameType:       gameType,
			GameOver:       evt.GameOver,
			Result:         resultDTO,
			Status:         status,
		}
		return payload
	}

	// Send all events for the current game as catch-up on connect.
	var lastSeq uint
	if eng != nil && room.CurrentGameID != nil {
		tableName := eng.NewEventModel().TableName()

		// Check for Last-Event-ID to resume from a specific point
		startSeq := uint(0)
		if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
			if parsed, err := strconv.ParseUint(lastEventID, 10, 64); err == nil {
				startSeq = uint(parsed)
			}
		}

		var records []game.BaseGameEvent
		q := h.db.Table(tableName).Where("game_id = ?", *room.CurrentGameID)
		if startSeq > 0 {
			q = q.Where("seq > ?", startSeq)
		}
		q.Order("seq ASC").Find(&records)

		for _, rec := range records {
			evt := rec.ToGameEvent()
			// Only send public events to spectators
			if evt.Visibility != "public" {
				continue
			}
			status := string(room.Status)
			if evt.GameOver {
				status = "intermission"
			}
			payload := buildSpectatorEvent(&rec, eng, uint(roomID), room.GameType.Name, agents, status)
			data, _ := json.Marshal(payload)
			fmt.Fprintf(w, "id: %d\nevent: game_event\ndata: %s\n\n", rec.Seq, data)
			lastSeq = rec.Seq
		}
		flusher.Flush()
	}

	// If the room is in a terminal state, close after catch-up
	if room.Status == models.RoomClosed {
		data, _ := json.Marshal(map[string]any{
			"room_id":   roomID,
			"game_over": true,
			"status":    string(room.Status),
			"agents":    agents,
		})
		fmt.Fprintf(w, "event: game_event\ndata: %s\n\n", data)
		flusher.Flush()
		return
	}

	ch := h.hub.Subscribe(uint(roomID))
	defer h.hub.Unsubscribe(uint(roomID), ch)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		case msg, open := <-ch:
			if !open {
				data, _ := json.Marshal(map[string]any{
					"room_id":    roomID,
					"game_over":  true,
					"event_type": "room_closed",
					"status":     "closed",
				})
				fmt.Fprintf(w, "event: game_event\ndata: %s\n\n", data)
				flusher.Flush()
				return
			}

			// Parse the broadcast payload
			var broadcast dto.SSEEventPayload
			if err := json.Unmarshal(msg, &broadcast); err != nil {
				// Fallback: forward raw message (e.g., room-level events like player_joined)
				lastSeq++
				fmt.Fprintf(w, "id: %d\nevent: game_event\ndata: %s\n\n", lastSeq, msg)
				flusher.Flush()
				continue
			}

			// Filter: only public events for spectators
			if broadcast.Visibility != "" && broadcast.Visibility != "public" {
				continue
			}

			// Replace state with spectator view
			if eng != nil && len(broadcast.State) > 0 {
				if sv, err := eng.GetSpectatorView(broadcast.State); err == nil {
					broadcast.State = sv
				}
			}

			data, _ := json.Marshal(broadcast)
			fmt.Fprintf(w, "id: %d\nevent: game_event\ndata: %s\n\n", broadcast.Seq, data)
			flusher.Flush()
		}
	}
}

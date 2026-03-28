package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/clawarena/clawarena/internal/api/dto"
	"github.com/clawarena/clawarena/internal/api/middleware"
	"github.com/clawarena/clawarena/internal/game"
	"github.com/clawarena/clawarena/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// PlayHandler handles authenticated SSE streams for agents playing in a room.
type PlayHandler struct {
	db  *gorm.DB
	hub *RoomHub
}

func NewPlayHandler(db *gorm.DB, hub *RoomHub) *PlayHandler {
	return &PlayHandler{db: db, hub: hub}
}

// Play streams player-specific game events over SSE to an authenticated agent.
func (h *PlayHandler) Play(w http.ResponseWriter, r *http.Request) {
	// Authenticate agent before writing any headers.
	claims := middleware.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}
	agent, err := GetOrProvisionByAuthUID(h.db, claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agent", "INTERNAL_ERROR")
		return
	}

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

	// Verify agent is a member of this room (and not KIA).
	inRoom := false
	for _, ra := range room.Agents {
		if ra.AgentID == agent.ID && ra.Status != models.RoomAgentKIA {
			inRoom = true
			break
		}
	}
	if !inRoom {
		writeError(w, http.StatusForbidden, "you are not in this room", "NOT_IN_ROOM")
		return
	}

	// Mark agent as reconnected (clear disconnect timestamp, set status active)
	h.db.Model(&models.RoomAgent{}).
		Where("room_id = ? AND agent_id = ?", roomID, agent.ID).
		Updates(map[string]any{"disconnected_at": nil, "status": string(models.RoomAgentActive)})

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
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
				m["status"] = "post_game"
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

	// buildPlayerEvent builds a player-specific SSE payload from a game state.
	buildPlayerEvent := func(gs *models.GameState, eng game.GameEngine, status string) []byte {
		stateView, err := eng.GetPlayerView(json.RawMessage(gs.State), agent.ID)
		if err != nil {
			stateView = json.RawMessage(gs.State)
		}

		// Reload agents for fresh data
		var freshAgents []models.RoomAgent
		h.db.Preload("Agent").Where("room_id = ?", roomID).Find(&freshAgents)

		pending, _ := eng.GetPendingActions(json.RawMessage(gs.State))
		resp := dto.GameStateResponse{
			RoomID: uint(roomID),
			Status: status,
			Turn:   gs.Turn,
			State:  stateView,
			Agents: roomAgentsInfo(freshAgents),
		}
		for _, pa := range pending {
			if resp.CurrentAgentID == nil {
				id := pa.PlayerID
				resp.CurrentAgentID = &id
			}
			if pa.PlayerID == agent.ID {
				resp.PendingAction = &dto.PendingActionDTO{
					PlayerID:     pa.PlayerID,
					ActionType:   pa.ActionType,
					Prompt:       pa.Prompt,
					ValidTargets: pa.ValidTargets,
				}
			}
		}
		data, _ := json.Marshal(resp)
		return data
	}

	eng, engOK := game.Registry[room.GameType.Name]

	// Replay missed events via Last-Event-ID
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		if lastTurn, err := strconv.ParseUint(lastEventID, 10, 64); err == nil {
			var states []models.GameState
			h.db.Where("room_id = ? AND turn > ?", roomID, lastTurn).
				Order("turn ASC").Find(&states)
			for _, gs := range states {
				if engOK {
					data := buildPlayerEvent(&gs, eng, string(room.Status))
					data = enrichEvent(data, gs.Turn, string(room.Status))
					fmt.Fprintf(w, "id: %d\ndata: %s\n\n", gs.Turn, data)
				}
			}
			flusher.Flush()
		}
	}

	// If the room is in a terminal state, send a final event and close.
	if room.Status == models.RoomFinished || room.Status == models.RoomCancelled || room.Status == models.RoomDead {
		raw, _ := json.Marshal(map[string]any{
			"type":      "game_over",
			"status":    string(room.Status),
			"game_over": true,
			"room_id":   roomID,
		})
		data := enrichEvent(raw, 0, string(room.Status))
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	// Send initial state event immediately.
	if engOK {
		var gs models.GameState
		if err := h.db.Where("room_id = ?", roomID).Order("turn DESC").First(&gs).Error; err == nil {
			data := buildPlayerEvent(&gs, eng, string(room.Status))
			data = enrichEvent(data, gs.Turn, string(room.Status))
			fmt.Fprintf(w, "id: %d\ndata: %s\n\n", gs.Turn, data)
			flusher.Flush()
		}
	}

	ch := h.hub.Subscribe(uint(roomID))
	defer h.hub.Unsubscribe(uint(roomID), ch)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	var turn uint
	for {
		select {
		case <-r.Context().Done():
			log.Printf("[play] agent %d disconnected from room %d", agent.ID, roomID)
			// Record disconnect time
			now := time.Now()
			h.db.Model(&models.RoomAgent{}).
				Where("room_id = ? AND agent_id = ?", roomID, agent.ID).
				Updates(map[string]any{
					"disconnected_at": now,
					"status":          string(models.RoomAgentDisconnected),
				})
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		case msg, open := <-ch:
			if !open {
				turn++
				raw, _ := json.Marshal(map[string]any{"type": "room_closed", "game_over": true})
				data := enrichEvent(raw, turn, "dead")
				fmt.Fprintf(w, "id: %d\ndata: %s\n\n", turn, data)
				flusher.Flush()
				return
			}
			turn++

			// Re-query the latest state and build a player-specific view.
			if engOK {
				var gs models.GameState
				if err := h.db.Where("room_id = ?", roomID).Order("turn DESC").First(&gs).Error; err == nil {
					// Reload room to get fresh status.
					var freshRoom models.Room
					if err := h.db.Preload("Agents.Agent").First(&freshRoom, roomID).Error; err == nil {
						room = freshRoom
					}
					data := buildPlayerEvent(&gs, eng, string(room.Status))
					data = enrichEvent(data, gs.Turn, string(room.Status))
					fmt.Fprintf(w, "id: %d\ndata: %s\n\n", gs.Turn, data)
					flusher.Flush()
					continue
				}
			}

			// Fallback: forward the raw broadcast with enrichment.
			data := enrichEvent(msg, turn, "playing")
			fmt.Fprintf(w, "id: %d\ndata: %s\n\n", turn, data)
			flusher.Flush()
		}
	}
}

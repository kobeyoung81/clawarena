package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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

// isVisibleToPlayer checks if an event's visibility allows a specific agent to see it.
func isVisibleToPlayer(visibility string, agentID uint, room *models.Room) bool {
	if visibility == "public" || visibility == "" {
		return true
	}
	// Check player-specific visibility: "player:<agentID>"
	if strings.HasPrefix(visibility, "player:") {
		visID := strings.TrimPrefix(visibility, "player:")
		return visID == strconv.FormatUint(uint64(agentID), 10)
	}
	// Check team-specific visibility: "team:<teamName>"
	// Would need team membership lookup - for now, allow if it matches
	if strings.HasPrefix(visibility, "team:") {
		// Team visibility is allowed through - the engine controls what's in state_after
		return true
	}
	return false
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

	eng := game.GetEngine(room.GameType.Name)

	// buildPlayerEvent builds a player-specific SSE payload from a stored event record.
	buildPlayerEvent := func(rec *game.BaseGameEvent, eng game.GameEngine, agentID uint, roomID uint, gameType string, agents []dto.RoomAgentInfo, status string) dto.SSEEventPayload {
		evt := rec.ToGameEvent()

		stateView := evt.StateAfter
		if eng != nil {
			if pv, err := eng.GetPlayerView(evt.StateAfter, agentID); err == nil {
				stateView = pv
			}
		}

		var pendingDTO *dto.PendingActionDTO
		var currentAgentID *uint
		if eng != nil {
			if pending, err := eng.GetPendingActions(evt.StateAfter); err == nil && len(pending) > 0 {
				// Set current agent ID to first pending
				id := pending[0].PlayerID
				currentAgentID = &id
				// Find this player's pending action specifically
				for _, pa := range pending {
					if pa.PlayerID == agentID {
						pendingDTO = &dto.PendingActionDTO{
							PlayerID:     pa.PlayerID,
							ActionType:   pa.ActionType,
							Prompt:       pa.Prompt,
							ValidTargets: pa.ValidTargets,
						}
						break
					}
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

	agents := roomAgentsInfo(room.Agents)

	// Send catch-up events for the current game
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
			// Filter by player visibility
			if !isVisibleToPlayer(evt.Visibility, agent.ID, &room) {
				continue
			}

			status := string(room.Status)
			if evt.GameOver {
				status = "intermission"
			}
			payload := buildPlayerEvent(&rec, eng, agent.ID, uint(roomID), room.GameType.Name, agents, status)
			data, _ := json.Marshal(payload)
			sseType := "game_event"
			if evt.GameOver {
				sseType = "game_over"
			}
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", rec.Seq, sseType, data)
			lastSeq = rec.Seq
		}
		flusher.Flush()
	}

	// If the room is in a terminal state, send a final event and close.
	if room.Status == models.RoomClosed {
		data, _ := json.Marshal(map[string]any{
			"room_id":   roomID,
			"game_over": true,
			"status":    string(room.Status),
		})
		fmt.Fprintf(w, "event: game_over\ndata: %s\n\n", data)
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
			log.Printf("[play] agent %d disconnected from room %d", agent.ID, roomID)
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
				lastSeq++
				data, _ := json.Marshal(map[string]any{
					"room_id":    roomID,
					"game_over":  true,
					"event_type": "room_closed",
					"status":     "closed",
				})
				fmt.Fprintf(w, "id: %d\nevent: game_over\ndata: %s\n\n", lastSeq, data)
				flusher.Flush()
				return
			}

			// Parse the broadcast payload
			var broadcast dto.SSEEventPayload
			if err := json.Unmarshal(msg, &broadcast); err != nil {
				// Fallback: forward raw message (room-level events like player_joined, game_start)
				lastSeq++
				fmt.Fprintf(w, "id: %d\nevent: game_event\ndata: %s\n\n", lastSeq, msg)
				flusher.Flush()
				continue
			}

			// Filter by player visibility
			if broadcast.Visibility != "" && !isVisibleToPlayer(broadcast.Visibility, agent.ID, &room) {
				continue
			}

			// Replace state with player view
			if eng != nil && len(broadcast.State) > 0 {
				if pv, err := eng.GetPlayerView(broadcast.State, agent.ID); err == nil {
					broadcast.State = pv
				}
			}

			// Add player-specific pending action
			if eng != nil && len(broadcast.State) > 0 {
				// Re-parse state to get pending actions for this player
				// Use the original (non-view-filtered) state from the broadcast
				// We already replaced it, so re-derive from original msg
				var origBroadcast dto.SSEEventPayload
				if json.Unmarshal(msg, &origBroadcast) == nil {
					if pending, err := eng.GetPendingActions(origBroadcast.State); err == nil {
						broadcast.PendingAction = nil
						broadcast.CurrentAgentID = nil
						if len(pending) > 0 {
							id := pending[0].PlayerID
							broadcast.CurrentAgentID = &id
							for _, pa := range pending {
								if pa.PlayerID == agent.ID {
									broadcast.PendingAction = &dto.PendingActionDTO{
										PlayerID:     pa.PlayerID,
										ActionType:   pa.ActionType,
										Prompt:       pa.Prompt,
										ValidTargets: pa.ValidTargets,
									}
									break
								}
							}
						}
					}
				}
			}

			// Reload agents for fresh status data
			var freshAgents []models.RoomAgent
			if err := h.db.Preload("Agent").Where("room_id = ?", roomID).Find(&freshAgents).Error; err == nil {
				broadcast.Agents = roomAgentsInfo(freshAgents)
			}

			data, _ := json.Marshal(broadcast)
			sseType := "game_event"
			if broadcast.GameOver {
				sseType = "game_over"
			}
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", broadcast.Seq, sseType, data)
			flusher.Flush()
		}
	}
}

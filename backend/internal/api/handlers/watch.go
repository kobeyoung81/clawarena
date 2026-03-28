package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

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
	if err := h.db.First(&room, roomID).Error; err != nil {
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

	// Replay missed events via Last-Event-ID
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		if lastTurn, err := strconv.ParseUint(lastEventID, 10, 64); err == nil {
			var actions []models.GameAction
			h.db.Preload("Agent").Where("room_id = ? AND turn > ?", roomID, lastTurn).
				Order("turn ASC").Find(&actions)
			for _, act := range actions {
				raw, _ := json.Marshal(map[string]any{
					"turn":     act.Turn,
					"agent":    act.Agent.Name,
					"action":   json.RawMessage(act.Action),
					"replayed": true,
				})
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

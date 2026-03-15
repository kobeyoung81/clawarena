package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/clawarena/clawarena/internal/api/dto"
	"github.com/clawarena/clawarena/internal/game"
	"github.com/clawarena/clawarena/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoomHandler struct {
	db                *gorm.DB
	hub               *RoomHub
	readyCheckTimeout time.Duration
	cancelsMu         sync.Mutex
	readyCancels      map[uint]context.CancelFunc
}

func NewRoomHandler(db *gorm.DB, hub *RoomHub, readyCheckTimeout time.Duration) *RoomHandler {
	return &RoomHandler{
		db:                db,
		hub:               hub,
		readyCheckTimeout: readyCheckTimeout,
		readyCancels:      map[uint]context.CancelFunc{},
	}
}

func roomResponse(room *models.Room) dto.RoomResponse {
	agents := make([]dto.RoomAgentInfo, 0, len(room.Agents))
	for _, ra := range room.Agents {
		agents = append(agents, dto.RoomAgentInfo{
			ID:      ra.ID,
			AgentID: ra.AgentID,
			Name:    ra.Agent.Name,
			Slot:    ra.Slot,
			Score:   ra.Score,
			Ready:   ra.Ready,
		})
	}
	return dto.RoomResponse{
		ID: room.ID,
		GameType: dto.GameTypeInfo{
			ID:          room.GameType.ID,
			Name:        room.GameType.Name,
			Description: room.GameType.Description,
			MinPlayers:  room.GameType.MinPlayers,
			MaxPlayers:  room.GameType.MaxPlayers,
		},
		Status:    string(room.Status),
		Owner:     dto.OwnerInfo{ID: room.Owner.ID, Name: room.Owner.Name},
		Agents:    agents,
		CreatedAt: room.CreatedAt,
	}
}

func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	query := h.db.Preload("GameType").Preload("Owner").Preload("Agents.Agent")
	if gt := r.URL.Query().Get("game_type_id"); gt != "" {
		query = query.Where("game_type_id = ?", gt)
	}
	if st := r.URL.Query().Get("status"); st != "" {
		query = query.Where("status = ?", st)
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var rooms []models.Room
	if err := query.Order("created_at DESC").Limit(perPage).Offset(offset).Find(&rooms).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list rooms", "INTERNAL_ERROR")
		return
	}
	resp := make([]dto.RoomResponse, len(rooms))
	for i := range rooms {
		resp[i] = roomResponse(&rooms[i])
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *RoomHandler) Get(w http.ResponseWriter, r *http.Request) {
	room, ok := h.loadRoom(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, roomResponse(room))
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	agent, ok := requireAgent(w, r, h.db)
	if !ok {
		return
	}

	// Check if agent already has an active room
	if h.agentHasActiveRoom(agent.ID) {
		writeError(w, http.StatusConflict, "you are already in an active room", "ALREADY_IN_ROOM")
		return
	}

	var req dto.CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}
	if req.GameTypeID == 0 {
		writeError(w, http.StatusBadRequest, "game_type_id is required", "INVALID_REQUEST")
		return
	}
	var gt models.GameType
	if err := h.db.First(&gt, req.GameTypeID).Error; err != nil {
		writeError(w, http.StatusNotFound, "game type not found", "NOT_FOUND")
		return
	}

	var room models.Room
	err := h.db.Transaction(func(tx *gorm.DB) error {
		room = models.Room{
			GameTypeID: gt.ID,
			OwnerID:    agent.ID,
			Status:     models.RoomWaiting,
		}
		if err := tx.Create(&room).Error; err != nil {
			return err
		}
		ra := models.RoomAgent{
			RoomID:   room.ID,
			AgentID:  agent.ID,
			Slot:     0,
			JoinedAt: time.Now(),
		}
		return tx.Create(&ra).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create room", "INTERNAL_ERROR")
		return
	}

	if err := h.db.Preload("GameType").Preload("Owner").Preload("Agents.Agent").First(&room, room.ID).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load room", "INTERNAL_ERROR")
		return
	}
	writeJSON(w, http.StatusCreated, roomResponse(&room))
}

func (h *RoomHandler) Join(w http.ResponseWriter, r *http.Request) {
	agent, ok := requireAgent(w, r, h.db)
	if !ok {
		return
	}
	roomID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id", "INVALID_REQUEST")
		return
	}

	if h.agentHasActiveRoom(agent.ID) {
		writeError(w, http.StatusConflict, "you are already in an active room", "ALREADY_IN_ROOM")
		return
	}

	var resp dto.JoinRoomResponse
	err = h.db.Transaction(func(tx *gorm.DB) error {
		var room models.Room
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("GameType").Preload("Agents").
			First(&room, roomID).Error; err != nil {
			return errNotFound
		}
		if room.Status != models.RoomWaiting {
			return errRoomNotOpen
		}
		if len(room.Agents) >= int(room.GameType.MaxPlayers) {
			return errRoomFull
		}

		// Check agent not already in room
		for _, ra := range room.Agents {
			if ra.AgentID == agent.ID {
				return errAlreadyInRoom
			}
		}

		nextSlot := uint8(len(room.Agents))
		ra := models.RoomAgent{
			RoomID:   room.ID,
			AgentID:  agent.ID,
			Slot:     nextSlot,
			JoinedAt: time.Now(),
		}
		if err := tx.Create(&ra).Error; err != nil {
			return err
		}

		newCount := len(room.Agents) + 1
		if newCount >= int(room.GameType.MinPlayers) {
			deadline := time.Now().Add(h.readyCheckTimeout)
			room.Status = models.RoomReadyCheck
			room.ReadyDeadline = &deadline
			if err := tx.Save(&room).Error; err != nil {
				return err
			}
			resp = dto.JoinRoomResponse{
				Slot:     nextSlot,
				Status:   string(models.RoomReadyCheck),
				Message:  "All seats filled. Ready check started — confirm within 20s.",
				Deadline: &deadline,
			}
			// Launch ready-check goroutine
			go h.startReadyCheck(uint(roomID), deadline)
		} else {
			resp = dto.JoinRoomResponse{
				Slot:    nextSlot,
				Status:  string(models.RoomWaiting),
				Message: "Joined room.",
			}
		}
		return nil
	})

	if err != nil {
		switch err {
		case errNotFound:
			writeError(w, http.StatusNotFound, "room not found", "NOT_FOUND")
		case errRoomNotOpen:
			writeError(w, http.StatusConflict, "room is not open for joining", "ROOM_NOT_OPEN")
		case errRoomFull:
			writeError(w, http.StatusConflict, "room is full", "ROOM_FULL")
		case errAlreadyInRoom:
			writeError(w, http.StatusConflict, "you are already in this room", "ALREADY_IN_ROOM")
		default:
			writeError(w, http.StatusInternalServerError, "failed to join room", "INTERNAL_ERROR")
		}
		return
	}

	h.hub.Broadcast(uint(roomID), mustMarshal(map[string]any{"type": "player_joined", "status": resp.Status}))
	writeJSON(w, http.StatusOK, resp)
}

func (h *RoomHandler) Ready(w http.ResponseWriter, r *http.Request) {
	agent, ok := requireAgent(w, r, h.db)
	if !ok {
		return
	}
	roomID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id", "INVALID_REQUEST")
		return
	}

	var resp dto.ReadyResponse
	var shouldStart bool
	var playerIDs []uint
	var gameTypeID uint

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var room models.Room
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("GameType").Preload("Agents.Agent").
			First(&room, roomID).Error; err != nil {
			return errNotFound
		}
		if room.Status != models.RoomReadyCheck {
			return errWrongStatus
		}
		if room.ReadyDeadline != nil && time.Now().After(*room.ReadyDeadline) {
			return errDeadlinePassed
		}

		// Find agent's membership
		var mySlot *models.RoomAgent
		for i := range room.Agents {
			if room.Agents[i].AgentID == agent.ID {
				mySlot = &room.Agents[i]
				break
			}
		}
		if mySlot == nil {
			return errNotInRoom
		}

		mySlot.Ready = true
		if err := tx.Save(mySlot).Error; err != nil {
			return err
		}

		readyCount := 0
		for _, ra := range room.Agents {
			if ra.AgentID == agent.ID || ra.Ready {
				readyCount++
			}
		}

		if readyCount >= len(room.Agents) {
			shouldStart = true
			gameTypeID = room.GameTypeID
			for _, ra := range room.Agents {
				playerIDs = append(playerIDs, ra.AgentID)
			}
			room.Status = models.RoomPlaying
			if err := tx.Save(&room).Error; err != nil {
				return err
			}
			resp = dto.ReadyResponse{
				Status:  string(models.RoomPlaying),
				Message: "All players ready. Game started!",
			}
		} else {
			resp = dto.ReadyResponse{
				Status:     string(models.RoomReadyCheck),
				ReadyCount: readyCount,
				Total:      len(room.Agents),
				Deadline:   room.ReadyDeadline,
			}
		}
		return nil
	})

	if err != nil {
		switch err {
		case errNotFound:
			writeError(w, http.StatusNotFound, "room not found", "NOT_FOUND")
		case errWrongStatus:
			writeError(w, http.StatusConflict, "room is not in ready check", "WRONG_STATUS")
		case errDeadlinePassed:
			writeError(w, http.StatusConflict, "ready check deadline has passed", "DEADLINE_PASSED")
		case errNotInRoom:
			writeError(w, http.StatusForbidden, "you are not in this room", "NOT_IN_ROOM")
		default:
			writeError(w, http.StatusInternalServerError, "failed to mark ready", "INTERNAL_ERROR")
		}
		return
	}

	if shouldStart {
		h.cancelReadyCheck(uint(roomID))
		if err := h.initGame(uint(roomID), gameTypeID, playerIDs); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to start game", "INTERNAL_ERROR")
			return
		}
		h.hub.Broadcast(uint(roomID), mustMarshal(map[string]any{"type": "game_start", "status": "playing"}))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *RoomHandler) Leave(w http.ResponseWriter, r *http.Request) {
	agent, ok := requireAgent(w, r, h.db)
	if !ok {
		return
	}
	roomID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id", "INVALID_REQUEST")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var room models.Room
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("GameType").Preload("Agents.Agent").
			First(&room, roomID).Error; err != nil {
			return errNotFound
		}

		if room.Status == models.RoomFinished || room.Status == models.RoomCancelled {
			return nil // no-op
		}

		// Find membership
		var mySlot *models.RoomAgent
		var myIdx int
		for i := range room.Agents {
			if room.Agents[i].AgentID == agent.ID {
				mySlot = &room.Agents[i]
				myIdx = i
				break
			}
		}
		if mySlot == nil {
			return nil // not in room, no-op
		}

		switch room.Status {
		case models.RoomWaiting, models.RoomReadyCheck:
			if err := tx.Delete(mySlot).Error; err != nil {
				return err
			}
			remaining := append(room.Agents[:myIdx], room.Agents[myIdx+1:]...)
			if len(remaining) == 0 {
				room.Status = models.RoomCancelled
				if err := tx.Save(&room).Error; err != nil {
					return err
				}
				h.hub.CloseRoom(uint(roomID))
			} else {
				if room.OwnerID == agent.ID {
					// Transfer ownership to first remaining
					room.OwnerID = remaining[0].AgentID
				}
				if room.Status == models.RoomReadyCheck {
					room.Status = models.RoomWaiting
					room.ReadyDeadline = nil
					// Reset all ready flags
					for _, ra := range remaining {
						ra.Ready = false
						tx.Save(&ra)
					}
				}
				if err := tx.Save(&room).Error; err != nil {
					return err
				}
				h.cancelReadyCheck(uint(roomID))
			}

		case models.RoomPlaying:
			if err := tx.Delete(mySlot).Error; err != nil {
				return err
			}
			remaining := append(room.Agents[:myIdx], room.Agents[myIdx+1:]...)
			if len(remaining) == 0 {
				room.Status = models.RoomCancelled
				tx.Save(&room)
				h.hub.CloseRoom(uint(roomID))
			} else if len(room.Agents) == 2 {
				// 1v1: remaining player wins
				winnerID := remaining[0].AgentID
				room.Status = models.RoomFinished
				room.WinnerID = &winnerID
				if err := tx.Save(&room).Error; err != nil {
					return err
				}
				updateElo(tx, []uint{winnerID}, []uint{agent.ID})
				h.hub.Broadcast(uint(roomID), mustMarshal(map[string]any{
					"type":      "game_over",
					"winner_id": winnerID,
					"reason":    "opponent_left",
				}))
			}
			// Multi-player: handled by game engine (not implemented here for brevity)
		}
		return nil
	})

	if err != nil && err != errNotFound {
		writeError(w, http.StatusInternalServerError, "failed to leave room", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, dto.LeaveResponse{Message: "Left room."})
}

// startReadyCheck waits for the deadline and evicts unready agents.
func (h *RoomHandler) startReadyCheck(roomID uint, deadline time.Time) {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	h.cancelsMu.Lock()
	h.readyCancels[roomID] = cancel
	h.cancelsMu.Unlock()
	defer func() {
		cancel()
		h.cancelsMu.Lock()
		delete(h.readyCancels, roomID)
		h.cancelsMu.Unlock()
	}()

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		h.evictUnready(roomID)
	}
}

func (h *RoomHandler) cancelReadyCheck(roomID uint) {
	h.cancelsMu.Lock()
	if cancel, ok := h.readyCancels[roomID]; ok {
		cancel()
	}
	h.cancelsMu.Unlock()
}

func (h *RoomHandler) evictUnready(roomID uint) {
	h.db.Transaction(func(tx *gorm.DB) error {
		var room models.Room
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Agents").First(&room, roomID).Error; err != nil {
			return err
		}
		if room.Status != models.RoomReadyCheck {
			return nil
		}
		var remaining []models.RoomAgent
		for _, ra := range room.Agents {
			if ra.Ready {
				remaining = append(remaining, ra)
			} else {
				tx.Delete(&ra)
			}
		}
		if len(remaining) == 0 {
			room.Status = models.RoomCancelled
		} else {
			room.Status = models.RoomWaiting
			room.ReadyDeadline = nil
		}
		return tx.Save(&room).Error
	})
}

func (h *RoomHandler) initGame(roomID, gameTypeID uint, playerIDs []uint) error {
	var gt models.GameType
	if err := h.db.First(&gt, gameTypeID).Error; err != nil {
		return err
	}
	eng, ok := game.Registry[gt.Name]
	if !ok {
		return nil
	}
	stateRaw, err := eng.InitState(json.RawMessage(gt.Config), playerIDs)
	if err != nil {
		return err
	}
	gs := models.GameState{
		RoomID:    roomID,
		Turn:      0,
		State:     datatypes.JSON(stateRaw),
		CreatedAt: time.Now(),
	}
	return h.db.Create(&gs).Error
}

func (h *RoomHandler) agentHasActiveRoom(agentID uint) bool {
	var count int64
	h.db.Model(&models.RoomAgent{}).
		Joins("JOIN rooms ON rooms.id = room_agents.room_id").
		Where("room_agents.agent_id = ? AND rooms.status IN ?", agentID,
			[]string{string(models.RoomWaiting), string(models.RoomReadyCheck), string(models.RoomPlaying)}).
		Count(&count)
	return count > 0
}

func (h *RoomHandler) loadRoom(w http.ResponseWriter, r *http.Request) (*models.Room, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id", "INVALID_REQUEST")
		return nil, false
	}
	var room models.Room
	if err := h.db.Preload("GameType").Preload("Owner").Preload("Agents.Agent").First(&room, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "room not found", "NOT_FOUND")
		return nil, false
	}
	return &room, true
}

// sentinel errors
var (
	errNotFound       = &appError{"not found"}
	errRoomNotOpen    = &appError{"room not open"}
	errRoomFull       = &appError{"room full"}
	errAlreadyInRoom  = &appError{"already in room"}
	errWrongStatus    = &appError{"wrong status"}
	errDeadlinePassed = &appError{"deadline passed"}
	errNotInRoom      = &appError{"not in room"}
)

type appError struct{ msg string }

func (e *appError) Error() string { return e.msg }

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

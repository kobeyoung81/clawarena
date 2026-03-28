package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clawarena/clawarena/internal/api/dto"
	"github.com/clawarena/clawarena/internal/game"
	"github.com/clawarena/clawarena/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type GameHistoryHandler struct {
	db *gorm.DB
}

func NewGameHistoryHandler(db *gorm.DB) *GameHistoryHandler {
	return &GameHistoryHandler{db: db}
}

// ListGames returns a paginated list of games, with fallback to rooms for pre-migration data.
func (h *GameHistoryHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "finished"
	}
	gameTypeID := r.URL.Query().Get("game_type_id")
	roomIDFilter := r.URL.Query().Get("room_id")

	// Try querying games table first
	var totalCount int64
	q := h.db.Model(&models.Game{}).Where("status = ?", status)
	if gameTypeID != "" {
		q = q.Where("game_type_id = ?", gameTypeID)
	}
	if roomIDFilter != "" {
		q = q.Where("room_id = ?", roomIDFilter)
	}
	q.Count(&totalCount)

	if totalCount > 0 {
		var games []models.Game
		offset := (page - 1) * perPage
		qFetch := h.db.Preload("GameType").Preload("Players.Agent").
			Where("status = ?", status)
		if gameTypeID != "" {
			qFetch = qFetch.Where("game_type_id = ?", gameTypeID)
		}
		if roomIDFilter != "" {
			qFetch = qFetch.Where("room_id = ?", roomIDFilter)
		}
		if err := qFetch.Order("started_at DESC").Offset(offset).Limit(perPage).Find(&games).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load games", "INTERNAL_ERROR")
			return
		}

		items := make([]dto.GameListItem, len(games))
		for i, g := range games {
			items[i] = gameToListItem(g)
		}

		writeJSON(w, http.StatusOK, dto.GameListResponse{
			Games:      items,
			TotalCount: totalCount,
			Page:       page,
			PerPage:    perPage,
		})
		return
	}

	// Fallback: query rooms with matching status for pre-migration data
	roomStatus := models.RoomClosed
	if status == "playing" {
		roomStatus = models.RoomPlaying
	} else if status == "aborted" {
		roomStatus = models.RoomClosed
	}

	var roomCount int64
	rq := h.db.Model(&models.Room{}).Where("status = ?", roomStatus)
	if gameTypeID != "" {
		rq = rq.Where("game_type_id = ?", gameTypeID)
	}
	if roomIDFilter != "" {
		rq = rq.Where("id = ?", roomIDFilter)
	}
	rq.Count(&roomCount)

	var rooms []models.Room
	offset := (page - 1) * perPage
	rqFetch := h.db.Preload("GameType").Preload("Agents.Agent").
		Where("status = ?", roomStatus)
	if gameTypeID != "" {
		rqFetch = rqFetch.Where("game_type_id = ?", gameTypeID)
	}
	if roomIDFilter != "" {
		rqFetch = rqFetch.Where("id = ?", roomIDFilter)
	}
	if err := rqFetch.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&rooms).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load rooms", "INTERNAL_ERROR")
		return
	}

	items := make([]dto.GameListItem, len(rooms))
	for i, rm := range rooms {
		items[i] = roomToGameListItem(rm)
	}

	writeJSON(w, http.StatusOK, dto.GameListResponse{
		Games:      items,
		TotalCount: roomCount,
		Page:       page,
		PerPage:    perPage,
	})
}

// GetGameHistory returns the full history for a single game.
func (h *GameHistoryHandler) GetGameHistory(w http.ResponseWriter, r *http.Request) {
	gameID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game id", "INVALID_REQUEST")
		return
	}

	var g models.Game
	if err := h.db.Preload("GameType").Preload("Players.Agent").First(&g, gameID).Error; err != nil {
		writeError(w, http.StatusNotFound, "game not found", "NOT_FOUND")
		return
	}

	// Try loading states/actions by game_id first, fall back to room_id
	var states []models.GameState
	if err := h.db.Where("game_id = ?", g.ID).Order("turn ASC").Find(&states).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load states", "INTERNAL_ERROR")
		return
	}
	var actions []models.GameAction
	if len(states) > 0 {
		if err := h.db.Preload("Agent").Where("game_id = ?", g.ID).Order("turn ASC").Find(&actions).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load actions", "INTERNAL_ERROR")
			return
		}
	} else {
		// Fallback to room_id
		if err := h.db.Where("room_id = ?", g.RoomID).Order("turn ASC").Find(&states).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load states", "INTERNAL_ERROR")
			return
		}
		if err := h.db.Preload("Agent").Where("room_id = ?", g.RoomID).Order("turn ASC").Find(&actions).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load actions", "INTERNAL_ERROR")
			return
		}
	}

	eng, hasEngine := game.Registry[g.GameType.Name]

	actionByTurn := map[uint]*models.GameAction{}
	for i := range actions {
		actionByTurn[actions[i].Turn] = &actions[i]
	}

	timeline := make([]dto.HistoryEntry, len(states))
	for i, gs := range states {
		var stateView json.RawMessage
		if hasEngine {
			stateView, _ = eng.GetSpectatorView(json.RawMessage(gs.State))
		} else {
			stateView = json.RawMessage(gs.State)
		}

		entry := dto.HistoryEntry{
			Turn:      gs.Turn,
			State:     stateView,
			Events:    []dto.GameEventDTO{},
			CreatedAt: gs.CreatedAt,
		}
		if act, ok := actionByTurn[gs.Turn]; ok {
			entry.AgentID = &act.AgentID
			// Filter out private actions (CW night phases) for spectator consistency
			if !isPrivateAction(act.Action) {
				entry.Action = json.RawMessage(act.Action)
			}
		}
		timeline[i] = entry
	}

	players := make([]dto.HistoryPlayer, len(g.Players))
	for i, gp := range g.Players {
		slot := gp.Slot
		players[i] = dto.HistoryPlayer{
			Slot:    &slot,
			AgentID: gp.AgentID,
			Name:    gp.Agent.Name,
		}
	}

	var resultDTO *dto.GameResultDTO
	if g.Result != nil {
		var gr game.GameResult
		if err := json.Unmarshal(g.Result, &gr); err == nil {
			resultDTO = &dto.GameResultDTO{
				WinnerIDs:  gr.WinnerIDs,
				WinnerTeam: gr.WinnerTeam,
			}
		}
	}

	writeJSON(w, http.StatusOK, dto.HistoryResponse{
		RoomID:   g.RoomID,
		Status:   string(g.Status),
		GameType: g.GameType.Name,
		Result:   resultDTO,
		Players:  players,
		Timeline: timeline,
	})
}

func gameToListItem(g models.Game) dto.GameListItem {
	item := dto.GameListItem{
		ID:     g.ID,
		RoomID: g.RoomID,
		GameType: dto.GameTypeInfo{
			ID:          g.GameType.ID,
			Name:        g.GameType.Name,
			Description: g.GameType.Description,
			MinPlayers:  g.GameType.MinPlayers,
			MaxPlayers:  g.GameType.MaxPlayers,
		},
		Status:     string(g.Status),
		WinnerID:   g.WinnerID,
		StartedAt:  g.StartedAt,
		FinishedAt: g.FinishedAt,
	}

	if g.Result != nil {
		var gr game.GameResult
		if err := json.Unmarshal(g.Result, &gr); err == nil {
			item.Result = &dto.GameResultDTO{
				WinnerIDs:  gr.WinnerIDs,
				WinnerTeam: gr.WinnerTeam,
			}
		}
	}

	for _, gp := range g.Players {
		item.Players = append(item.Players, dto.GamePlayerInfo{
			AgentID: gp.AgentID,
			Name:    gp.Agent.Name,
			Slot:    gp.Slot,
		})
	}
	if item.Players == nil {
		item.Players = []dto.GamePlayerInfo{}
	}

	return item
}

func roomToGameListItem(rm models.Room) dto.GameListItem {
	item := dto.GameListItem{
		ID:     rm.ID,
		RoomID: rm.ID,
		GameType: dto.GameTypeInfo{
			ID:          rm.GameType.ID,
			Name:        rm.GameType.Name,
			Description: rm.GameType.Description,
			MinPlayers:  rm.GameType.MinPlayers,
			MaxPlayers:  rm.GameType.MaxPlayers,
		},
		Status:    string(rm.Status),
		WinnerID:  rm.WinnerID,
		StartedAt: rm.CreatedAt,
	}

	if rm.Result != nil {
		var gr game.GameResult
		if err := json.Unmarshal(rm.Result, &gr); err == nil {
			item.Result = &dto.GameResultDTO{
				WinnerIDs:  gr.WinnerIDs,
				WinnerTeam: gr.WinnerTeam,
			}
		}
	}

	for _, ra := range rm.Agents {
		item.Players = append(item.Players, dto.GamePlayerInfo{
			AgentID: ra.AgentID,
			Name:    ra.Agent.Name,
			Slot:    ra.Slot,
		})
	}
	if item.Players == nil {
		item.Players = []dto.GamePlayerInfo{}
	}

	return item
}

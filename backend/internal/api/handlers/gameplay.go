package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/clawarena/clawarena/internal/api/dto"
	"github.com/clawarena/clawarena/internal/game"
	"github.com/clawarena/clawarena/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GameplayHandler struct {
	db         *gorm.DB
	hub        *RoomHub
	eloKFactor float64
}

func NewGameplayHandler(db *gorm.DB, hub *RoomHub, eloKFactor float64) *GameplayHandler {
	return &GameplayHandler{db: db, hub: hub, eloKFactor: eloKFactor}
}

// loadLatestState loads the latest game state from the per-game event table.
func loadLatestState(db *gorm.DB, eng game.GameEngine, gameID uint) (json.RawMessage, uint, error) {
	tableName := eng.NewEventModel().TableName()
	var result struct {
		Seq        uint
		StateAfter datatypes.JSON
	}
	err := db.Table(tableName).
		Select("seq, state_after").
		Where("game_id = ?", gameID).
		Order("seq DESC").
		Limit(1).
		Scan(&result).Error
	if err != nil {
		return nil, 0, err
	}
	return json.RawMessage(result.StateAfter), result.Seq, nil
}

func (h *GameplayHandler) SubmitAction(w http.ResponseWriter, r *http.Request) {
	agent, ok := requireAgent(w, r, h.db)
	if !ok {
		return
	}
	roomID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id", "INVALID_REQUEST")
		return
	}

	var req dto.SubmitActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	var applyResult *game.ApplyResult
	var lastSeq uint
	var gameID uint
	var gameType string
	var roomSnapshot models.Room

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var room models.Room
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("GameType").Preload("Agents.Agent").
			First(&room, roomID).Error; err != nil {
			return errNotFound
		}
		if room.Status != models.RoomPlaying {
			return errWrongStatus
		}
		if room.CurrentGameID == nil {
			return errWrongStatus
		}
		gameType = room.GameType.Name
		gameID = *room.CurrentGameID

		// Check agent is in room
		inRoom := false
		for _, ra := range room.Agents {
			if ra.AgentID == agent.ID {
				inRoom = true
				break
			}
		}
		if !inRoom {
			return errNotInRoom
		}

		eng := game.GetEngine(room.GameType.Name)
		if eng == nil {
			return &appError{"game engine not found"}
		}

		// Load latest state from per-game event table
		latestState, seq, err := loadLatestState(tx, eng, gameID)
		if err != nil {
			return err
		}
		lastSeq = seq

		// Check it's the agent's turn
		pending, err := eng.GetPendingActions(latestState)
		if err != nil {
			return err
		}
		hasTurn := false
		for _, pa := range pending {
			if pa.PlayerID == agent.ID {
				hasTurn = true
				break
			}
		}
		if !hasTurn {
			return &appError{"not your turn"}
		}

		result, err := eng.ApplyAction(latestState, agent.ID, req.Action)
		if err != nil {
			return &appError{err.Error()}
		}
		applyResult = result

		// Store each event in per-game event table
		tableName := eng.NewEventModel().TableName()
		now := time.Now()
		for i, evt := range result.Events {
			record := game.BaseGameEvent{}
			record.SetFields(gameID, lastSeq+uint(i)+1, evt, now)
			if err := tx.Table(tableName).Create(&record).Error; err != nil {
				return err
			}
		}

		// Check if game is over (any event with GameOver: true)
		var gameOverResult *game.GameResult
		for _, evt := range result.Events {
			if evt.GameOver {
				gameOverResult = evt.Result
				break
			}
		}

		if gameOverResult != nil {
			nowTime := time.Now()
			room.Status = models.RoomIntermission
			if gameOverResult != nil && len(gameOverResult.WinnerIDs) > 0 {
				room.WinnerID = &gameOverResult.WinnerIDs[0]
			}
			resultJSON, _ := json.Marshal(gameOverResult)
			room.Result = resultJSON
			if err := tx.Save(&room).Error; err != nil {
				return err
			}

			// Update the Game record
			gameUpdates := map[string]any{
				"status":      string(models.GameFinished),
				"finished_at": nowTime,
			}
			if gameOverResult != nil {
				gameResultJSON, _ := json.Marshal(gameOverResult)
				gameUpdates["result"] = datatypes.JSON(gameResultJSON)
				if len(gameOverResult.WinnerIDs) > 0 {
					gameUpdates["winner_id"] = gameOverResult.WinnerIDs[0]
				}
			}
			tx.Model(&models.Game{}).Where("id = ?", gameID).Updates(gameUpdates)

			// Reset all agents' ready flags for next game
			tx.Model(&models.RoomAgent{}).Where("room_id = ?", room.ID).
				Updates(map[string]any{"ready": false})

			if gameOverResult != nil {
				var loserIDs []uint
				winSet := map[uint]bool{}
				for _, id := range gameOverResult.WinnerIDs {
					winSet[id] = true
				}
				for _, ra := range room.Agents {
					if !winSet[ra.AgentID] {
						loserIDs = append(loserIDs, ra.AgentID)
					}
				}
				updateElo(tx, gameOverResult.WinnerIDs, loserIDs, h.eloKFactor)
			}
		}
		roomSnapshot = room
		return nil
	})

	if err != nil {
		switch e := err.(type) {
		case *appError:
			if e.msg == "not your turn" {
				writeError(w, http.StatusBadRequest, e.msg, "NOT_YOUR_TURN")
			} else if e.msg == "game is already over" {
				writeError(w, http.StatusBadRequest, e.msg, "GAME_OVER")
			} else {
				writeError(w, http.StatusBadRequest, e.msg, "INVALID_ACTION")
			}
		default:
			if err == errNotFound {
				writeError(w, http.StatusNotFound, "room not found", "NOT_FOUND")
			} else if err == errWrongStatus {
				writeError(w, http.StatusBadRequest, "game is not active", "GAME_OVER")
			} else if err == errNotInRoom {
				writeError(w, http.StatusForbidden, "you are not in this room", "NOT_IN_ROOM")
			} else {
				writeError(w, http.StatusInternalServerError, "failed to apply action", "INTERNAL_ERROR")
			}
		}
		return
	}

	eng := game.GetEngine(gameType)
	agents := roomAgentsInfo(roomSnapshot.Agents)

	// Build response events and broadcast each event individually
	respEvents := make([]dto.GameEventDTO, 0, len(applyResult.Events))
	var finalGameOver bool
	var finalResult *dto.GameResultDTO

	now := time.Now()
	for i, evt := range applyResult.Events {
		seq := lastSeq + uint(i) + 1

		// Build spectator view for state
		stateView := evt.StateAfter
		if eng != nil {
			if sv, err := eng.GetSpectatorView(evt.StateAfter); err == nil {
				stateView = sv
			}
		}

		// Build pending action from state
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

		var evtResultDTO *dto.GameResultDTO
		if evt.Result != nil {
			evtResultDTO = &dto.GameResultDTO{
				WinnerIDs:  evt.Result.WinnerIDs,
				WinnerTeam: evt.Result.WinnerTeam,
				Scores:     evt.Result.Scores,
			}
		}

		// Build the SSE broadcast payload (raw state, not view-filtered)
		broadcast := dto.SSEEventPayload{
			Seq:            seq,
			GameID:         gameID,
			RoomID:         uint(roomID),
			Source:         evt.Source,
			EventType:      evt.EventType,
			Actor:          evt.Actor,
			Target:         evt.Target,
			Details:        evt.Details,
			State:          evt.StateAfter, // raw state; SSE handlers will filter
			Visibility:     evt.Visibility,
			PendingAction:  pendingDTO,
			CurrentAgentID: currentAgentID,
			Agents:         agents,
			GameType:       gameType,
			GameOver:       evt.GameOver,
			Result:         evtResultDTO,
		}
		if evt.GameOver {
			broadcast.Status = "intermission"
			broadcast.Message = "POST /ready to play again or /leave to exit"
		}
		h.hub.Broadcast(uint(roomID), mustMarshal(broadcast))

		// Build response event (with spectator view)
		respEvent := dto.GameEventDTO{
			Seq:        seq,
			GameID:     gameID,
			Source:     evt.Source,
			EventType:  evt.EventType,
			Actor:      evt.Actor,
			Target:     evt.Target,
			Details:    evt.Details,
			State:      stateView,
			Visibility: evt.Visibility,
			GameOver:   evt.GameOver,
			Result:     evtResultDTO,
			CreatedAt:  now,
		}
		respEvents = append(respEvents, respEvent)

		if evt.GameOver {
			finalGameOver = true
			finalResult = evtResultDTO
		}
	}

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Events:   respEvents,
		GameOver: finalGameOver,
		Result:   finalResult,
	})
}

func (h *GameplayHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
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

	eng := game.GetEngine(room.GameType.Name)
	if eng == nil || room.CurrentGameID == nil {
		writeJSON(w, http.StatusOK, dto.EventHistoryResponse{
			RoomID:   room.ID,
			Status:   string(room.Status),
			GameType: room.GameType.Name,
			Events:   []dto.GameEventDTO{},
			Players:  []dto.HistoryPlayer{},
		})
		return
	}

	gameID := *room.CurrentGameID
	tableName := eng.NewEventModel().TableName()

	// Query all events for the current game
	var records []game.BaseGameEvent
	if err := h.db.Table(tableName).
		Where("game_id = ?", gameID).
		Order("seq ASC").
		Find(&records).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load events", "INTERNAL_ERROR")
		return
	}

	events := make([]dto.GameEventDTO, 0, len(records))
	for _, rec := range records {
		evt := rec.ToGameEvent()
		// Filter out non-public visibility events for spectators
		if evt.Visibility != "public" {
			continue
		}

		stateView := evt.StateAfter
		if sv, err := eng.GetSpectatorView(evt.StateAfter); err == nil {
			stateView = sv
		}

		var resultDTO *dto.GameResultDTO
		if evt.Result != nil {
			resultDTO = &dto.GameResultDTO{
				WinnerIDs:  evt.Result.WinnerIDs,
				WinnerTeam: evt.Result.WinnerTeam,
				Scores:     evt.Result.Scores,
			}
		}

		events = append(events, dto.GameEventDTO{
			Seq:        rec.Seq,
			GameID:     rec.GameID,
			Source:     evt.Source,
			EventType:  evt.EventType,
			Actor:      evt.Actor,
			Target:     evt.Target,
			Details:    evt.Details,
			State:      stateView,
			Visibility: evt.Visibility,
			GameOver:   evt.GameOver,
			Result:     resultDTO,
			CreatedAt:  rec.CreatedAt,
		})
	}

	// Build players list
	var players []dto.HistoryPlayer
	var gamePlayers []models.GamePlayer
	h.db.Preload("Agent").Where("game_id = ?", gameID).Find(&gamePlayers)
	for _, gp := range gamePlayers {
		slot := gp.Slot
		players = append(players, dto.HistoryPlayer{Slot: &slot, AgentID: gp.AgentID, Name: gp.Agent.Name})
	}
	if len(players) == 0 {
		for _, ra := range room.Agents {
			slot := ra.Slot
			players = append(players, dto.HistoryPlayer{Slot: &slot, AgentID: ra.AgentID, Name: ra.Agent.Name})
		}
	}

	var resultDTO *dto.GameResultDTO
	if room.Result != nil {
		var gr game.GameResult
		if err := json.Unmarshal(room.Result, &gr); err == nil {
			resultDTO = &dto.GameResultDTO{
				WinnerIDs:  gr.WinnerIDs,
				WinnerTeam: gr.WinnerTeam,
			}
		}
	}

	writeJSON(w, http.StatusOK, dto.EventHistoryResponse{
		RoomID:   room.ID,
		GameID:   gameID,
		Status:   string(room.Status),
		GameType: room.GameType.Name,
		Result:   resultDTO,
		Players:  players,
		Events:   events,
	})
}

func roomAgentsInfo(agents []models.RoomAgent) []dto.RoomAgentInfo {
	result := make([]dto.RoomAgentInfo, len(agents))
	for i, ra := range agents {
		result[i] = dto.RoomAgentInfo{
			ID:      ra.ID,
			AgentID: ra.AgentID,
			Name:    ra.Agent.Name,
			Slot:    ra.Slot,
			Score:   ra.Score,
			Ready:   ra.Ready,
			Status:  string(ra.Status),
		}
	}
	return result
}

// updateElo updates Elo ratings for winners and losers using the given K-factor.
func updateElo(db *gorm.DB, winnerIDs, loserIDs []uint, K float64) {
	if len(winnerIDs) == 0 || len(loserIDs) == 0 {
		return
	}

	var winners, losers []models.Agent
	db.Find(&winners, winnerIDs)
	db.Find(&losers, loserIDs)

	// Average rating of each side
	avgWinner := avgRating(winners)
	avgLoser := avgRating(losers)

	eWinner := 1.0 / (1.0 + math.Pow(10, (avgLoser-avgWinner)/400))
	eLoss := 1.0 / (1.0 + math.Pow(10, (avgWinner-avgLoser)/400))

	for i := range winners {
		delta := int(math.Round(K * (1.0 - eWinner)))
		db.Model(&winners[i]).Update("elo_rating", winners[i].EloRating+delta)
	}
	for i := range losers {
		delta := int(math.Round(K * (0.0 - eLoss)))
		db.Model(&losers[i]).Update("elo_rating", losers[i].EloRating+delta)
	}
}

func avgRating(agents []models.Agent) float64 {
	if len(agents) == 0 {
		return 1000
	}
	sum := 0
	for _, a := range agents {
		sum += a.EloRating
	}
	return float64(sum) / float64(len(agents))
}

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

	var actionResult game.ActionResult
	var newTurn uint
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
		gameType = room.GameType.Name

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

		var gs models.GameState
		q := tx.Where("room_id = ?", roomID)
		if room.CurrentGameID != nil {
			q = q.Where("game_id = ?", *room.CurrentGameID)
		}
		if err := q.Order("turn DESC").First(&gs).Error; err != nil {
			return err
		}

		eng, ok := game.Registry[room.GameType.Name]
		if !ok {
			return &appError{"game engine not found"}
		}

		// Check it's the agent's turn
		pending, err := eng.GetPendingActions(json.RawMessage(gs.State))
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

		result, err := eng.ApplyAction(json.RawMessage(gs.State), agent.ID, req.Action)
		if err != nil {
			return &appError{err.Error()}
		}
		actionResult = result

		newTurn = gs.Turn + 1
		newState := models.GameState{
			RoomID:    room.ID,
			GameID:    room.CurrentGameID,
			Turn:      newTurn,
			State:     datatypes.JSON(result.NewState),
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&newState).Error; err != nil {
			return err
		}
		gameAction := models.GameAction{
			RoomID:    room.ID,
			GameID:    room.CurrentGameID,
			AgentID:   agent.ID,
			Turn:      newTurn,
			Action:    datatypes.JSON(req.Action),
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&gameAction).Error; err != nil {
			return err
		}

		if result.GameOver {
			now := time.Now()
			room.Status = models.RoomIntermission
			if result.Result != nil && len(result.Result.WinnerIDs) > 0 {
				room.WinnerID = &result.Result.WinnerIDs[0]
			}
			resultJSON, _ := json.Marshal(result.Result)
			room.Result = resultJSON
			if err := tx.Save(&room).Error; err != nil {
				return err
			}

			// Update the Game record
			if room.CurrentGameID != nil {
				gameUpdates := map[string]any{
					"status":      string(models.GameFinished),
					"finished_at": now,
				}
				if result.Result != nil {
					gameResultJSON, _ := json.Marshal(result.Result)
					gameUpdates["result"] = datatypes.JSON(gameResultJSON)
					if len(result.Result.WinnerIDs) > 0 {
						gameUpdates["winner_id"] = result.Result.WinnerIDs[0]
					}
				}
				tx.Model(&models.Game{}).Where("id = ?", *room.CurrentGameID).Updates(gameUpdates)
			}

			// Reset all agents' ready flags for next game
			tx.Model(&models.RoomAgent{}).Where("room_id = ?", room.ID).
				Updates(map[string]any{"ready": false})

			if result.Result != nil {
				var loserIDs []uint
				winSet := map[uint]bool{}
				for _, id := range result.Result.WinnerIDs {
					winSet[id] = true
				}
				for _, ra := range room.Agents {
					if !winSet[ra.AgentID] {
						loserIDs = append(loserIDs, ra.AgentID)
					}
				}
				updateElo(tx, result.Result.WinnerIDs, loserIDs, h.eloKFactor)
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

	// Broadcast SSE event
	events := make([]dto.GameEventDTO, len(actionResult.Events))
	for i, e := range actionResult.Events {
		events[i] = dto.GameEventDTO{Type: e.Type, Message: e.Message, Visibility: e.Visibility}
	}

	var resultDTO *dto.GameResultDTO
	if actionResult.Result != nil {
		resultDTO = &dto.GameResultDTO{
			WinnerIDs:  actionResult.Result.WinnerIDs,
			WinnerTeam: actionResult.Result.WinnerTeam,
		}
	}

	// Build spectator view for broadcast
	broadcastState := json.RawMessage(actionResult.NewState)
	if eng, ok := game.Registry[gameType]; ok {
		if sv, err := eng.GetSpectatorView(json.RawMessage(actionResult.NewState)); err == nil {
			broadcastState = sv
		}
	}

	// Build spectator-friendly pending action (who needs to act, not details)
	var pendingDTO *dto.PendingActionDTO
	var currentAgentID *uint
	if eng, ok := game.Registry[gameType]; ok {
		if pending, err := eng.GetPendingActions(json.RawMessage(actionResult.NewState)); err == nil && len(pending) > 0 {
			id := pending[0].PlayerID
			currentAgentID = &id
			pendingDTO = &dto.PendingActionDTO{
				PlayerID:   pending[0].PlayerID,
				ActionType: pending[0].ActionType,
				Prompt:     pending[0].Prompt,
			}
		}
	}

	// Extract phase from state if available
	var phase string
	var stateMap map[string]any
	if json.Unmarshal(broadcastState, &stateMap) == nil {
		if p, ok := stateMap["phase"].(string); ok {
			phase = p
		}
	}

	broadcast := map[string]any{
		"turn":             newTurn,
		"state":            broadcastState,
		"events":           events,
		"game_over":        actionResult.GameOver,
		"result":           resultDTO,
		"game_type":        gameType,
		"agents":           roomAgentsInfo(roomSnapshot.Agents),
		"pending_action":   pendingDTO,
		"current_agent_id": currentAgentID,
		"phase":            phase,
		"agent_id":         agent.ID,
		"action":           req.Action,
	}
	if actionResult.GameOver {
		broadcast["status"] = "post_game"
		broadcast["message"] = "POST /ready to play again or /leave to exit"
	}
	h.hub.Broadcast(uint(roomID), mustMarshal(broadcast))

	// Don't close the room hub on game over — room is reusable in post_game state

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Events:   events,
		GameOver: actionResult.GameOver,
		Result:   resultDTO,
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

	var states []models.GameState
	if err := h.db.Where("room_id = ?", roomID).Order("turn ASC").Find(&states).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load states", "INTERNAL_ERROR")
		return
	}
	var actions []models.GameAction
	if err := h.db.Preload("Agent").Where("room_id = ?", roomID).Order("turn ASC").Find(&actions).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load actions", "INTERNAL_ERROR")
		return
	}

	eng, ok := game.Registry[room.GameType.Name]

	// Build action map by turn
	actionByTurn := map[uint]*models.GameAction{}
	for i := range actions {
		actionByTurn[actions[i].Turn] = &actions[i]
	}

	timeline := make([]dto.HistoryEntry, len(states))
	for i, gs := range states {
		var stateView json.RawMessage
		if ok {
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

	// Build players list from game_players (actual participants), falling back to room agents
	var players []dto.HistoryPlayer
	if room.CurrentGameID != nil {
		var gamePlayers []models.GamePlayer
		h.db.Preload("Agent").Where("game_id = ?", *room.CurrentGameID).Find(&gamePlayers)
		for _, gp := range gamePlayers {
			slot := gp.Slot
			players = append(players, dto.HistoryPlayer{Slot: &slot, AgentID: gp.AgentID, Name: gp.Agent.Name})
		}
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

	writeJSON(w, http.StatusOK, dto.HistoryResponse{
		RoomID:   room.ID,
		Status:   string(room.Status),
		GameType: room.GameType.Name,
		Result:   resultDTO,
		Players:  players,
		Timeline: timeline,
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

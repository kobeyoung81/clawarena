package tictactoe

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clawarena/clawarena/internal/game"
)

func init() {
	game.Register("tic_tac_toe", &Engine{})
}

// State is the internal Tic-Tac-Toe game state.
type State struct {
	Board   [9]string `json:"board"`
	Players []uint    `json:"players"`
	Turn    int       `json:"turn"` // index into Players
	Winner  *uint     `json:"winner"`
	IsDraw  bool      `json:"is_draw"`
}

// PlayerView is what a player sees (same as full state for TTT).
type PlayerView struct {
	Board         [9]string           `json:"board"`
	Players       []uint              `json:"players"`
	Turn          int                 `json:"turn"`
	Winner        *uint               `json:"winner"`
	IsDraw        bool                `json:"is_draw"`
	PendingAction *game.PendingAction `json:"pending_action,omitempty"`
}

type Engine struct{}

func (e *Engine) Syncronym() string { return "ttt" }

func (e *Engine) NewEventModel() game.GameEventRecord { return &TttGameEvent{} }

func (e *Engine) GetPhaseTimeout(_ json.RawMessage) *game.PhaseTimeout { return nil }

var winLines = [][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // cols
	{0, 4, 8}, {2, 4, 6},             // diagonals
}

func parseState(raw json.RawMessage) (*State, error) {
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (e *Engine) InitState(config json.RawMessage, players []uint) (json.RawMessage, []game.GameEvent, error) {
	if len(players) != 2 {
		return nil, nil, errors.New("tic_tac_toe requires exactly 2 players")
	}
	s := State{
		Players: players,
		Turn:    0,
	}
	stateJSON, err := json.Marshal(s)
	if err != nil {
		return nil, nil, err
	}

	// Build player details for the init event
	type playerInfo struct {
		ID     uint   `json:"id"`
		Seat   int    `json:"seat"`
		Symbol string `json:"symbol"`
	}
	details := map[string]any{
		"players": []playerInfo{
			{ID: players[0], Seat: 0, Symbol: "X"},
			{ID: players[1], Seat: 1, Symbol: "O"},
		},
	}
	detailsJSON, _ := json.Marshal(details)

	events := []game.GameEvent{
		{
			Source:     "system",
			EventType:  "game_start",
			Details:    detailsJSON,
			StateAfter: stateJSON,
			Visibility: "public",
		},
	}

	return stateJSON, events, nil
}

func (e *Engine) GetPlayerView(raw json.RawMessage, playerID uint) (json.RawMessage, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	view := PlayerView{
		Board:   s.Board,
		Players: s.Players,
		Turn:    s.Turn,
		Winner:  s.Winner,
		IsDraw:  s.IsDraw,
	}
	if s.Winner == nil && !s.IsDraw && len(s.Players) > s.Turn && s.Players[s.Turn] == playerID {
		view.PendingAction = &game.PendingAction{
			PlayerID:   playerID,
			ActionType: "move",
			Prompt:     "Place your mark on an empty cell (0-8).",
		}
	}
	return json.Marshal(view)
}

func (e *Engine) GetSpectatorView(raw json.RawMessage) (json.RawMessage, error) {
	return raw, nil
}

func (e *Engine) GetGodView(raw json.RawMessage) (json.RawMessage, error) {
	return raw, nil
}

func (e *Engine) GetPendingActions(raw json.RawMessage) ([]game.PendingAction, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	if s.Winner != nil || s.IsDraw {
		return nil, nil
	}
	if s.Turn >= len(s.Players) {
		return nil, nil
	}
	return []game.PendingAction{{
		PlayerID:   s.Players[s.Turn],
		ActionType: "move",
		Prompt:     "Place your mark on an empty cell (0-8).",
	}}, nil
}

type moveAction struct {
	Position int `json:"position"`
}

func intPtr(v int) *int { return &v }

func (e *Engine) ApplyAction(raw json.RawMessage, playerID uint, actionRaw json.RawMessage) (*game.ApplyResult, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	if s.Winner != nil || s.IsDraw {
		return nil, errors.New("game is already over")
	}
	if s.Turn >= len(s.Players) || s.Players[s.Turn] != playerID {
		return nil, errors.New("not your turn")
	}

	var action moveAction
	if err := json.Unmarshal(actionRaw, &action); err != nil {
		return nil, fmt.Errorf("invalid action: %w", err)
	}
	if action.Position < 0 || action.Position > 8 {
		return nil, errors.New("position must be between 0 and 8")
	}
	if s.Board[action.Position] != "" {
		return nil, errors.New("cell already occupied")
	}

	mark := "X"
	if s.Turn == 1 {
		mark = "O"
	}
	s.Board[action.Position] = mark

	// Find the player's seat index
	seat := 0
	for i, pid := range s.Players {
		if pid == playerID {
			seat = i
			break
		}
	}

	var events []game.GameEvent

	// Build the move details
	moveDetails, _ := json.Marshal(map[string]any{
		"position": action.Position,
		"symbol":   mark,
	})

	// Check win
	if winner := checkWin(s.Board, mark); winner {
		s.Winner = &playerID
		stateJSON, _ := json.Marshal(s)

		// Agent move event
		events = append(events, game.GameEvent{
			Source:     "agent",
			EventType:  "move",
			Actor:     &game.EventEntity{AgentID: &playerID, Seat: intPtr(seat)},
			Details:    moveDetails,
			StateAfter: stateJSON,
			Visibility: "public",
		})

		// Find winning line for details
		var winningLine []int
		for _, line := range winLines {
			if s.Board[line[0]] == mark && s.Board[line[1]] == mark && s.Board[line[2]] == mark {
				winningLine = []int{line[0], line[1], line[2]}
				break
			}
		}
		gameOverDetails, _ := json.Marshal(map[string]any{
			"winner_ids":   []uint{playerID},
			"winner_symbol": mark,
			"winning_line": winningLine,
		})

		// System game_over event
		events = append(events, game.GameEvent{
			Source:     "system",
			EventType:  "game_over",
			Details:    gameOverDetails,
			StateAfter: stateJSON,
			Visibility: "public",
			GameOver:   true,
			Result: &game.GameResult{
				WinnerIDs: []uint{playerID},
			},
		})

		return &game.ApplyResult{Events: events}, nil
	}

	// Check draw
	if isBoardFull(s.Board) {
		s.IsDraw = true
		stateJSON, _ := json.Marshal(s)

		// Agent move event
		events = append(events, game.GameEvent{
			Source:     "agent",
			EventType:  "move",
			Actor:     &game.EventEntity{AgentID: &playerID, Seat: intPtr(seat)},
			Details:    moveDetails,
			StateAfter: stateJSON,
			Visibility: "public",
		})

		// System game_over (draw) event
		drawDetails, _ := json.Marshal(map[string]any{
			"is_draw": true,
		})
		events = append(events, game.GameEvent{
			Source:     "system",
			EventType:  "game_over",
			Details:    drawDetails,
			StateAfter: stateJSON,
			Visibility: "public",
			GameOver:   true,
			Result:     &game.GameResult{WinnerIDs: []uint{}},
		})

		return &game.ApplyResult{Events: events}, nil
	}

	// Normal move — advance turn
	s.Turn = (s.Turn + 1) % len(s.Players)
	stateJSON, _ := json.Marshal(s)

	events = append(events, game.GameEvent{
		Source:     "agent",
		EventType:  "move",
		Actor:     &game.EventEntity{AgentID: &playerID, Seat: intPtr(seat)},
		Details:    moveDetails,
		StateAfter: stateJSON,
		Visibility: "public",
	})

	return &game.ApplyResult{Events: events}, nil
}

func checkWin(board [9]string, mark string) bool {
	for _, line := range winLines {
		if board[line[0]] == mark && board[line[1]] == mark && board[line[2]] == mark {
			return true
		}
	}
	return false
}

func isBoardFull(board [9]string) bool {
	for _, cell := range board {
		if cell == "" {
			return false
		}
	}
	return true
}

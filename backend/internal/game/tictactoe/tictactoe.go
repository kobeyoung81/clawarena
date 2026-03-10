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
	Board         [9]string      `json:"board"`
	Players       []uint         `json:"players"`
	Turn          int            `json:"turn"`
	Winner        *uint          `json:"winner"`
	IsDraw        bool           `json:"is_draw"`
	PendingAction *game.PendingAction `json:"pending_action,omitempty"`
}

type Engine struct{}

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

func (e *Engine) InitState(config json.RawMessage, players []uint) (json.RawMessage, error) {
	if len(players) != 2 {
		return nil, errors.New("tic_tac_toe requires exactly 2 players")
	}
	s := State{
		Players: players,
		Turn:    0,
	}
	return json.Marshal(s)
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

func (e *Engine) ApplyAction(raw json.RawMessage, playerID uint, actionRaw json.RawMessage) (game.ActionResult, error) {
	s, err := parseState(raw)
	if err != nil {
		return game.ActionResult{}, err
	}
	if s.Winner != nil || s.IsDraw {
		return game.ActionResult{}, errors.New("game is already over")
	}
	if s.Turn >= len(s.Players) || s.Players[s.Turn] != playerID {
		return game.ActionResult{}, errors.New("not your turn")
	}

	var action moveAction
	if err := json.Unmarshal(actionRaw, &action); err != nil {
		return game.ActionResult{}, fmt.Errorf("invalid action: %w", err)
	}
	if action.Position < 0 || action.Position > 8 {
		return game.ActionResult{}, errors.New("position must be between 0 and 8")
	}
	if s.Board[action.Position] != "" {
		return game.ActionResult{}, errors.New("cell already occupied")
	}

	mark := "X"
	if s.Turn == 1 {
		mark = "O"
	}
	s.Board[action.Position] = mark

	var events []game.GameEvent

	// Check win
	if winner := checkWin(s.Board, mark); winner {
		s.Winner = &playerID
		events = append(events, game.GameEvent{
			Type:       "game_over",
			Message:    fmt.Sprintf("Player %d wins!", playerID),
			Visibility: "public",
		})
		newState, _ := json.Marshal(s)
		return game.ActionResult{
			NewState: newState,
			Events:   events,
			GameOver: true,
			Result: &game.GameResult{
				WinnerIDs: []uint{playerID},
			},
		}, nil
	}

	// Check draw
	if isBoardFull(s.Board) {
		s.IsDraw = true
		events = append(events, game.GameEvent{
			Type:       "game_over",
			Message:    "It's a draw!",
			Visibility: "public",
		})
		newState, _ := json.Marshal(s)
		return game.ActionResult{
			NewState: newState,
			Events:   events,
			GameOver: true,
			Result:   &game.GameResult{WinnerIDs: []uint{}},
		}, nil
	}

	s.Turn = (s.Turn + 1) % len(s.Players)
	newState, _ := json.Marshal(s)
	return game.ActionResult{
		NewState: newState,
		Events:   events,
		GameOver: false,
	}, nil
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

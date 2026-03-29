package tictactoe

import (
	"encoding/json"
	"testing"

	"github.com/clawarena/clawarena/internal/game"
)

// lastState extracts the final state from an ApplyResult's last event.
func lastState(result *game.ApplyResult) json.RawMessage {
	if len(result.Events) == 0 {
		return nil
	}
	return result.Events[len(result.Events)-1].StateAfter
}

// isGameOver checks if any event in the result signals game over.
func isGameOver(result *game.ApplyResult) bool {
	for _, ev := range result.Events {
		if ev.GameOver {
			return true
		}
	}
	return false
}

func initGame(t *testing.T) json.RawMessage {
	t.Helper()
	e := &Engine{}
	state, _, err := e.InitState(nil, []uint{1, 2})
	if err != nil {
		t.Fatalf("InitState failed: %v", err)
	}
	return state
}

func TestInitState(t *testing.T) {
	e := &Engine{}
	state, events, err := e.InitState(nil, []uint{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected seed events from InitState")
	}
	if events[0].EventType != "game_start" {
		t.Errorf("expected first event to be game_start, got %q", events[0].EventType)
	}
	if events[0].StateAfter == nil {
		t.Error("expected StateAfter on game_start event")
	}
	var s State
	if err := json.Unmarshal(state, &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if s.Turn != 0 {
		t.Errorf("expected turn 0, got %d", s.Turn)
	}
	for i, cell := range s.Board {
		if cell != "" {
			t.Errorf("cell %d should be empty, got %q", i, cell)
		}
	}
	if s.Winner != nil {
		t.Errorf("expected no winner initially")
	}
}

func TestInitStateWrongPlayers(t *testing.T) {
	e := &Engine{}
	_, _, err := e.InitState(nil, []uint{1})
	if err == nil {
		t.Fatal("expected error for wrong player count")
	}
}

func TestValidMove(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	action, _ := json.Marshal(map[string]int{"position": 4})
	result, err := e.ApplyAction(state, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isGameOver(result) {
		t.Error("game should not be over after first move")
	}
	var s State
	json.Unmarshal(lastState(result), &s)
	if s.Board[4] != "X" {
		t.Errorf("expected X at position 4, got %q", s.Board[4])
	}
	if s.Turn != 1 {
		t.Errorf("expected turn 1, got %d", s.Turn)
	}
}

func TestInvalidMove_OccupiedCell(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	action, _ := json.Marshal(map[string]int{"position": 4})
	r, err := e.ApplyAction(state, 1, action)
	if err != nil {
		t.Fatalf("first move failed: %v", err)
	}
	state = lastState(r)
	// Player 2 tries same cell
	_, err = e.ApplyAction(state, 2, action)
	if err == nil {
		t.Fatal("expected error for occupied cell")
	}
}

func TestInvalidMove_OutOfRange(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	action, _ := json.Marshal(map[string]int{"position": 9})
	_, err := e.ApplyAction(state, 1, action)
	if err == nil {
		t.Fatal("expected error for out-of-range position")
	}
}

func TestNotYourTurn(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	action, _ := json.Marshal(map[string]int{"position": 0})
	_, err := e.ApplyAction(state, 2, action) // player 2 goes first (should fail)
	if err == nil {
		t.Fatal("expected error: not player 2's turn")
	}
}

func TestWinDetection(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	// P1: 0,1,2 (top row win), P2: 3,4
	moves := []struct {
		player uint
		pos    int
	}{
		{1, 0}, {2, 3},
		{1, 1}, {2, 4},
		{1, 2}, // winning move
	}
	var gameEnded bool
	for _, m := range moves {
		action, _ := json.Marshal(map[string]int{"position": m.pos})
		r, err := e.ApplyAction(state, m.player, action)
		if err != nil {
			t.Fatalf("move (%d→%d) failed: %v", m.player, m.pos, err)
		}
		state = lastState(r)
		if isGameOver(r) {
			gameEnded = true
			break
		}
	}
	if !gameEnded {
		t.Fatal("expected game over after row win")
	}
}

func TestDrawDetection(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	// Draw game: X O X / O X X / O X O
	moves := []struct {
		player uint
		pos    int
	}{
		{1, 0}, {2, 1}, {1, 2},
		{2, 3}, {1, 4}, {2, 6},
		{1, 5}, {2, 8}, {1, 7},
	}
	var lastResult *game.ApplyResult
	for _, m := range moves {
		action, _ := json.Marshal(map[string]int{"position": m.pos})
		r, err := e.ApplyAction(state, m.player, action)
		if err != nil {
			t.Fatalf("move (%d→%d) failed: %v", m.player, m.pos, err)
		}
		state = lastState(r)
		lastResult = r
	}
	if !isGameOver(lastResult) {
		t.Fatal("expected draw (game over)")
	}
	var s State
	json.Unmarshal(lastState(lastResult), &s)
	if !s.IsDraw {
		t.Error("expected is_draw to be true")
	}
	if s.Winner != nil {
		t.Error("expected no winner in draw")
	}
}

func TestGetPlayerView_WithPendingAction(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	view, err := e.GetPlayerView(state, 1)
	if err != nil {
		t.Fatalf("GetPlayerView failed: %v", err)
	}
	var pv PlayerView
	if err := json.Unmarshal(view, &pv); err != nil {
		t.Fatalf("unmarshal PlayerView failed: %v", err)
	}
	if pv.PendingAction == nil {
		t.Error("expected pending action for player 1 (their turn)")
	}
	if pv.PendingAction != nil && pv.PendingAction.ActionType != "move" {
		t.Errorf("expected action type 'move', got %q", pv.PendingAction.ActionType)
	}
}

func TestGetPlayerView_NoPendingAction_OtherPlayer(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	view, err := e.GetPlayerView(state, 2)
	if err != nil {
		t.Fatalf("GetPlayerView failed: %v", err)
	}
	var pv PlayerView
	json.Unmarshal(view, &pv)
	if pv.PendingAction != nil {
		t.Error("player 2 should have no pending action (not their turn)")
	}
}

func TestGetSpectatorView(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	view, err := e.GetSpectatorView(state)
	if err != nil {
		t.Fatalf("GetSpectatorView failed: %v", err)
	}
	if len(view) == 0 {
		t.Error("expected non-empty spectator view")
	}
}

func TestGetPendingActions(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	actions, err := e.GetPendingActions(state)
	if err != nil {
		t.Fatalf("GetPendingActions failed: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 pending action, got %d", len(actions))
	}
	if actions[0].PlayerID != 1 {
		t.Errorf("expected pending action for player 1, got %d", actions[0].PlayerID)
	}
}

func TestGetPendingActions_GameOver(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	moves := []struct {
		player uint
		pos    int
	}{
		{1, 0}, {2, 3}, {1, 1}, {2, 4}, {1, 2},
	}
	for _, m := range moves {
		action, _ := json.Marshal(map[string]int{"position": m.pos})
		r, _ := e.ApplyAction(state, m.player, action)
		state = lastState(r)
	}
	actions, err := e.GetPendingActions(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected no pending actions after game over, got %d", len(actions))
	}
}

func TestSyncronym(t *testing.T) {
	e := &Engine{}
	if e.Syncronym() != "ttt" {
		t.Errorf("expected syncronym 'ttt', got %q", e.Syncronym())
	}
}

func TestNewEventModel(t *testing.T) {
	e := &Engine{}
	m := e.NewEventModel()
	if m.TableName() != "ttt_game_events" {
		t.Errorf("expected table name 'ttt_game_events', got %q", m.TableName())
	}
}

func TestEventsHaveStateAfter(t *testing.T) {
	e := &Engine{}
	state := initGame(t)
	action, _ := json.Marshal(map[string]int{"position": 4})
	result, err := e.ApplyAction(state, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, ev := range result.Events {
		if ev.StateAfter == nil {
			t.Errorf("event %d (%s) has nil StateAfter", i, ev.EventType)
		}
	}
}

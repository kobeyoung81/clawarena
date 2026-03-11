package integration

import (
	"fmt"
	"net/http"
	"testing"
)

func TestTTT_WinDiagonal(t *testing.T) {
	cleanDB(t)
	roomID, a, b := createAndStartTTTGame(t)

	//   X | O |       Move 1: A->0, Move 2: B->1
	//   ---------
	//     | X | O     Move 3: A->4, Move 4: B->5
	//   ---------
	//     |   | X     Move 5: A->8  --> A wins!

	submitAction(t, a, roomID, map[string]any{"position": 0})
	submitAction(t, b, roomID, map[string]any{"position": 1})
	submitAction(t, a, roomID, map[string]any{"position": 4})
	submitAction(t, b, roomID, map[string]any{"position": 5})
	result := submitAction(t, a, roomID, map[string]any{"position": 8})

	if result["game_over"] != true {
		t.Fatal("expected game_over=true")
	}

	// Verify room is finished
	resp := a.get(t, fmt.Sprintf("/api/v1/rooms/%d", roomID))
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"finished"`)

	// Verify Elo changes
	resp = a.get(t, "/api/v1/agents/me")
	assertStatus(t, resp, http.StatusOK)
	var meA map[string]any
	readJSON(t, resp, &meA)
	if eloA, ok := meA["elo_rating"].(float64); !ok || eloA <= 1000 {
		t.Fatalf("winner Elo should increase, got %v", meA["elo_rating"])
	}

	resp = b.get(t, "/api/v1/agents/me")
	assertStatus(t, resp, http.StatusOK)
	var meB map[string]any
	readJSON(t, resp, &meB)
	if eloB, ok := meB["elo_rating"].(float64); !ok || eloB >= 1000 {
		t.Fatalf("loser Elo should decrease, got %v", meB["elo_rating"])
	}
}

func TestTTT_WinRow(t *testing.T) {
	cleanDB(t)
	roomID, a, b := createAndStartTTTGame(t)

	//   X | X | X     Move 1: A->0, Move 3: A->1, Move 5: A->2
	//   ---------
	//   O | O |       Move 2: B->3, Move 4: B->4
	//   ---------
	//     |   |

	submitAction(t, a, roomID, map[string]any{"position": 0})
	submitAction(t, b, roomID, map[string]any{"position": 3})
	submitAction(t, a, roomID, map[string]any{"position": 1})
	submitAction(t, b, roomID, map[string]any{"position": 4})
	result := submitAction(t, a, roomID, map[string]any{"position": 2})

	if result["game_over"] != true {
		t.Fatal("expected game_over=true")
	}
}

func TestTTT_WinColumn(t *testing.T) {
	cleanDB(t)
	roomID, a, b := createAndStartTTTGame(t)

	//   X | O |       Move 1: A->0, Move 3: A->3, Move 5: A->6
	//   ---------
	//   X | O |       Move 2: B->1, Move 4: B->4
	//   ---------
	//   X |   |

	submitAction(t, a, roomID, map[string]any{"position": 0})
	submitAction(t, b, roomID, map[string]any{"position": 1})
	submitAction(t, a, roomID, map[string]any{"position": 3})
	submitAction(t, b, roomID, map[string]any{"position": 4})
	result := submitAction(t, a, roomID, map[string]any{"position": 6})

	if result["game_over"] != true {
		t.Fatal("expected game_over=true")
	}
}

func TestTTT_Draw(t *testing.T) {
	cleanDB(t)
	roomID, a, b := createAndStartTTTGame(t)

	//   X | O | X
	//   ---------
	//   X | O | O
	//   ---------
	//   O | X | X

	submitAction(t, a, roomID, map[string]any{"position": 0}) // X at 0
	submitAction(t, b, roomID, map[string]any{"position": 1}) // O at 1
	submitAction(t, a, roomID, map[string]any{"position": 2}) // X at 2
	submitAction(t, b, roomID, map[string]any{"position": 4}) // O at 4
	submitAction(t, a, roomID, map[string]any{"position": 3}) // X at 3
	submitAction(t, b, roomID, map[string]any{"position": 5}) // O at 5
	submitAction(t, a, roomID, map[string]any{"position": 7}) // X at 7
	submitAction(t, b, roomID, map[string]any{"position": 6}) // O at 6
	result := submitAction(t, a, roomID, map[string]any{"position": 8}) // X at 8

	if result["game_over"] != true {
		t.Fatal("expected game_over=true for draw")
	}

	// Verify Elo unchanged (draw has empty winner_ids)
	resp := a.get(t, "/api/v1/agents/me")
	assertStatus(t, resp, http.StatusOK)
	var meA map[string]any
	readJSON(t, resp, &meA)
	if elo, ok := meA["elo_rating"].(float64); !ok || elo != 1000 {
		t.Fatalf("draw Elo should stay 1000, got %v", meA["elo_rating"])
	}
}

func TestTTT_ErrorWrongTurn(t *testing.T) {
	cleanDB(t)
	roomID, _, b := createAndStartTTTGame(t)

	// B tries to move first (it's A's turn)
	submitActionExpectError(t, b, roomID, map[string]any{"position": 0}, http.StatusBadRequest)
}

func TestTTT_ErrorOccupiedCell(t *testing.T) {
	cleanDB(t)
	roomID, a, b := createAndStartTTTGame(t)

	submitAction(t, a, roomID, map[string]any{"position": 0}) // A plays 0
	submitActionExpectError(t, b, roomID, map[string]any{"position": 0}, http.StatusBadRequest) // B tries same cell
}

func TestTTT_ErrorOutOfRange(t *testing.T) {
	cleanDB(t)
	roomID, a, _ := createAndStartTTTGame(t)

	submitActionExpectError(t, a, roomID, map[string]any{"position": 99}, http.StatusBadRequest)
}

func TestTTT_ErrorActionOnFinishedGame(t *testing.T) {
	cleanDB(t)
	roomID, a, b := createAndStartTTTGame(t)

	// Play to completion (A wins diagonal)
	submitAction(t, a, roomID, map[string]any{"position": 0})
	submitAction(t, b, roomID, map[string]any{"position": 1})
	submitAction(t, a, roomID, map[string]any{"position": 4})
	submitAction(t, b, roomID, map[string]any{"position": 5})
	submitAction(t, a, roomID, map[string]any{"position": 8})

	// Try to act on finished game
	submitActionExpectError(t, a, roomID, map[string]any{"position": 0}, http.StatusBadRequest)
}

func TestTTT_History(t *testing.T) {
	cleanDB(t)
	roomID, a, b := createAndStartTTTGame(t)

	// Play a quick game
	submitAction(t, a, roomID, map[string]any{"position": 0})
	submitAction(t, b, roomID, map[string]any{"position": 1})
	submitAction(t, a, roomID, map[string]any{"position": 4})
	submitAction(t, b, roomID, map[string]any{"position": 5})
	submitAction(t, a, roomID, map[string]any{"position": 8})

	history := getHistory(t, roomID)
	if history["status"] != "finished" {
		t.Fatalf("expected finished, got %v", history["status"])
	}
	timeline, ok := history["timeline"].([]any)
	if !ok || len(timeline) == 0 {
		t.Fatal("expected non-empty timeline")
	}
	// Timeline should have initial state + one entry per move = 6 entries
	if len(timeline) != 6 {
		t.Fatalf("expected 6 timeline entries (init + 5 moves), got %d", len(timeline))
	}
}

func TestTTT_PlayerView(t *testing.T) {
	cleanDB(t)
	roomID, a, b := createAndStartTTTGame(t)

	// A should have a pending action (it's their turn)
	stateA := getState(t, a, roomID)
	if !hasPendingAction(stateA) {
		t.Fatal("player A should have a pending action on their turn")
	}

	// B should NOT have a pending action
	stateB := getState(t, b, roomID)
	if hasPendingAction(stateB) {
		t.Fatal("player B should NOT have a pending action when it's A's turn")
	}

	// After A moves, B should have the pending action
	submitAction(t, a, roomID, map[string]any{"position": 0})

	stateB = getState(t, b, roomID)
	if !hasPendingAction(stateB) {
		t.Fatal("player B should have a pending action after A moves")
	}

	stateA = getState(t, a, roomID)
	if hasPendingAction(stateA) {
		t.Fatal("player A should NOT have a pending action after their move")
	}
}

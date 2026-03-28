package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	cleanDB(t)

	anon := anonClient()
	resp := anon.get(t, "/health")
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"status":"ok"`)
}

func TestGameTypes(t *testing.T) {
	anon := anonClient()
	resp := anon.get(t, "/api/v1/games")
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"tic_tac_toe"`)
	assertContains(t, body, `"clawedwolf"`)
}

func TestGameTypeByID(t *testing.T) {
	gtID := getGameTypeID(t, "tic_tac_toe")
	anon := anonClient()
	resp := anon.get(t, fmt.Sprintf("/api/v1/games/%d", gtID))
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"tic_tac_toe"`)
}

func TestAgentRegistration(t *testing.T) {
	cleanDB(t)

	// Register succeeds
	anon := anonClient()
	resp := anon.post(t, "/api/v1/agents/register", map[string]string{"name": "AgentReg1"})
	assertStatus(t, resp, http.StatusCreated)
	body := readBody(t, resp)
	assertContains(t, body, `"api_key"`)
	assertContains(t, body, `"AgentReg1"`)

	// Duplicate name → 409
	resp = anon.post(t, "/api/v1/agents/register", map[string]string{"name": "AgentReg1"})
	assertStatus(t, resp, http.StatusConflict)
	resp.Body.Close()
}

func TestAgentRegistrationValidation(t *testing.T) {
	cleanDB(t)

	anon := anonClient()

	// Empty name → 400
	resp := anon.post(t, "/api/v1/agents/register", map[string]string{"name": ""})
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()

	// Too-long name → 400
	longName := strings.Repeat("x", 101)
	resp = anon.post(t, "/api/v1/agents/register", map[string]string{"name": longName})
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestAuthentication(t *testing.T) {
	cleanDB(t)

	agent := registerAgent(t, "AuthAgent")

	// No auth → 401
	anon := anonClient()
	resp := anon.get(t, "/api/v1/rooms")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()

	// Invalid token → 401
	badClient := &apiClient{baseURL: baseURL, apiKey: "invalid-token-xxx"}
	resp = badClient.get(t, "/api/v1/rooms")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()

	// Valid token → 200
	resp = agent.get(t, "/api/v1/rooms")
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// /agents/me
	resp = agent.get(t, "/api/v1/agents/me")
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"AuthAgent"`)
}

func TestRoomCreation(t *testing.T) {
	cleanDB(t)

	agent := registerAgent(t, "RoomOwner")
	gtID := getGameTypeID(t, "tic_tac_toe")

	// Create room → 201
	resp := agent.post(t, "/api/v1/rooms", map[string]any{"game_type_id": gtID})
	assertStatus(t, resp, http.StatusCreated)
	body := readBody(t, resp)
	assertContains(t, body, `"waiting"`)

	// Second room while active → 409
	resp = agent.post(t, "/api/v1/rooms", map[string]any{"game_type_id": gtID})
	assertStatus(t, resp, http.StatusConflict)
	resp.Body.Close()
}

func TestRoomJoinAndReadyCheck(t *testing.T) {
	cleanDB(t)

	a := registerAgent(t, "JoinA")
	b := registerAgent(t, "JoinB")
	gtID := getGameTypeID(t, "tic_tac_toe")

	roomID := createRoom(t, a, gtID)

	// Join → ready_check
	status := joinRoom(t, b, roomID)
	if status != "ready_check" {
		t.Fatalf("expected ready_check, got %s", status)
	}

	// Ready → one at a time
	result := readyUp(t, a, roomID)
	if result["status"] != "ready_check" {
		t.Fatalf("expected ready_check after first ready, got %v", result["status"])
	}

	// Second ready → playing
	result = readyUp(t, b, roomID)
	if result["status"] != "playing" {
		t.Fatalf("expected playing, got %v", result["status"])
	}
}

func TestRoomLeaveWhileWaiting(t *testing.T) {
	cleanDB(t)

	a := registerAgent(t, "LeaveA")
	b := registerAgent(t, "LeaveB")
	gtID := getGameTypeID(t, "tic_tac_toe")

	roomID := createRoom(t, a, gtID)

	// Leave while waiting
	resp := a.post(t, fmt.Sprintf("/api/v1/rooms/%d/leave", roomID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Room should be cancelled
	resp = b.get(t, fmt.Sprintf("/api/v1/rooms/%d", roomID))
	// Need auth to view room, register b first then check
	// Actually b needs to be registered already (done above). But b is not in the room.
	// Need an agent who can view rooms. Let's register a new one.
	viewer := registerAgent(t, "Viewer1")
	resp = viewer.get(t, fmt.Sprintf("/api/v1/rooms/%d", roomID))
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"cancelled"`)
}

func TestRoomLeaveForfeit(t *testing.T) {
	cleanDB(t)

	roomID, a, b := createAndStartTTTGame(t)

	// A leaves during play → B wins
	resp := a.post(t, fmt.Sprintf("/api/v1/rooms/%d/leave", roomID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	resp = b.get(t, fmt.Sprintf("/api/v1/rooms/%d", roomID))
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"finished"`)
}

func TestRoomListFilters(t *testing.T) {
	cleanDB(t)

	// Create and finish a game
	roomID, a, _ := createAndStartTTTGame(t)
	// Play a quick game: A wins diagonal 0-4-8
	submitAction(t, a, roomID, map[string]any{"position": 0})

	// We need a second agent handle - get the second agent from the room state
	// Actually b was returned from createAndStartTTTGame, let me rework...
	// Let me just create a new game and forfeit to get a finished room
	roomID2, a2, b2 := createAndStartTTTGame(t)
	resp := a2.post(t, fmt.Sprintf("/api/v1/rooms/%d/leave", roomID2), nil)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Filter by status=finished
	resp = b2.get(t, "/api/v1/rooms?status=finished")
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"finished"`)
}

func TestSpectatorRoomView(t *testing.T) {
	cleanDB(t)

	roomID, _, _ := createAndStartTTTGame(t)

	// Spectator (no auth) can view room details
	anon := anonClient()
	resp := anon.get(t, fmt.Sprintf("/api/v1/rooms/%d", roomID))
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"playing"`)
}

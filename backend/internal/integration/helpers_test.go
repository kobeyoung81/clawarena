package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/clawarena/clawarena/internal/models"
)

// apiClient wraps HTTP calls to the test server with optional auth.
type apiClient struct {
	baseURL string
	apiKey  string
	agentID uint
	name    string
}

func (c *apiClient) get(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func (c *apiClient) post(t *testing.T, path string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest("POST", c.baseURL+path, reader)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// anonClient returns a client with no auth credentials.
func anonClient() *apiClient {
	return &apiClient{baseURL: baseURL}
}

// readJSON decodes a response body into the given target and closes the body.
func readJSON(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// readBody reads and returns the response body as a string.
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// assertStatus checks that the response has the expected status code.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d; body: %s", want, resp.StatusCode, string(body))
	}
}

// assertContains checks that s contains substr.
func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !contains(s, substr) {
		t.Fatalf("expected %q to contain %q", truncate(s, 200), substr)
	}
}

func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// registerAgent registers a new agent and returns a configured apiClient.
func registerAgent(t *testing.T, name string) *apiClient {
	t.Helper()
	anon := anonClient()
	resp := anon.post(t, "/api/v1/agents/register", map[string]string{"name": name})
	assertStatus(t, resp, http.StatusCreated)

	var result struct {
		ID     uint   `json:"id"`
		Name   string `json:"name"`
		APIKey string `json:"api_key"`
	}
	readJSON(t, resp, &result)

	return &apiClient{
		baseURL: baseURL,
		apiKey:  result.APIKey,
		agentID: result.ID,
		name:    result.Name,
	}
}

// getGameTypeID looks up a game type by name and returns its ID.
func getGameTypeID(t *testing.T, name string) uint {
	t.Helper()
	var gt models.GameType
	if err := testDB.Where("name = ?", name).First(&gt).Error; err != nil {
		t.Fatalf("game type %q not found: %v", name, err)
	}
	return gt.ID
}

// createRoom creates a room for the given game type. Returns room ID.
func createRoom(t *testing.T, owner *apiClient, gameTypeID uint) uint {
	t.Helper()
	resp := owner.post(t, "/api/v1/rooms", map[string]any{"game_type_id": gameTypeID})
	assertStatus(t, resp, http.StatusCreated)
	var result struct {
		ID uint `json:"id"`
	}
	readJSON(t, resp, &result)
	return result.ID
}

// joinRoom joins a room and returns the join response status.
func joinRoom(t *testing.T, agent *apiClient, roomID uint) string {
	t.Helper()
	resp := agent.post(t, fmt.Sprintf("/api/v1/rooms/%d/join", roomID), nil)
	assertStatus(t, resp, http.StatusOK)
	var result struct {
		Status string `json:"status"`
	}
	readJSON(t, resp, &result)
	return result.Status
}

// readyUp marks an agent as ready in a room. Returns the ready response.
func readyUp(t *testing.T, agent *apiClient, roomID uint) map[string]any {
	t.Helper()
	resp := agent.post(t, fmt.Sprintf("/api/v1/rooms/%d/ready", roomID), nil)
	assertStatus(t, resp, http.StatusOK)
	var result map[string]any
	readJSON(t, resp, &result)
	return result
}

// createAndStartTTTGame sets up a 2-player tic-tac-toe room and starts the game.
// Returns roomID, and the two agents (A is always slot 0 / X, B is slot 1 / O).
func createAndStartTTTGame(t *testing.T) (uint, *apiClient, *apiClient) {
	t.Helper()
	a := registerAgent(t, uniqueName("ttt_a"))
	b := registerAgent(t, uniqueName("ttt_b"))

	gtID := getGameTypeID(t, "tic_tac_toe")
	roomID := createRoom(t, a, gtID)
	joinRoom(t, b, roomID)
	readyUp(t, a, roomID)
	result := readyUp(t, b, roomID)
	if result["status"] != "playing" {
		t.Fatalf("expected playing, got %v", result["status"])
	}
	return roomID, a, b
}

// createAndStartWWGame sets up a 6-player clawedwolf room and starts the game.
// Returns roomID and the 6 agents in slot order.
func createAndStartWWGame(t *testing.T) (uint, []*apiClient) {
	t.Helper()
	agents := make([]*apiClient, 6)
	for i := range agents {
		agents[i] = registerAgent(t, uniqueName(fmt.Sprintf("ww_%d", i)))
	}

	gtID := getGameTypeID(t, "clawedwolf")
	roomID := createRoom(t, agents[0], gtID)
	for i := 1; i < 6; i++ {
		joinRoom(t, agents[i], roomID)
	}
	for i := 0; i < 6; i++ {
		readyUp(t, agents[i], roomID)
	}

	return roomID, agents
}

// submitAction submits a game action and returns the response body as a map.
func submitAction(t *testing.T, agent *apiClient, roomID uint, action any) map[string]any {
	t.Helper()
	resp := agent.post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{"action": action})
	assertStatus(t, resp, http.StatusOK)
	var result map[string]any
	readJSON(t, resp, &result)
	return result
}

// submitActionExpectError submits an action and expects a specific HTTP error code.
func submitActionExpectError(t *testing.T, agent *apiClient, roomID uint, action any, wantCode int) string {
	t.Helper()
	resp := agent.post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{"action": action})
	assertStatus(t, resp, wantCode)
	return readBody(t, resp)
}

// getState returns the game state response for a room.
func getState(t *testing.T, agent *apiClient, roomID uint) map[string]any {
	t.Helper()
	resp := agent.get(t, fmt.Sprintf("/api/v1/rooms/%d/state", roomID))
	assertStatus(t, resp, http.StatusOK)
	var result map[string]any
	readJSON(t, resp, &result)
	return result
}

// getHistory returns the game history for a room.
func getHistory(t *testing.T, roomID uint) map[string]any {
	t.Helper()
	anon := anonClient()
	resp := anon.get(t, fmt.Sprintf("/api/v1/rooms/%d/history", roomID))
	assertStatus(t, resp, http.StatusOK)
	var result map[string]any
	readJSON(t, resp, &result)
	return result
}

// cleanDB truncates all game-related tables for test isolation.
func cleanDB(t *testing.T) {
	t.Helper()
	// Disable FK checks for truncation
	testDB.Exec("SET FOREIGN_KEY_CHECKS = 0")
	for _, table := range []string{"game_actions", "game_states", "room_agents", "rooms", "agents"} {
		if err := testDB.Exec("TRUNCATE TABLE " + table).Error; err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
	testDB.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

// --- ClawedWolf helpers ---

// wwRoleMap maps role names to lists of apiClients with that role.
type wwRoleMap map[string][]*apiClient

// getInnerState extracts the nested "state" map from a GameStateResponse.
func getInnerState(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	s, ok := resp["state"].(map[string]any)
	if !ok {
		t.Fatal("state field is not a map")
	}
	return s
}

// discoverRoles queries each agent's state to find their role assignment.
func discoverRoles(t *testing.T, agents []*apiClient, roomID uint) wwRoleMap {
	t.Helper()
	roles := wwRoleMap{}
	for _, a := range agents {
		resp := getState(t, a, roomID)
		inner := getInnerState(t, resp)
		role, ok := inner["your_role"].(string)
		if !ok {
			t.Fatalf("agent %d: no your_role in state", a.agentID)
		}
		roles[role] = append(roles[role], a)
	}
	return roles
}

// findAgentSeat returns the seat number for the given agent in the room.
func findAgentSeat(t *testing.T, agent *apiClient, roomID uint) int {
	t.Helper()
	resp := getState(t, agent, roomID)
	inner := getInnerState(t, resp)
	seat, ok := inner["your_seat"].(float64)
	if !ok {
		t.Fatalf("agent %d: no your_seat in state", agent.agentID)
	}
	return int(seat)
}

// wwSubmitKillVote submits a clawedwolf kill vote.
func wwSubmitKillVote(t *testing.T, wolf *apiClient, roomID uint, targetSeat int) map[string]any {
	t.Helper()
	return submitAction(t, wolf, roomID, map[string]any{
		"type":        "kill_vote",
		"target_seat": targetSeat,
	})
}

// wwSubmitInvestigate submits a seer investigation.
func wwSubmitInvestigate(t *testing.T, seer *apiClient, roomID uint, targetSeat int) map[string]any {
	t.Helper()
	return submitAction(t, seer, roomID, map[string]any{
		"type":        "investigate",
		"target_seat": targetSeat,
	})
}

// wwSubmitProtect submits a guard protection.
func wwSubmitProtect(t *testing.T, guard *apiClient, roomID uint, targetSeat int) map[string]any {
	t.Helper()
	return submitAction(t, guard, roomID, map[string]any{
		"type":        "protect",
		"target_seat": targetSeat,
	})
}

// wwSubmitSpeak submits a day discussion speech.
func wwSubmitSpeak(t *testing.T, agent *apiClient, roomID uint, message string) map[string]any {
	t.Helper()
	return submitAction(t, agent, roomID, map[string]any{
		"type":    "speak",
		"message": message,
	})
}

// wwSubmitVote submits a day vote.
func wwSubmitVote(t *testing.T, agent *apiClient, roomID uint, targetSeat int) map[string]any {
	t.Helper()
	return submitAction(t, agent, roomID, map[string]any{
		"type":        "vote",
		"target_seat": targetSeat,
	})
}

// wwGetPhase returns the current game phase.
func wwGetPhase(t *testing.T, agent *apiClient, roomID uint) string {
	t.Helper()
	state := getState(t, agent, roomID)
	s, ok := state["state"].(map[string]any)
	if !ok {
		t.Fatalf("state field is not a map")
	}
	phase, ok := s["phase"].(string)
	if !ok {
		t.Fatalf("no phase in state")
	}
	return phase
}

// wwPlayNightRound plays a full night round (wolves vote, seer investigates, guard protects).
// Returns the events from the guard's protect action (which resolves the night).
func wwPlayNightRound(t *testing.T, roles wwRoleMap, agents []*apiClient, roomID uint, killSeat, seerTargetSeat, guardTargetSeat int) {
	t.Helper()

	// Wolves vote
	for _, wolf := range roles["clawedwolf"] {
		state := getState(t, wolf, roomID)
		if hasPendingAction(state) {
			wwSubmitKillVote(t, wolf, roomID, killSeat)
		}
	}

	// Seer investigates (if alive)
	if len(roles["seer"]) > 0 {
		seer := roles["seer"][0]
		state := getState(t, seer, roomID)
		if hasPendingAction(state) {
			wwSubmitInvestigate(t, seer, roomID, seerTargetSeat)
		}
	}

	// Guard protects (if alive)
	if len(roles["guard"]) > 0 {
		guard := roles["guard"][0]
		state := getState(t, guard, roomID)
		if hasPendingAction(state) {
			wwSubmitProtect(t, guard, roomID, guardTargetSeat)
		}
	}
}

// wwPlayDayRound plays a full day round (all alive speak, then all vote for targetSeat).
// Use targetSeat=-1 to abstain. The player at targetSeat will abstain (can't self-vote).
func wwPlayDayRound(t *testing.T, agents []*apiClient, roomID uint, voteSeat int) {
	t.Helper()

	// Discussion: each alive player speaks in order
	for {
		// Find the agent who has a pending "speak" action
		speaker := findPendingAgent(t, agents, roomID, "speak")
		if speaker == nil {
			break
		}
		wwSubmitSpeak(t, speaker, roomID, "I have nothing to say.")
	}

	// Vote: each alive player votes
	for _, agent := range agents {
		state := getState(t, agent, roomID)
		if hasPendingAction(state) {
			pa := getPendingAction(state)
			if pa["action_type"] == "vote" {
				target := voteSeat
				// Can't vote for yourself — abstain instead
				mySeat := findAgentSeat(t, agent, roomID)
				if target == mySeat {
					target = -1
				}
				wwSubmitVote(t, agent, roomID, target)
			}
		}
	}
}

// findPendingAgent finds the agent with a pending action of the given type.
func findPendingAgent(t *testing.T, agents []*apiClient, roomID uint, actionType string) *apiClient {
	t.Helper()
	for _, a := range agents {
		state := getState(t, a, roomID)
		if hasPendingAction(state) {
			pa := getPendingAction(state)
			if pa["action_type"] == actionType {
				return a
			}
		}
	}
	return nil
}

// hasPendingAction checks if the state response has a pending action.
func hasPendingAction(state map[string]any) bool {
	pa, ok := state["pending_action"]
	return ok && pa != nil
}

// getPendingAction extracts the pending_action map from state.
func getPendingAction(state map[string]any) map[string]any {
	pa, ok := state["pending_action"].(map[string]any)
	if !ok {
		return nil
	}
	return pa
}

// --- Name generation ---

var nameCounter int

func uniqueName(prefix string) string {
	nameCounter++
	return fmt.Sprintf("%s_%d", prefix, nameCounter)
}

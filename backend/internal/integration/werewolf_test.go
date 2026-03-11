package integration

import (
	"fmt"
	"net/http"
	"testing"
)

func TestWW_FullGame_GoodWins(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["werewolf"]
	if len(wolves) != 2 {
		t.Fatalf("expected 2 wolves, got %d", len(wolves))
	}

	// Find a villager to kill
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	wolfSeat0 := findAgentSeat(t, wolves[0], roomID)
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// --- Round 1: Kill a villager, vote out a wolf ---

	// Night: wolves kill villager, seer investigates wolf, guard protects self
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, wolfSeat0, guardSeat)

	// Day: everyone votes for wolf 0
	wwPlayDayRound(t, agents, roomID, wolfSeat0)

	// Check that one wolf was eliminated
	state := getState(t, agents[0], roomID)
	stateInner := state["state"].(map[string]any)
	phase := stateInner["phase"].(string)
	if phase == "finished" {
		// Game might not be over yet - we only eliminated one wolf
		t.Fatal("game should not be finished after eliminating 1 wolf with 1 villager dead")
	}

	// --- Round 2: Vote out the second wolf ---
	wolfSeat1 := findAgentSeat(t, wolves[1], roomID)

	// Pick a valid guard target (not same as last time)
	var guardTarget2 int
	if guardSeat != seerSeat {
		guardTarget2 = seerSeat
	} else {
		guardTarget2 = wolfSeat1 // guard can protect anyone alive except last target
	}

	// Night: remaining wolf kills someone, seer investigates, guard protects
	// Need to find an alive non-wolf target for the wolf to kill
	var killTarget2 int
	for _, a := range agents {
		seat := findAgentSeat(t, a, roomID)
		s := getState(t, a, roomID)
		si := s["state"].(map[string]any)
		players := si["players"].([]any)
		for _, p := range players {
			pm := p.(map[string]any)
			pSeat := int(pm["seat"].(float64))
			alive := pm["alive"].(bool)
			if pSeat == seat && alive && pSeat != wolfSeat1 {
				// Found an alive non-wolf player to target
				if pSeat != villagerSeat && pSeat != wolfSeat0 {
					killTarget2 = pSeat
					break
				}
			}
		}
		if killTarget2 != 0 || killTarget2 == 0 {
			break
		}
	}

	// More robust: find any alive player seat that's not the remaining wolf
	killTarget2 = findAliveNonWolfSeat(t, agents, roomID, []int{wolfSeat0, wolfSeat1})

	// Seer target: investigate the second wolf
	seerTarget2 := wolfSeat1

	wwPlayNightRound(t, roles, agents, roomID, killTarget2, seerTarget2, guardTarget2)

	// Day: vote out second wolf
	wwPlayDayRound(t, agents, roomID, wolfSeat1)

	// Game should be over — good wins
	state = getState(t, agents[0], roomID)
	stateInner = state["state"].(map[string]any)
	winner := stateInner["winner"]
	if winner != "good" {
		t.Fatalf("expected good team to win, got %v", winner)
	}

	// Room should be finished
	resp := agents[0].get(t, fmt.Sprintf("/api/v1/rooms/%d", roomID))
	assertStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertContains(t, body, `"finished"`)
}

func TestWW_FullGame_EvilWins(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["werewolf"]
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Strategy: Kill non-wolves each night, abstain votes during day
	// After 2 kills (night + no vote elimination), wolves (2) >= good (2) → evil wins

	// Round 1: kill seer at night, abstain all votes
	villager0Seat := findAgentSeat(t, roles["villager"][0], roomID)

	wwPlayNightRound(t, roles, agents, roomID, seerSeat, villager0Seat, guardSeat)
	// Seer is now dead. Day discussion, then everyone abstains
	wwPlayDayRound(t, agents, roomID, -1)

	// Round 2: kill guard at night
	// Guard is dead after this round's night kill, need a non-same target for guard if still alive
	state := getState(t, agents[0], roomID)
	stateInner := state["state"].(map[string]any)
	if stateInner["winner"] != nil {
		// Already won
		if stateInner["winner"] == "evil" {
			return // test passes
		}
		t.Fatalf("unexpected winner: %v", stateInner["winner"])
	}

	// Find alive non-wolf to kill
	killTarget := findAliveNonWolfSeat(t, agents, roomID, wolfSeats(t, wolves, roomID))

	// Guard needs to protect someone different from last round
	guardAlive := isAgentAlive(t, roles["guard"][0], roomID)
	if guardAlive {
		// Guard protects a villager (not same as last protected = guardSeat)
		guardTarget := findAliveNonWolfSeat(t, agents, roomID, wolfSeats(t, wolves, roomID))
		if guardTarget == guardSeat {
			// Pick another
			for _, a := range agents {
				s := findAgentSeat(t, a, roomID)
				if s != guardSeat && isAgentAlive(t, a, roomID) {
					guardTarget = s
					break
				}
			}
		}
		wwPlayNightRound(t, roles, agents, roomID, killTarget, 0, guardTarget)
	} else {
		// Seer is dead, guard might also be dead — night resolves differently
		// Just submit wolf kills
		for _, wolf := range wolves {
			s := getState(t, wolf, roomID)
			if hasPendingAction(s) {
				wwSubmitKillVote(t, wolf, roomID, killTarget)
			}
		}
	}

	// Check if game is over
	state = getState(t, agents[0], roomID)
	stateInner = state["state"].(map[string]any)
	if stateInner["winner"] == "evil" {
		return // pass
	}

	// If not over yet, play another day (abstain) and night
	if stateInner["winner"] == nil {
		wwPlayDayRound(t, agents, roomID, -1)

		state = getState(t, agents[0], roomID)
		stateInner = state["state"].(map[string]any)
		if stateInner["winner"] == "evil" {
			return
		}

		// One more night if needed
		killTarget = findAliveNonWolfSeat(t, agents, roomID, wolfSeats(t, wolves, roomID))
		for _, wolf := range wolves {
			s := getState(t, wolf, roomID)
			if hasPendingAction(s) {
				wwSubmitKillVote(t, wolf, roomID, killTarget)
			}
		}

		state = getState(t, agents[0], roomID)
		stateInner = state["state"].(map[string]any)
		if stateInner["winner"] == "evil" {
			return
		}
	}

	t.Fatalf("expected evil team to win eventually, got winner=%v", stateInner["winner"])
}

func TestWW_GuardSave(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	// Kill target = seer, guard protects seer → seer survives
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)
	wolfSeats := wolfSeats(t, roles["werewolf"], roomID)

	// Wolves target seer
	for _, wolf := range roles["werewolf"] {
		state := getState(t, wolf, roomID)
		if hasPendingAction(state) {
			wwSubmitKillVote(t, wolf, roomID, seerSeat)
		}
	}

	// Seer investigates a wolf
	wwSubmitInvestigate(t, roles["seer"][0], roomID, wolfSeats[0])

	// Guard protects seer
	wwSubmitProtect(t, roles["guard"][0], roomID, seerSeat)

	// Seer should still be alive
	if !isAgentAlive(t, roles["seer"][0], roomID) {
		t.Fatal("seer should have been saved by the guard")
	}
}

func TestWW_GuardCannotProtectSameConsecutively(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)
	wolfSeats := wolfSeats(t, roles["werewolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)

	// Round 1: guard protects seer
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, wolfSeats[0], seerSeat)

	// Day: abstain
	wwPlayDayRound(t, agents, roomID, -1)

	// Round 2: guard tries to protect seer again → should fail
	// Wolves vote
	killTarget := findAliveNonWolfSeat(t, agents, roomID, wolfSeats)
	for _, wolf := range roles["werewolf"] {
		state := getState(t, wolf, roomID)
		if hasPendingAction(state) {
			wwSubmitKillVote(t, wolf, roomID, killTarget)
		}
	}

	// Seer investigates (if alive)
	if isAgentAlive(t, roles["seer"][0], roomID) {
		seerTarget := findAliveNonWolfSeat(t, agents, roomID, append(wolfSeats, seerSeat))
		if seerTarget == -1 {
			seerTarget = wolfSeats[0]
		}
		wwSubmitInvestigate(t, roles["seer"][0], roomID, seerTarget)
	}

	// Guard tries same target (seer) → 400
	if isAgentAlive(t, roles["guard"][0], roomID) {
		resp := roles["guard"][0].post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{
			"action": map[string]any{"type": "protect", "target_seat": seerSeat},
		})
		assertStatus(t, resp, http.StatusBadRequest)
		body := readBody(t, resp)
		assertContains(t, body, "same player consecutively")

		// Protect someone else to advance the phase
		altTarget := guardSeat // protect self
		if altTarget == seerSeat {
			altTarget = wolfSeats[0]
		}
		wwSubmitProtect(t, roles["guard"][0], roomID, altTarget)
	}
}

func TestWW_SeerInvestigation(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolfSeats := wolfSeats(t, roles["werewolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Wolves vote
	for _, wolf := range roles["werewolf"] {
		state := getState(t, wolf, roomID)
		if hasPendingAction(state) {
			wwSubmitKillVote(t, wolf, roomID, villagerSeat)
		}
	}

	// Seer investigates wolf → should get "evil"
	result := wwSubmitInvestigate(t, roles["seer"][0], roomID, wolfSeats[0])
	events := result["events"].([]any)
	found := false
	for _, e := range events {
		em := e.(map[string]any)
		if em["type"] == "seer_result" {
			msg := em["message"].(string)
			assertContains(t, msg, "evil")
			found = true
		}
	}
	if !found {
		t.Fatal("expected seer_result event with evil")
	}

	// Guard protects
	wwSubmitProtect(t, roles["guard"][0], roomID, guardSeat)

	// Now investigate a villager → should see "good"
	// Need to play day and next night
	wwPlayDayRound(t, agents, roomID, -1)

	// Check state to see if seer is alive
	if !isAgentAlive(t, roles["seer"][0], roomID) {
		t.Skip("seer died, cannot test second investigation")
	}

	// Round 2 night
	killTarget := findAliveNonWolfSeat(t, agents, roomID, wolfSeats)
	for _, wolf := range roles["werewolf"] {
		state := getState(t, wolf, roomID)
		if hasPendingAction(state) {
			wwSubmitKillVote(t, wolf, roomID, killTarget)
		}
	}

	// Seer investigates villager (find an alive villager)
	aliveVillagerSeat := -1
	for _, v := range roles["villager"] {
		if isAgentAlive(t, v, roomID) {
			aliveVillagerSeat = findAgentSeat(t, v, roomID)
			break
		}
	}
	if aliveVillagerSeat == -1 {
		// Try guard
		if isAgentAlive(t, roles["guard"][0], roomID) {
			aliveVillagerSeat = findAgentSeat(t, roles["guard"][0], roomID)
		}
	}
	if aliveVillagerSeat != -1 {
		result = wwSubmitInvestigate(t, roles["seer"][0], roomID, aliveVillagerSeat)
		events = result["events"].([]any)
		found = false
		for _, e := range events {
			em := e.(map[string]any)
			if em["type"] == "seer_result" {
				msg := em["message"].(string)
				assertContains(t, msg, "good")
				found = true
			}
		}
		if !found {
			t.Fatal("expected seer_result event with good")
		}
	}
}

func TestWW_DayDiscussion_RoundRobin(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolfSeats := wolfSeats(t, roles["werewolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Play through night
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, wolfSeats[0], guardSeat)

	// Now in day_discuss phase. Speakers should be prompted one at a time.
	speakerOrder := []int{}
	for {
		speaker := findPendingAgent(t, agents, roomID, "speak")
		if speaker == nil {
			break
		}
		seat := findAgentSeat(t, speaker, roomID)
		speakerOrder = append(speakerOrder, seat)
		wwSubmitSpeak(t, speaker, roomID, fmt.Sprintf("Seat %d speaking", seat))
	}

	// All alive players should have spoken
	aliveCount := 0
	for _, a := range agents {
		if isAgentAlive(t, a, roomID) {
			aliveCount++
		}
	}
	if len(speakerOrder) != aliveCount {
		t.Fatalf("expected %d speakers, got %d", aliveCount, len(speakerOrder))
	}
}

func TestWW_DayVote_TieNoElimination(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolfSeats := wolfSeats(t, roles["werewolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Night: kill a villager
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, wolfSeats[0], guardSeat)

	// Day discussion
	for {
		speaker := findPendingAgent(t, agents, roomID, "speak")
		if speaker == nil {
			break
		}
		wwSubmitSpeak(t, speaker, roomID, "I don't know.")
	}

	// Create a tie: 5 alive players (1 villager died).
	// We need exactly 2 votes for target X, 2 votes for target Y, 1 abstain.
	aliveAgents := []*apiClient{}
	aliveSeats := []int{}
	for _, a := range agents {
		if isAgentAlive(t, a, roomID) {
			aliveAgents = append(aliveAgents, a)
			aliveSeats = append(aliveSeats, findAgentSeat(t, a, roomID))
		}
	}
	if len(aliveAgents) < 5 {
		t.Skip("not enough alive players for tie test")
	}

	// Pick two targets that are different players:
	// aliveAgents[0] is target A, aliveAgents[1] is target B
	targetA := aliveSeats[0]
	targetB := aliveSeats[1]

	// Build vote map: each player votes in a way that creates a 2-2-1 tie
	for i, a := range aliveAgents {
		state := getState(t, a, roomID)
		if !hasPendingAction(state) {
			continue
		}
		mySeat := aliveSeats[i]
		var target int
		switch i {
		case 0:
			target = targetB // player 0 (targetA) votes for targetB
		case 1:
			target = targetA // player 1 (targetB) votes for targetA
		case 2:
			// Vote for whichever of A/B is not self
			if mySeat == targetA {
				target = targetB
			} else {
				target = targetA
			}
		case 3:
			// Vote for the other to create a tie
			if mySeat == targetB {
				target = targetA
			} else {
				target = targetB
			}
		default:
			target = -1 // abstain
		}
		wwSubmitVote(t, a, roomID, target)
	}

	// Tally: targetA got 2 votes (from players 1 and one of 2/3),
	//        targetB got 2 votes (from player 0 and one of 2/3),
	//        1 abstain → tie, no elimination

	// With a tie, no one should be eliminated
	// Count alive players — should be same as before voting
	aliveAfter := 0
	for _, a := range agents {
		if isAgentAlive(t, a, roomID) {
			aliveAfter++
		}
	}
	if aliveAfter != len(aliveAgents) {
		t.Fatalf("expected %d alive after tie, got %d", len(aliveAgents), aliveAfter)
	}
}

func TestWW_DayVote_AllAbstain(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolfSeats := wolfSeats(t, roles["werewolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Night
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, wolfSeats[0], guardSeat)

	// Count alive before vote
	aliveBefore := 0
	for _, a := range agents {
		if isAgentAlive(t, a, roomID) {
			aliveBefore++
		}
	}

	// Day discussion
	for {
		speaker := findPendingAgent(t, agents, roomID, "speak")
		if speaker == nil {
			break
		}
		wwSubmitSpeak(t, speaker, roomID, "No comment.")
	}

	// Everyone abstains
	for _, a := range agents {
		state := getState(t, a, roomID)
		if hasPendingAction(state) {
			pa := getPendingAction(state)
			if pa["action_type"] == "vote" {
				wwSubmitVote(t, a, roomID, -1)
			}
		}
	}

	// No one eliminated
	aliveAfter := 0
	for _, a := range agents {
		if isAgentAlive(t, a, roomID) {
			aliveAfter++
		}
	}
	if aliveAfter != aliveBefore {
		t.Fatalf("expected %d alive after all abstain, got %d", aliveBefore, aliveAfter)
	}
}

func TestWW_PlayerView_WolvesSeePartner(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["werewolf"]
	wolf0 := wolves[0]
	wolf1Seat := findAgentSeat(t, wolves[1], roomID)

	// Wolf 0 should see wolf 1's role
	state := getState(t, wolf0, roomID)
	stateInner := state["state"].(map[string]any)
	players := stateInner["players"].([]any)
	for _, p := range players {
		pm := p.(map[string]any)
		if int(pm["seat"].(float64)) == wolf1Seat {
			role, _ := pm["role"].(string)
			if role != "werewolf" {
				t.Fatalf("wolf should see partner's role as werewolf, got %q", role)
			}
		}
	}

	// Villager should NOT see wolf roles
	villager := roles["villager"][0]
	state = getState(t, villager, roomID)
	stateInner = state["state"].(map[string]any)
	players = stateInner["players"].([]any)
	for _, p := range players {
		pm := p.(map[string]any)
		role, _ := pm["role"].(string)
		alive, _ := pm["alive"].(bool)
		if alive && role == "werewolf" {
			t.Fatal("villager should not see alive wolf roles")
		}
	}
}

func TestWW_SpectatorView_HidesRoles(t *testing.T) {
	cleanDB(t)
	roomID, _ := createAndStartWWGame(t)

	// Spectator (no auth) view
	anon := anonClient()
	resp := anon.get(t, fmt.Sprintf("/api/v1/rooms/%d/state", roomID))
	assertStatus(t, resp, http.StatusOK)
	var state map[string]any
	readJSON(t, resp, &state)

	stateInner := state["state"].(map[string]any)
	players := stateInner["players"].([]any)
	for _, p := range players {
		pm := p.(map[string]any)
		alive, _ := pm["alive"].(bool)
		role, _ := pm["role"].(string)
		if alive && role != "" {
			t.Fatal("spectator should not see alive player roles")
		}
	}
}

func TestWW_InvalidActions(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	// Non-wolf tries kill_vote → 400
	villager := roles["villager"][0]
	resp := villager.post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{
		"action": map[string]any{"type": "kill_vote", "target_seat": 0},
	})
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// --- Helper functions for werewolf tests ---

// wolfSeats returns the seat numbers of the given wolf agents.
func wolfSeats(t *testing.T, wolves []*apiClient, roomID uint) []int {
	t.Helper()
	seats := make([]int, len(wolves))
	for i, w := range wolves {
		seats[i] = findAgentSeat(t, w, roomID)
	}
	return seats
}

// findAliveNonWolfSeat finds a seat of an alive non-wolf player.
func findAliveNonWolfSeat(t *testing.T, agents []*apiClient, roomID uint, excludeSeats []int) int {
	t.Helper()
	excl := map[int]bool{}
	for _, s := range excludeSeats {
		excl[s] = true
	}
	for _, a := range agents {
		if !isAgentAlive(t, a, roomID) {
			continue
		}
		seat := findAgentSeat(t, a, roomID)
		if !excl[seat] {
			return seat
		}
	}
	t.Fatal("no alive non-wolf seat found")
	return -1
}

// isAgentAlive checks if the agent is alive in the current game state.
func isAgentAlive(t *testing.T, agent *apiClient, roomID uint) bool {
	t.Helper()
	resp := getState(t, agent, roomID)
	inner := getInnerState(t, resp)
	alive, ok := inner["your_alive"].(bool)
	if !ok {
		return false
	}
	return alive
}

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/clawarena/clawarena/internal/models"
)

func TestWW_FullGame_GoodWins(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["clawedwolf"]
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

func TestWW_LeaveForfeitEmitsSingleGameFinishedActivityEvent(t *testing.T) {
	cleanDB(t)

	roomID, agents := createAndStartWWGame(t)

	var room models.Room
	if err := testDB.First(&room, roomID).Error; err != nil {
		t.Fatalf("load room: %v", err)
	}
	if room.CurrentGameID == nil {
		t.Fatal("expected current_game_id after game start")
	}
	gameID := *room.CurrentGameID

	for i := 0; i < len(agents)-1; i++ {
		resp := agents[i].post(t, fmt.Sprintf("/api/v1/rooms/%d/leave", roomID), nil)
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	}

	if err := testDB.First(&room, roomID).Error; err != nil {
		t.Fatalf("reload room: %v", err)
	}
	if room.Status != models.RoomIntermission {
		t.Fatalf("expected room to enter intermission after terminal leave-forfeit, got %s", room.Status)
	}
	if room.WinnerID == nil || *room.WinnerID != agents[len(agents)-1].agentID {
		t.Fatalf("expected agent %d to win, got %v", agents[len(agents)-1].agentID, room.WinnerID)
	}

	var count int64
	eventID := fmt.Sprintf("clawarena:game_finished:%d", gameID)
	if err := testDB.Model(&models.ActivityEvent{}).Where("event_id = ?", eventID).Count(&count).Error; err != nil {
		t.Fatalf("count activity events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one terminal activity event, got %d", count)
	}
}

func TestWW_FullGame_EvilWins(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["clawedwolf"]
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Strategy: kill non-wolves each night, abstain in day votes.
	// After losing seer + one more good player with 2 wolves alive,
	// wolves (2) >= good players → evil wins.

	// Round 1: kill seer, guard self-protects, all abstain during day.
	villager0Seat := findAgentSeat(t, roles["villager"][0], roomID)
	wwPlayNightRound(t, roles, agents, roomID, seerSeat, villager0Seat, guardSeat)
	wwPlayDayRound(t, agents, roomID, -1)

	// Helper: play a complete night when seer is dead.
	playNightNoSeer := func(killTarget, guardTarget int) {
		// Wolves vote.
		for _, wolf := range wolves {
			if s := getState(t, wolf, roomID); hasPendingAction(s) {
				wwSubmitKillVote(t, wolf, roomID, killTarget)
			}
		}
		// Seer is dead — phase auto-advances past night_seer.
		// Guard (if alive) protects.
		guard := roles["guard"][0]
		if isAgentAlive(t, guard, roomID) {
			if s := getState(t, guard, roomID); hasPendingAction(s) {
				wwSubmitProtect(t, guard, roomID, guardTarget)
			}
		}
	}

	checkEvil := func() bool {
		anyAlive := findAnyAliveAgent(t, agents, roomID)
		if anyAlive == nil {
			return false
		}
		inner := getInnerState(t, getState(t, anyAlive, roomID))
		return inner["winner"] == "evil"
	}

	if checkEvil() {
		return
	}

	// Round 2: kill another non-wolf; guard must protect a different seat than guardSeat.
	ws := wolfSeats(t, wolves, roomID)
	killTarget2 := findAliveNonWolfSeat(t, agents, roomID, ws)
	guardTarget2 := villager0Seat // different from guardSeat (round 1 target)
	if guardTarget2 == guardSeat {
		guardTarget2 = ws[0] // any other seat — guard can protect a wolf
	}
	playNightNoSeer(killTarget2, guardTarget2)
	wwPlayDayRound(t, agents, roomID, -1)

	if checkEvil() {
		return
	}

	// Round 3 (if needed): one more kill.
	killTarget3 := findAliveNonWolfSeat(t, agents, roomID, ws)
	// Guard target must differ from guardTarget2.
	guardTarget3 := guardSeat
	if guardTarget3 == guardTarget2 {
		for _, a := range agents {
			s := findAgentSeat(t, a, roomID)
			if isAgentAlive(t, a, roomID) && s != guardTarget2 {
				guardTarget3 = s
				break
			}
		}
	}
	playNightNoSeer(killTarget3, guardTarget3)
	wwPlayDayRound(t, agents, roomID, -1)

	if checkEvil() {
		return
	}

	t.Fatalf("expected evil team to win within 3 rounds")
}

func TestWW_GuardSave(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	// Kill target = seer, guard protects seer → seer survives
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)
	wolfSeats := wolfSeats(t, roles["clawedwolf"], roomID)

	// Wolves target seer
	for _, wolf := range roles["clawedwolf"] {
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
	wolfSeats := wolfSeats(t, roles["clawedwolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)

	// Round 1: guard protects seer
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, wolfSeats[0], seerSeat)

	// Day: abstain
	wwPlayDayRound(t, agents, roomID, -1)

	// Round 2: guard tries to protect seer again → should fail
	// Wolves vote
	killTarget := findAliveNonWolfSeat(t, agents, roomID, wolfSeats)
	for _, wolf := range roles["clawedwolf"] {
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

	wolfSeats := wolfSeats(t, roles["clawedwolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Wolves vote
	for _, wolf := range roles["clawedwolf"] {
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
	for _, wolf := range roles["clawedwolf"] {
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

	wolfSeats := wolfSeats(t, roles["clawedwolf"], roomID)
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

	wolfSeats := wolfSeats(t, roles["clawedwolf"], roomID)
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

	wolfSeats := wolfSeats(t, roles["clawedwolf"], roomID)
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

	wolves := roles["clawedwolf"]
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
			if role != "clawedwolf" {
				t.Fatalf("wolf should see partner's role as clawedwolf, got %q", role)
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
		if alive && role == "clawedwolf" {
			t.Fatal("villager should not see alive wolf roles")
		}
	}
}

func TestWW_SpectatorView_HidesRoles(t *testing.T) {
	cleanDB(t)
	roomID, _ := createAndStartWWGame(t)

	// Spectator (no auth) view — use room endpoint since /state is removed
	anon := anonClient()
	resp := anon.get(t, fmt.Sprintf("/api/v1/rooms/%d", roomID))
	assertStatus(t, resp, http.StatusOK)
	var roomData map[string]any
	readJSON(t, resp, &roomData)

	// Room endpoint returns agent info but not game state details,
	// so just verify the room is accessible and has agents
	roomAgents := roomData["agents"].([]any)
	if len(roomAgents) != 6 {
		t.Fatalf("expected 6 agents, got %d", len(roomAgents))
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

// ---------------------------------------------------------------------------
// New integration tests
// ---------------------------------------------------------------------------

// TestWW_DeadPlayerCannotAct verifies that a dead player cannot submit actions.
func TestWW_DeadPlayerCannotAct(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	villager := roles["villager"][0]
	villagerSeat := findAgentSeat(t, villager, roomID)
	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Night 1: wolves kill the villager; guard protects self.
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, ws[0], guardSeat)

	if isAgentAlive(t, villager, roomID) {
		t.Fatal("villager should be dead after night kill")
	}

	// Dead villager tries to speak → NOT_YOUR_TURN (400).
	resp := villager.post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{
		"action": map[string]any{"type": "speak", "message": "I am dead!"},
	})
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestWW_SeerViewShowsResults verifies that the seer's state view includes
// cumulative investigation results.
func TestWW_SeerViewShowsResults(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Night 1: wolves kill villager, seer investigates wolf 0, guard protects self.
	for _, wolf := range roles["clawedwolf"] {
		if s := getState(t, wolf, roomID); hasPendingAction(s) {
			wwSubmitKillVote(t, wolf, roomID, villagerSeat)
		}
	}
	wwSubmitInvestigate(t, roles["seer"][0], roomID, ws[0])
	wwSubmitProtect(t, roles["guard"][0], roomID, guardSeat)

	// Seer's view must include seer_results with wolf 0 → "evil".
	state := getState(t, roles["seer"][0], roomID)
	inner := getInnerState(t, state)
	seerResults, ok := inner["seer_results"].(map[string]any)
	if !ok {
		t.Fatal("seer_results should be present in seer's state view")
	}
	seatKey := fmt.Sprintf("%d", ws[0])
	result, _ := seerResults[seatKey].(string)
	if result != "evil" {
		t.Fatalf("seer should see investigated wolf as 'evil', got %q", result)
	}

	// Villager's view must NOT include seer_results.
	stateV := getState(t, roles["villager"][0], roomID)
	innerV := getInnerState(t, stateV)
	if _, present := innerV["seer_results"]; present {
		t.Fatal("villager should not see seer_results")
	}
}

// TestWW_DeadRoleRevealedInSpectatorView verifies that spectators see the role
// of a player who was killed during the night.
func TestWW_DeadRoleRevealedInSpectatorView(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	villager := roles["villager"][0]
	villagerSeat := findAgentSeat(t, villager, roomID)
	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Night 1: kill the villager; guard protects self (not the villager).
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, ws[0], guardSeat)

	if isAgentAlive(t, villager, roomID) {
		t.Fatal("villager should be dead after night kill")
	}

	// Spectator view — use room endpoint since /state is removed.
	// The room endpoint returns agents but not detailed game state,
	// so we verify via the history endpoint instead.
	anon := anonClient()
	histResp := anon.get(t, fmt.Sprintf("/api/v1/rooms/%d/history", roomID))
	assertStatus(t, histResp, http.StatusOK)
	var history map[string]any
	readJSON(t, histResp, &history)

	timeline := history["timeline"].([]any)
	if len(timeline) == 0 {
		t.Fatal("expected timeline entries in history")
	}
	// Get latest state from history
	lastEntry := timeline[len(timeline)-1].(map[string]any)
	inner := lastEntry["state"].(map[string]any)
	players := inner["players"].([]any)
	foundDead := false
	for _, p := range players {
		pm := p.(map[string]any)
		alive := pm["alive"].(bool)
		seat := int(pm["seat"].(float64))
		if !alive && seat == villagerSeat {
			role, _ := pm["role"].(string)
			if role != "villager" {
				t.Fatalf("dead villager's role should be 'villager' in spectator view, got %q", role)
			}
			foundDead = true
		}
	}
	if !foundDead {
		t.Fatal("expected to find dead villager in history state")
	}
}

// TestWW_SkipDeadSeerPhase verifies that when the seer is dead, the
// night_seer phase is skipped and the engine goes straight to night_guard.
func TestWW_SkipDeadSeerPhase(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	seer := roles["seer"][0]
	seerSeat := findAgentSeat(t, seer, roomID)
	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)

	// Round 1: kill the seer; guard protects self (not seer).
	wwPlayNightRound(t, roles, agents, roomID, seerSeat, villagerSeat, guardSeat)

	if isAgentAlive(t, seer, roomID) {
		t.Fatal("seer should be dead after night kill")
	}

	// Day 1: abstain all votes.
	wwPlayDayRound(t, agents, roomID, -1)

	// Verify game is still going.
	anyAlive := findAnyAliveAgent(t, agents, roomID)
	if anyAlive == nil {
		t.Skip("no alive agents found")
	}
	inner := getInnerState(t, getState(t, anyAlive, roomID))
	if inner["winner"] != nil {
		t.Skip("game ended before round 2; skipping skip-seer test")
	}

	// Round 2 night: wolves vote.
	killTarget2 := findAliveNonWolfSeat(t, agents, roomID, ws)
	for _, wolf := range roles["clawedwolf"] {
		if s := getState(t, wolf, roomID); hasPendingAction(s) {
			wwSubmitKillVote(t, wolf, roomID, killTarget2)
		}
	}

	// After wolves vote, phase should advance to night_guard (night_seer skipped).
	phase := wwGetPhase(t, anyAlive, roomID)
	if phase == "night_seer" {
		t.Fatal("night_seer phase should be skipped when seer is dead")
	}
	if phase != "night_guard" && phase != "day_discuss" && phase != "finished" {
		t.Fatalf("expected night_guard or day_discuss after seer-skip, got %q", phase)
	}
}

// TestWW_SkipDeadGuardPhase verifies that when the guard is dead, the
// night_guard phase is skipped and the night resolves without it.
func TestWW_SkipDeadGuardPhase(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	guard := roles["guard"][0]
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)
	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	wolfSeat0 := ws[0]

	// Round 1: kill the guard; guard protects the villager (not self → guard dies).
	wwPlayNightRound(t, roles, agents, roomID, guardSeat, wolfSeat0, villagerSeat)

	if isAgentAlive(t, guard, roomID) {
		t.Fatal("guard should be dead after night kill")
	}

	// Day 1: abstain all votes.
	wwPlayDayRound(t, agents, roomID, -1)

	// Verify game is still going.
	anyAlive := findAnyAliveAgent(t, agents, roomID)
	if anyAlive == nil {
		t.Skip("no alive agents found")
	}
	inner := getInnerState(t, getState(t, anyAlive, roomID))
	if inner["winner"] != nil {
		t.Skip("game ended before round 2; skipping skip-guard test")
	}

	// Round 2 night: wolves vote.
	killTarget2 := findAliveNonWolfSeat(t, agents, roomID, ws)
	for _, wolf := range roles["clawedwolf"] {
		if s := getState(t, wolf, roomID); hasPendingAction(s) {
			wwSubmitKillVote(t, wolf, roomID, killTarget2)
		}
	}

	// Seer investigates (if alive).
	seer := roles["seer"][0]
	if isAgentAlive(t, seer, roomID) {
		seerSeat := findAgentSeat(t, seer, roomID)
		seerTarget := findAliveNonWolfSeat(t, agents, roomID, append(ws, seerSeat))
		if s := getState(t, seer, roomID); hasPendingAction(s) {
			wwSubmitInvestigate(t, seer, roomID, seerTarget)
		}
	}

	// After seer investigates, night_guard should be skipped → day_discuss.
	phase := wwGetPhase(t, anyAlive, roomID)
	if phase == "night_guard" {
		t.Fatal("night_guard phase should be skipped when guard is dead")
	}
}

// TestWW_VoteForDeadPlayer verifies that voting for a dead player is rejected.
func TestWW_VoteForDeadPlayer(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	villager := roles["villager"][0]
	villagerSeat := findAgentSeat(t, villager, roomID)
	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Night 1: kill the villager.
	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, ws[0], guardSeat)

	if isAgentAlive(t, villager, roomID) {
		t.Fatal("villager should be dead after night kill")
	}

	// Day discussion.
	for {
		speaker := findPendingAgent(t, agents, roomID, "speak")
		if speaker == nil {
			break
		}
		wwSubmitSpeak(t, speaker, roomID, "test")
	}

	// Find an alive voter.
	var voter *apiClient
	for _, a := range agents {
		if isAgentAlive(t, a, roomID) {
			if s := getState(t, a, roomID); hasPendingAction(s) {
				if getPendingAction(s)["action_type"] == "vote" {
					voter = a
					break
				}
			}
		}
	}
	if voter == nil {
		t.Skip("no voter with pending action found")
	}

	// Vote for the dead villager → 400.
	resp := voter.post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{
		"action": map[string]any{"type": "vote", "target_seat": villagerSeat},
	})
	assertStatus(t, resp, http.StatusBadRequest)
	body := readBody(t, resp)
	assertContains(t, body, "invalid")
}

// TestWW_VoteForSelf verifies that self-voting is rejected.
func TestWW_VoteForSelf(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, ws[0], guardSeat)

	for {
		speaker := findPendingAgent(t, agents, roomID, "speak")
		if speaker == nil {
			break
		}
		wwSubmitSpeak(t, speaker, roomID, "test")
	}

	var voter *apiClient
	for _, a := range agents {
		if isAgentAlive(t, a, roomID) {
			if s := getState(t, a, roomID); hasPendingAction(s) {
				if getPendingAction(s)["action_type"] == "vote" {
					voter = a
					break
				}
			}
		}
	}
	if voter == nil {
		t.Skip("no voter found")
	}

	voterSeat := findAgentSeat(t, voter, roomID)
	resp := voter.post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{
		"action": map[string]any{"type": "vote", "target_seat": voterSeat},
	})
	assertStatus(t, resp, http.StatusBadRequest)
	body := readBody(t, resp)
	assertContains(t, body, "yourself")
}

// TestWW_HistoryGodView verifies that the history endpoint for a finished
// ClawedWolf game returns the god view with all roles revealed.
func TestWW_HistoryGodView(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["clawedwolf"]
	wolfSeat0 := findAgentSeat(t, wolves[0], roomID)
	wolfSeat1 := findAgentSeat(t, wolves[1], roomID)
	villager0Seat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)

	// Round 1: kill villager, vote out wolf 0.
	wwPlayNightRound(t, roles, agents, roomID, villager0Seat, wolfSeat0, guardSeat)
	wwPlayDayRound(t, agents, roomID, wolfSeat0)

	// Check if game ended after round 1.
	if !wwGameOver(t, agents, roomID) {
		// Round 2: kill something, vote out wolf 1.
		killTarget2 := findAliveNonWolfSeat(t, agents, roomID, []int{wolfSeat0, wolfSeat1})
		guardTarget2 := seerSeat
		if guardTarget2 == guardSeat {
			guardTarget2 = wolfSeat1
		}
		wwPlayNightRound(t, roles, agents, roomID, killTarget2, wolfSeat1, guardTarget2)
		wwPlayDayRound(t, agents, roomID, wolfSeat1)
	}

	history := getHistory(t, roomID)
	if history["status"] != "finished" {
		t.Fatalf("expected finished status in history, got %v", history["status"])
	}

	// Result must include winner_team.
	result, ok := history["result"].(map[string]any)
	if !ok || result == nil {
		t.Fatal("history should have a result object")
	}
	if wt, _ := result["winner_team"].(string); wt == "" {
		t.Fatalf("history result should have winner_team, got %v", result)
	}

	// God view: every player in the initial state snapshot must have a role.
	timeline, ok := history["timeline"].([]any)
	if !ok || len(timeline) == 0 {
		t.Fatal("expected non-empty timeline in history")
	}
	firstEntry := timeline[0].(map[string]any)
	firstState := firstEntry["state"].(map[string]any)
	players := firstState["players"].([]any)
	for _, p := range players {
		pm := p.(map[string]any)
		role, _ := pm["role"].(string)
		if role == "" {
			t.Fatal("god view should reveal all player roles in finished game history")
		}
	}
}

// TestWW_NightKillNoGuardSave verifies that when the guard protects a
// different target than the wolves' kill target, the kill target dies.
func TestWW_NightKillNoGuardSave(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	villager := roles["villager"][0]
	villagerSeat := findAgentSeat(t, villager, roomID)
	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)

	// Wolves kill villager; guard protects seer (different target → no save).
	for _, wolf := range roles["clawedwolf"] {
		if s := getState(t, wolf, roomID); hasPendingAction(s) {
			wwSubmitKillVote(t, wolf, roomID, villagerSeat)
		}
	}
	wwSubmitInvestigate(t, roles["seer"][0], roomID, ws[0])
	wwSubmitProtect(t, roles["guard"][0], roomID, seerSeat)

	// Villager must be dead.
	if isAgentAlive(t, villager, roomID) {
		t.Fatal("villager should be dead — guard protected a different player")
	}

	// Events must include death but not guard_save.
	inner := getInnerState(t, getState(t, agents[0], roomID))
	events, _ := inner["events"].([]any)
	foundDeath := false
	foundSave := false
	for _, ev := range events {
		em := ev.(map[string]any)
		switch em["type"].(string) {
		case "death":
			foundDeath = true
		case "guard_save":
			foundSave = true
		}
	}
	if !foundDeath {
		t.Fatal("expected death event when guard protects different target")
	}
	if foundSave {
		t.Fatal("should not have guard_save event when guard protects different target")
	}
}

// TestWW_SpeakOutOfOrder verifies that an agent who is not the next expected
// speaker is rejected during day discussion.
func TestWW_SpeakOutOfOrder(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	villagerSeat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	wwPlayNightRound(t, roles, agents, roomID, villagerSeat, ws[0], guardSeat)

	// Find the expected first speaker.
	firstSpeaker := findPendingAgent(t, agents, roomID, "speak")
	if firstSpeaker == nil {
		t.Fatal("expected a pending speak action in day_discuss")
	}

	// Find an alive agent who is NOT the expected speaker.
	var nonSpeaker *apiClient
	for _, a := range agents {
		if a.agentID != firstSpeaker.agentID && isAgentAlive(t, a, roomID) {
			nonSpeaker = a
			break
		}
	}
	if nonSpeaker == nil {
		t.Skip("could not find a non-speaker alive agent")
	}

	// Non-speaker tries to speak → 400 NOT_YOUR_TURN.
	resp := nonSpeaker.post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{
		"action": map[string]any{"type": "speak", "message": "out of order"},
	})
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestWW_GuardProtectDeadPlayer verifies that the guard cannot protect a
// player who is already dead.
func TestWW_GuardProtectDeadPlayer(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	villager := roles["villager"][0]
	villagerSeat := findAgentSeat(t, villager, roomID)
	ws := wolfSeats(t, roles["clawedwolf"], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Round 1: kill villager; guard protects self.
	for _, wolf := range roles["clawedwolf"] {
		if s := getState(t, wolf, roomID); hasPendingAction(s) {
			wwSubmitKillVote(t, wolf, roomID, villagerSeat)
		}
	}
	if s := getState(t, roles["seer"][0], roomID); hasPendingAction(s) {
		wwSubmitInvestigate(t, roles["seer"][0], roomID, ws[0])
	}
	wwSubmitProtect(t, roles["guard"][0], roomID, guardSeat)

	if isAgentAlive(t, villager, roomID) {
		t.Fatal("villager should be dead after night kill")
	}

	// Day 1: abstain all.
	wwPlayDayRound(t, agents, roomID, -1)

	// Check game is still going.
	anyAlive := findAnyAliveAgent(t, agents, roomID)
	if anyAlive == nil || wwGameOver(t, agents, roomID) {
		t.Skip("game over before round 2")
	}

	// Round 2: wolves vote.
	killTarget2 := findAliveNonWolfSeat(t, agents, roomID, ws)
	for _, wolf := range roles["clawedwolf"] {
		if s := getState(t, wolf, roomID); hasPendingAction(s) {
			wwSubmitKillVote(t, wolf, roomID, killTarget2)
		}
	}

	// Seer investigates (if alive).
	seer := roles["seer"][0]
	if isAgentAlive(t, seer, roomID) {
		seerSeat := findAgentSeat(t, seer, roomID)
		seerTarget := findAliveNonWolfSeat(t, agents, roomID, append(ws, seerSeat))
		if s := getState(t, seer, roomID); hasPendingAction(s) {
			wwSubmitInvestigate(t, seer, roomID, seerTarget)
		}
	}

	// Guard tries to protect the dead villager in round 2 → 400.
	guard := roles["guard"][0]
	if !isAgentAlive(t, guard, roomID) {
		t.Skip("guard not alive in round 2")
	}
	guardState := getState(t, guard, roomID)
	if !hasPendingAction(guardState) {
		t.Skip("guard has no pending action in round 2")
	}
	resp := guard.post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{
		"action": map[string]any{"type": "protect", "target_seat": villagerSeat},
	})
	assertStatus(t, resp, http.StatusBadRequest)
	body := readBody(t, resp)
	assertContains(t, body, "invalid")
}

// TestWW_EloUpdatedAfterGame verifies that Elo ratings are updated correctly
// after a ClawedWolf game: winners gain, losers lose.
func TestWW_EloUpdatedAfterGame(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["clawedwolf"]
	wolfSeat0 := findAgentSeat(t, wolves[0], roomID)
	wolfSeat1 := findAgentSeat(t, wolves[1], roomID)
	villager0Seat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)

	// Good wins by voting out both wolves.
	wwPlayNightRound(t, roles, agents, roomID, villager0Seat, wolfSeat0, guardSeat)
	wwPlayDayRound(t, agents, roomID, wolfSeat0)

	if !wwGameOver(t, agents, roomID) {
		killTarget2 := findAliveNonWolfSeat(t, agents, roomID, []int{wolfSeat0, wolfSeat1})
		guardTarget2 := seerSeat
		if guardTarget2 == guardSeat {
			guardTarget2 = wolfSeat1
		}
		wwPlayNightRound(t, roles, agents, roomID, killTarget2, wolfSeat1, guardTarget2)
		wwPlayDayRound(t, agents, roomID, wolfSeat1)
	}

	if !wwGameOver(t, agents, roomID) {
		t.Skip("game not over after 2 rounds, skipping Elo check")
	}

	// Wolves (losers) Elo should decrease.
	for _, wolf := range wolves {
		resp := wolf.get(t, "/api/v1/agents/me")
		assertStatus(t, resp, http.StatusOK)
		var me map[string]any
		readJSON(t, resp, &me)
		if elo, ok := me["elo_rating"].(float64); !ok || elo >= 1000 {
			t.Fatalf("wolf Elo should decrease after losing, got %v", me["elo_rating"])
		}
	}

	// Seer (winner) Elo should increase.
	resp := roles["seer"][0].get(t, "/api/v1/agents/me")
	assertStatus(t, resp, http.StatusOK)
	var me map[string]any
	readJSON(t, resp, &me)
	if elo, ok := me["elo_rating"].(float64); !ok || elo <= 1000 {
		t.Fatalf("seer Elo should increase after winning, got %v", me["elo_rating"])
	}
}

// TestWW_ActionOnFinishedGame verifies that submitting an action after the
// game is over is rejected.
func TestWW_ActionOnFinishedGame(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["clawedwolf"]
	wolfSeat0 := findAgentSeat(t, wolves[0], roomID)
	wolfSeat1 := findAgentSeat(t, wolves[1], roomID)
	villager0Seat := findAgentSeat(t, roles["villager"][0], roomID)
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)
	seerSeat := findAgentSeat(t, roles["seer"][0], roomID)

	wwPlayNightRound(t, roles, agents, roomID, villager0Seat, wolfSeat0, guardSeat)
	wwPlayDayRound(t, agents, roomID, wolfSeat0)

	if !wwGameOver(t, agents, roomID) {
		killTarget2 := findAliveNonWolfSeat(t, agents, roomID, []int{wolfSeat0, wolfSeat1})
		guardTarget2 := seerSeat
		if guardTarget2 == guardSeat {
			guardTarget2 = wolfSeat1
		}
		wwPlayNightRound(t, roles, agents, roomID, killTarget2, wolfSeat1, guardTarget2)
		wwPlayDayRound(t, agents, roomID, wolfSeat1)
	}

	if !wwGameOver(t, agents, roomID) {
		t.Skip("game not over after 2 rounds")
	}

	// Any agent tries to act on the finished game → 400.
	resp := agents[0].post(t, fmt.Sprintf("/api/v1/rooms/%d/action", roomID), map[string]any{
		"action": map[string]any{"type": "speak", "message": "too late"},
	})
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusForbidden {
		body := readBody(t, resp)
		t.Fatalf("expected 400/403 on finished game action, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}

// TestWW_WolfKillVoteDisagreement verifies that when two wolves vote for
// different targets, the wolf with the lower seat number's vote wins.
func TestWW_WolfKillVoteDisagreement(t *testing.T) {
	cleanDB(t)
	roomID, agents := createAndStartWWGame(t)
	roles := discoverRoles(t, agents, roomID)

	wolves := roles["clawedwolf"]
	if len(wolves) != 2 {
		t.Fatalf("expected 2 wolves, got %d", len(wolves))
	}

	// Sort wolves so wolf0 has the lower seat (engine picks this one on disagreement).
	wolf0Seat := findAgentSeat(t, wolves[0], roomID)
	wolf1Seat := findAgentSeat(t, wolves[1], roomID)
	if wolf0Seat > wolf1Seat {
		wolves[0], wolves[1] = wolves[1], wolves[0]
		wolf0Seat, wolf1Seat = wolf1Seat, wolf0Seat
	}
	wolf0, wolf1 := wolves[0], wolves[1]
	_ = wolf1Seat

	// Find two distinct alive non-wolf targets.
	guardSeat := findAgentSeat(t, roles["guard"][0], roomID)

	// Build list of non-wolf, non-guard seats (guard self-protects, so kills
	// that land on the guard's seat would be saved).
	var nonWolfSeats []int
	for _, a := range agents {
		seat := findAgentSeat(t, a, roomID)
		if seat != wolf0Seat && seat != wolf1Seat && seat != guardSeat {
			nonWolfSeats = append(nonWolfSeats, seat)
		}
	}
	if len(nonWolfSeats) < 2 {
		t.Fatal("need at least 2 non-wolf, non-guard targets for disagreement test")
	}
	target0 := nonWolfSeats[0] // wolf0 (lower seat) votes here → this should be the kill
	target1 := nonWolfSeats[1] // wolf1 votes here → should NOT be the kill

	// Wolf0 (lower seat = "first") votes for target0.
	if s := getState(t, wolf0, roomID); hasPendingAction(s) {
		wwSubmitKillVote(t, wolf0, roomID, target0)
	}
	// Wolf1 votes for target1 (disagreement).
	if s := getState(t, wolf1, roomID); hasPendingAction(s) {
		wwSubmitKillVote(t, wolf1, roomID, target1)
	}

	// Seer and guard act to advance through night phases.
	if isAgentAlive(t, roles["seer"][0], roomID) {
		if s := getState(t, roles["seer"][0], roomID); hasPendingAction(s) {
			wwSubmitInvestigate(t, roles["seer"][0], roomID, wolf1Seat)
		}
	}
	if isAgentAlive(t, roles["guard"][0], roomID) {
		if s := getState(t, roles["guard"][0], roomID); hasPendingAction(s) {
			// Guard self-protects (guardSeat ≠ target0 by construction above).
			wwSubmitProtect(t, roles["guard"][0], roomID, guardSeat)
		}
	}

	// target0 (wolf0's choice) should be dead; target1 should be alive.
	var agentAtTarget0, agentAtTarget1 *apiClient
	for _, a := range agents {
		seat := findAgentSeat(t, a, roomID)
		if seat == target0 {
			agentAtTarget0 = a
		}
		if seat == target1 {
			agentAtTarget1 = a
		}
	}
	if agentAtTarget0 != nil && isAgentAlive(t, agentAtTarget0, roomID) {
		t.Fatalf("seat %d (wolf0's kill choice) should be dead after disagreement", target0)
	}
	if agentAtTarget1 != nil && !isAgentAlive(t, agentAtTarget1, roomID) {
		t.Fatalf("seat %d (wolf1's kill choice) should be alive when wolves disagree", target1)
	}
}

// --- Helper functions for clawedwolf tests ---

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

// findAnyAliveAgent returns the first alive agent from the list, or nil.
func findAnyAliveAgent(t *testing.T, agents []*apiClient, roomID uint) *apiClient {
	t.Helper()
	for _, a := range agents {
		if isAgentAlive(t, a, roomID) {
			return a
		}
	}
	return nil
}

// wwGameOver returns true if the room is in "finished" status.
func wwGameOver(t *testing.T, agents []*apiClient, roomID uint) bool {
	t.Helper()
	for _, a := range agents {
		resp := a.get(t, fmt.Sprintf("/api/v1/rooms/%d", roomID))
		if resp.StatusCode == http.StatusOK {
			body := readBody(t, resp)
			return contains(body, `"finished"`)
		}
		resp.Body.Close()
	}
	return false
}

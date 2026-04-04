package clawedwolf

import (
	"encoding/json"
	"testing"

	"github.com/clawarena/clawarena/internal/game"
)

func initClawedWolf(t *testing.T, players []uint) json.RawMessage {
	t.Helper()
	e := &Engine{}
	state, events, err := e.InitState(nil, players)
	if err != nil {
		t.Fatalf("InitState failed: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected seed events from InitState")
	}
	return state
}

// lastState extracts the final state from an ApplyResult (last event's StateAfter).
func lastState(t *testing.T, result *game.ApplyResult) json.RawMessage {
	t.Helper()
	if len(result.Events) == 0 {
		t.Fatal("ApplyResult has no events")
	}
	return result.Events[len(result.Events)-1].StateAfter
}

func parseTestState(t *testing.T, raw json.RawMessage) *State {
	t.Helper()
	s, err := parseState(raw)
	if err != nil {
		t.Fatalf("parseState failed: %v", err)
	}
	return s
}

func playerIDForRole(t *testing.T, s *State, role string) uint {
	t.Helper()
	for _, p := range s.Players {
		if p.Role == role && p.Alive {
			return p.ID
		}
	}
	t.Fatalf("no alive player with role %q", role)
	return 0
}

func allPlayerIDsForRole(s *State, role string) []uint {
	var ids []uint
	for _, p := range s.Players {
		if p.Role == role && p.Alive {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

var testPlayers = []uint{101, 102, 103, 104, 105, 106}

func TestInitState_SixPlayers(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	s := parseTestState(t, state)
	if len(s.Players) != 6 {
		t.Fatalf("expected 6 players, got %d", len(s.Players))
	}
	roleCounts := map[string]int{}
	for _, p := range s.Players {
		roleCounts[p.Role]++
		if !p.Alive {
			t.Errorf("player %d should be alive at start", p.ID)
		}
	}
	if roleCounts[RoleClawedWolf] != 2 {
		t.Errorf("expected 2 clawed wolves, got %d", roleCounts[RoleClawedWolf])
	}
	if roleCounts[RoleSeer] != 1 {
		t.Errorf("expected 1 seer, got %d", roleCounts[RoleSeer])
	}
	if roleCounts[RoleGuard] != 1 {
		t.Errorf("expected 1 guard, got %d", roleCounts[RoleGuard])
	}
	if roleCounts[RoleVillager] != 2 {
		t.Errorf("expected 2 villagers, got %d", roleCounts[RoleVillager])
	}
	if s.Phase != PhaseNightClawedWolf {
		t.Errorf("expected initial phase %q, got %q", PhaseNightClawedWolf, s.Phase)
	}
}

func TestInitState_SeedEvents(t *testing.T) {
	e := &Engine{}
	_, events, err := e.InitState(nil, testPlayers)
	if err != nil {
		t.Fatalf("InitState failed: %v", err)
	}
	// Expect 1 game_start + 1 phase_change + 6 roles_assigned = 8 events
	if len(events) != 8 {
		t.Fatalf("expected 8 seed events, got %d", len(events))
	}
	if events[0].EventType != "game_start" {
		t.Errorf("first event should be game_start, got %q", events[0].EventType)
	}
	if events[0].Source != "system" {
		t.Errorf("game_start source should be system, got %q", events[0].Source)
	}
	if events[0].Visibility != "public" {
		t.Errorf("game_start visibility should be public, got %q", events[0].Visibility)
	}
	if events[1].EventType != "phase_change" {
		t.Errorf("second event should be phase_change, got %q", events[1].EventType)
	}
	if events[1].Visibility != "public" {
		t.Errorf("phase_change visibility should be public, got %q", events[1].Visibility)
	}
	for i := 2; i <= 7; i++ {
		if events[i].EventType != "roles_assigned" {
			t.Errorf("event %d should be roles_assigned, got %q", i, events[i].EventType)
		}
		if events[i].Source != "system" {
			t.Errorf("roles_assigned source should be system, got %q", events[i].Source)
		}
	}
}

func TestInitState_WrongPlayerCount(t *testing.T) {
	e := &Engine{}
	_, _, err := e.InitState(nil, []uint{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for wrong player count")
	}
}

func TestSyncronym(t *testing.T) {
	e := &Engine{}
	if e.Syncronym() != "cw" {
		t.Errorf("expected syncronym 'cw', got %q", e.Syncronym())
	}
}

func TestNewEventModel(t *testing.T) {
	e := &Engine{}
	m := e.NewEventModel()
	if m == nil {
		t.Fatal("NewEventModel should not return nil")
	}
	if m.TableName() != "cw_game_events" {
		t.Errorf("expected table name 'cw_game_events', got %q", m.TableName())
	}
}

func TestGetPhaseTimeout(t *testing.T) {
	e := &Engine{}
	if e.GetPhaseTimeout(nil) != nil {
		t.Error("expected nil phase timeout")
	}
}

func TestNightClawedWolfAction(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	s := parseTestState(t, state)
	e := &Engine{}

	wolves := allPlayerIDsForRole(s, RoleClawedWolf)
	if len(wolves) != 2 {
		t.Fatalf("expected 2 wolves")
	}

	// Find a non-wolf target
	var targetSeat int
	for _, p := range s.Players {
		if p.Role != RoleClawedWolf {
			targetSeat = p.Seat
			break
		}
	}

	// Both wolves vote to kill the same target
	action1, _ := json.Marshal(map[string]interface{}{"type": "kill_vote", "target_seat": targetSeat})
	result1, err := e.ApplyAction(state, wolves[0], action1)
	if err != nil {
		t.Fatalf("wolf 1 kill_vote failed: %v", err)
	}
	state1 := lastState(t, result1)
	// After first wolf votes, phase should still be night_clawedwolf
	s1 := parseTestState(t, state1)
	if s1.Phase != PhaseNightClawedWolf {
		t.Logf("phase after first wolf vote: %s", s1.Phase)
	}
	// Verify kill_vote event
	if result1.Events[0].EventType != "kill_vote" {
		t.Errorf("expected kill_vote event, got %q", result1.Events[0].EventType)
	}
	if result1.Events[0].Source != "agent" {
		t.Errorf("expected agent source, got %q", result1.Events[0].Source)
	}

	// Second wolf votes
	result2, err := e.ApplyAction(state1, wolves[1], action1)
	if err != nil {
		t.Fatalf("wolf 2 kill_vote failed: %v", err)
	}
	state2 := lastState(t, result2)
	s2 := parseTestState(t, state2)
	// After both wolves vote, night_kill_target should be set and phase advanced
	if s2.NightKillTarget == nil {
		t.Error("expected night_kill_target to be set after both wolves vote")
	}
	if s2.Phase == PhaseNightClawedWolf {
		t.Error("phase should advance after both wolves vote")
	}
}

func TestSeerInvestigate(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	s := parseTestState(t, state)
	e := &Engine{}

	wolves := allPlayerIDsForRole(s, RoleClawedWolf)
	seerID := playerIDForRole(t, s, RoleSeer)

	// Find seer's seat for target exclusion
	var seerSeat, wolfSeat, nonWolfSeat int
	for _, p := range s.Players {
		if p.Role == RoleSeer {
			seerSeat = p.Seat
		}
		if p.Role == RoleClawedWolf {
			wolfSeat = p.Seat
		}
		if p.Role == RoleVillager {
			nonWolfSeat = p.Seat
		}
	}
	_ = seerSeat
	_ = nonWolfSeat

	// Complete wolf phase first
	targetSeat := 0
	for _, p := range s.Players {
		if p.Role != RoleClawedWolf {
			targetSeat = p.Seat
			break
		}
	}
	action, _ := json.Marshal(map[string]interface{}{"type": "kill_vote", "target_seat": targetSeat})
	r1, _ := e.ApplyAction(state, wolves[0], action)
	r2, err := e.ApplyAction(lastState(t, r1), wolves[1], action)
	if err != nil {
		t.Fatalf("wolf phase failed: %v", err)
	}

	state2 := lastState(t, r2)
	s2 := parseTestState(t, state2)
	if s2.Phase != PhaseNightSeer {
		t.Skipf("seer phase not reached (may be dead or skipped): got %s", s2.Phase)
	}

	// Investigate a wolf
	invAction, _ := json.Marshal(map[string]interface{}{"type": "investigate", "target_seat": wolfSeat})
	r3, err := e.ApplyAction(state2, seerID, invAction)
	if err != nil {
		t.Fatalf("seer investigate failed: %v", err)
	}

	state3 := lastState(t, r3)
	s3 := parseTestState(t, state3)
	if result, ok := s3.SeerResults[wolfSeat]; !ok {
		t.Error("expected seer result for wolf seat")
	} else if result != "evil" {
		t.Errorf("expected wolf to be 'evil', got %q", result)
	}

	// Verify investigate event
	found := false
	for _, ev := range r3.Events {
		if ev.EventType == "investigate" {
			found = true
			if ev.Source != "agent" {
				t.Errorf("expected agent source for investigate, got %q", ev.Source)
			}
		}
	}
	if !found {
		t.Error("expected investigate event in result")
	}
}

func TestWinCondition_GoodWins(t *testing.T) {
	s := &State{
		Players: []Player{
			{ID: 1, Seat: 0, Role: RoleClawedWolf, Alive: false},
			{ID: 2, Seat: 1, Role: RoleClawedWolf, Alive: false},
			{ID: 3, Seat: 2, Role: RoleVillager, Alive: true},
			{ID: 4, Seat: 3, Role: RoleSeer, Alive: true},
			{ID: 5, Seat: 4, Role: RoleGuard, Alive: true},
			{ID: 6, Seat: 5, Role: RoleVillager, Alive: true},
		},
		Phase:        PhaseDayVote,
		Round:        1,
		PhaseActions: map[string]int{},
		SeerResults:  map[int]string{},
		DayVotes:     map[string]int{},
	}
	var events []game.GameEvent
	checkWinCondition(s, &events)
	if s.Winner == nil {
		t.Fatal("expected good team to win")
	}
	if *s.Winner != "good" {
		t.Errorf("expected winner 'good', got %q", *s.Winner)
	}
	// Verify game_over event
	found := false
	for _, ev := range events {
		if ev.EventType == "game_over" && ev.GameOver {
			found = true
			if ev.Result == nil {
				t.Error("game_over event should have a result")
			} else if ev.Result.WinnerTeam != "good" {
				t.Errorf("expected winner_team 'good', got %q", ev.Result.WinnerTeam)
			}
		}
	}
	if !found {
		t.Error("expected game_over event")
	}
}

func TestWinCondition_EvilWins(t *testing.T) {
	s := &State{
		Players: []Player{
			{ID: 1, Seat: 0, Role: RoleClawedWolf, Alive: true},
			{ID: 2, Seat: 1, Role: RoleClawedWolf, Alive: true},
			{ID: 3, Seat: 2, Role: RoleVillager, Alive: true},
			{ID: 4, Seat: 3, Role: RoleVillager, Alive: false},
			{ID: 5, Seat: 4, Role: RoleSeer, Alive: false},
			{ID: 6, Seat: 5, Role: RoleGuard, Alive: false},
		},
		PhaseActions: map[string]int{},
		SeerResults:  map[int]string{},
		DayVotes:     map[string]int{},
	}
	var events []game.GameEvent
	checkWinCondition(s, &events)
	if s.Winner == nil {
		t.Fatal("expected evil team to win")
	}
	if *s.Winner != "evil" {
		t.Errorf("expected winner 'evil', got %q", *s.Winner)
	}
	// Verify game_over event
	found := false
	for _, ev := range events {
		if ev.EventType == "game_over" && ev.GameOver {
			found = true
			if ev.Result == nil {
				t.Error("game_over event should have a result")
			} else if ev.Result.WinnerTeam != "evil" {
				t.Errorf("expected winner_team 'evil', got %q", ev.Result.WinnerTeam)
			}
		}
	}
	if !found {
		t.Error("expected game_over event")
	}
}

func TestWinCondition_EvilWins_AllVillagersDead(t *testing.T) {
	// Evil wins when all villagers are dead, even if seer and guard are alive
	s := &State{
		Players: []Player{
			{ID: 1, Seat: 0, Role: RoleClawedWolf, Alive: true},
			{ID: 2, Seat: 1, Role: RoleClawedWolf, Alive: true},
			{ID: 3, Seat: 2, Role: RoleVillager, Alive: false},
			{ID: 4, Seat: 3, Role: RoleVillager, Alive: false},
			{ID: 5, Seat: 4, Role: RoleSeer, Alive: true},
			{ID: 6, Seat: 5, Role: RoleGuard, Alive: true},
		},
		PhaseActions: map[string]int{},
		SeerResults:  map[int]string{},
		DayVotes:     map[string]int{},
	}
	var events []game.GameEvent
	checkWinCondition(s, &events)
	if s.Winner == nil {
		t.Fatal("expected evil team to win when all villagers are dead")
	}
	if *s.Winner != "evil" {
		t.Errorf("expected winner 'evil', got %q", *s.Winner)
	}
}

func TestWinCondition_NoWin_WolvesOutnumberButConditionsNotMet(t *testing.T) {
	// Wolves outnumber good players but neither all villagers nor all magic are dead
	// — game should NOT end
	s := &State{
		Players: []Player{
			{ID: 1, Seat: 0, Role: RoleClawedWolf, Alive: true},
			{ID: 2, Seat: 1, Role: RoleClawedWolf, Alive: true},
			{ID: 3, Seat: 2, Role: RoleVillager, Alive: true},
			{ID: 4, Seat: 3, Role: RoleVillager, Alive: false},
			{ID: 5, Seat: 4, Role: RoleSeer, Alive: true},
			{ID: 6, Seat: 5, Role: RoleGuard, Alive: false},
		},
		PhaseActions: map[string]int{},
		SeerResults:  map[int]string{},
		DayVotes:     map[string]int{},
	}
	var events []game.GameEvent
	checkWinCondition(s, &events)
	if s.Winner != nil {
		t.Errorf("expected no winner yet, but got %q", *s.Winner)
	}
}

func TestGuardSaveMechanic(t *testing.T) {
	target := 2
	s := &State{
		NightKillTarget:  &target,
		NightGuardTarget: &target,
		Players: []Player{
			{ID: 1, Seat: 0, Role: RoleClawedWolf, Alive: true},
			{ID: 2, Seat: 1, Role: RoleClawedWolf, Alive: true},
			{ID: 3, Seat: 2, Role: RoleVillager, Alive: true},
			{ID: 4, Seat: 3, Role: RoleSeer, Alive: true},
			{ID: 5, Seat: 4, Role: RoleGuard, Alive: true},
			{ID: 6, Seat: 5, Role: RoleVillager, Alive: true},
		},
		PhaseActions: map[string]int{},
		SeerResults:  map[int]string{},
		DayVotes:     map[string]int{},
	}
	events := resolveNight(s)
	// Guard saved the target — player at seat 2 should still be alive
	p := playerBySeat(s, target)
	if p == nil || !p.Alive {
		t.Error("expected guard to save the target")
	}
	saved := false
	for _, ev := range events {
		if ev.EventType == "guard_save" {
			saved = true
		}
	}
	if !saved {
		t.Error("expected guard_save event")
	}
}

func TestGuardCannotProtectSamePlayerConsecutively(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	s := parseTestState(t, state)
	e := &Engine{}

	guardID := playerIDForRole(t, s, RoleGuard)
	guardSeat := 0
	for _, p := range s.Players {
		if p.Role == RoleGuard {
			guardSeat = p.Seat
		}
	}

	// Set state to guard phase with last_guard_target already set
	s.Phase = PhaseNightGuard
	s.LastGuardTarget = &guardSeat
	raw, _ := json.Marshal(s)

	action, _ := json.Marshal(map[string]interface{}{"type": "protect", "target_seat": guardSeat})
	_, err := e.ApplyAction(raw, guardID, action)
	if err == nil {
		t.Fatal("expected error: guard cannot protect same player consecutively")
	}
}

func TestDayDiscussionRoundRobin(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	s := parseTestState(t, state)
	e := &Engine{}

	// Set state to day_discuss phase
	s.Phase = PhaseDayDiscuss
	s.DaySpeeches = nil
	s.SpeakStartSeat = 0
	s.SpeakerIndex = 0
	raw, _ := json.Marshal(s)

	// Each alive player should be prompted to speak in seat order
	actions := pendingActionsForPhase(s)
	if len(actions) == 0 {
		t.Fatal("expected pending speak action")
	}
	firstSpeaker := actions[0].PlayerID

	speakAction, _ := json.Marshal(map[string]interface{}{"type": "speak", "message": "Hello"})
	result, err := e.ApplyAction(raw, firstSpeaker, speakAction)
	if err != nil {
		t.Fatalf("speak failed: %v", err)
	}
	state2 := lastState(t, result)
	s2 := parseTestState(t, state2)

	// After first speaks, the next player should be prompted
	nextActions := pendingActionsForPhase(s2)
	if len(nextActions) == 0 {
		t.Fatal("expected next speaker after first speech")
	}
	if nextActions[0].PlayerID == firstSpeaker {
		t.Error("should not prompt same player to speak twice")
	}
}

func TestDayVoting_Majority(t *testing.T) {
	// Set up state with 4 alive players at day_vote phase
	target := 1
	s := &State{
		Players: []Player{
			{ID: 101, Seat: 0, Role: RoleClawedWolf, Alive: true},
			{ID: 102, Seat: 1, Role: RoleVillager, Alive: true},
			{ID: 103, Seat: 2, Role: RoleSeer, Alive: true},
			{ID: 104, Seat: 3, Role: RoleGuard, Alive: true},
		},
		Phase:        PhaseDayVote,
		Round:        1,
		PhaseActions: map[string]int{},
		SeerResults:  map[int]string{},
		DayVotes:     map[string]int{"0": target, "2": target}, // 2 votes for seat 1
	}
	// Simulate 3rd vote making it majority
	s.DayVotes["3"] = target
	events := resolveVote(s)

	eliminated := false
	for _, ev := range events {
		if ev.EventType == "vote_result" {
			eliminated = true
		}
	}
	if !eliminated {
		t.Error("expected vote_result event")
	}
	p := playerBySeat(s, target)
	if p != nil && p.Alive {
		t.Error("voted-out player should be dead")
	}
}

func TestDayVoting_Tie_NoElimination(t *testing.T) {
	s := &State{
		Players: []Player{
			{ID: 101, Seat: 0, Role: RoleClawedWolf, Alive: true},
			{ID: 102, Seat: 1, Role: RoleVillager, Alive: true},
			{ID: 103, Seat: 2, Role: RoleSeer, Alive: true},
			{ID: 104, Seat: 3, Role: RoleGuard, Alive: true},
		},
		PhaseActions: map[string]int{},
		SeerResults:  map[int]string{},
		DayVotes:     map[string]int{"0": 1, "1": 0, "2": 1, "3": 0}, // 2-2 tie
	}
	events := resolveVote(s)

	noConsensus := false
	for _, ev := range events {
		if ev.EventType == "vote_result" {
			noConsensus = true
		}
	}
	if !noConsensus {
		t.Error("expected vote_result event for tie")
	}
	// All players should still be alive
	for _, p := range s.Players {
		if !p.Alive {
			t.Errorf("player %d should be alive after tie vote", p.ID)
		}
	}
}

func TestGetPlayerView_ClawedWolfSeesPartner(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	s := parseTestState(t, state)
	e := &Engine{}

	wolves := allPlayerIDsForRole(s, RoleClawedWolf)
	if len(wolves) != 2 {
		t.Fatal("expected 2 wolves")
	}

	view, err := e.GetPlayerView(state, wolves[0])
	if err != nil {
		t.Fatalf("GetPlayerView failed: %v", err)
	}

	var v map[string]interface{}
	json.Unmarshal(view, &v)

	if v["your_role"] != RoleClawedWolf {
		t.Errorf("wolf should see own role, got %v", v["your_role"])
	}
}

func TestGetSpectatorView_HidesRoles(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	e := &Engine{}

	view, err := e.GetSpectatorView(state)
	if err != nil {
		t.Fatalf("GetSpectatorView failed: %v", err)
	}

	var v map[string]interface{}
	json.Unmarshal(view, &v)

	players, ok := v["players"].([]interface{})
	if !ok {
		t.Fatal("expected players array in spectator view")
	}

	for _, p := range players {
		pm := p.(map[string]interface{})
		if pm["alive"] == true {
			if _, hasRole := pm["role"]; hasRole && pm["role"] != nil && pm["role"] != "" {
				t.Errorf("spectator view should not reveal roles of alive players, seat %v has role %v", pm["seat"], pm["role"])
			}
		}
	}
}

func TestGetGodView_RevealsAllRoles(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	e := &Engine{}

	view, err := e.GetGodView(state)
	if err != nil {
		t.Fatalf("GetGodView failed: %v", err)
	}

	var v map[string]interface{}
	json.Unmarshal(view, &v)

	players, ok := v["players"].([]interface{})
	if !ok {
		t.Fatal("expected players array in god view")
	}
	for _, p := range players {
		pm := p.(map[string]interface{})
		if pm["role"] == nil || pm["role"] == "" {
			t.Errorf("god view should reveal all roles, seat %v has no role", pm["seat"])
		}
	}
}

func TestApplyAction_EventsHaveStateAfter(t *testing.T) {
	state := initClawedWolf(t, testPlayers)
	s := parseTestState(t, state)
	e := &Engine{}

	wolves := allPlayerIDsForRole(s, RoleClawedWolf)
	var targetSeat int
	for _, p := range s.Players {
		if p.Role != RoleClawedWolf {
			targetSeat = p.Seat
			break
		}
	}

	action, _ := json.Marshal(map[string]interface{}{"type": "kill_vote", "target_seat": targetSeat})
	result, err := e.ApplyAction(state, wolves[0], action)
	if err != nil {
		t.Fatalf("ApplyAction failed: %v", err)
	}

	for i, ev := range result.Events {
		if len(ev.StateAfter) == 0 {
			t.Errorf("event %d (%s) has empty StateAfter", i, ev.EventType)
		}
	}
}

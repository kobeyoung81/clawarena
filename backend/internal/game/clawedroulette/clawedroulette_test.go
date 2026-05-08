package clawedroulette

import (
	"encoding/json"
	"testing"

	"github.com/clawarena/clawarena/internal/game"
)

func lastState(result *game.ApplyResult) json.RawMessage {
	if len(result.Events) == 0 {
		return nil
	}
	return result.Events[len(result.Events)-1].StateAfter
}

func isGameOver(result *game.ApplyResult) bool {
	for _, ev := range result.Events {
		if ev.GameOver {
			return true
		}
	}
	return false
}

// buildState creates a deterministic State for testing, bypassing random init.
func buildState(players []uint, bullets []string, gadgets [][]string) *State {
	pls := make([]Player, len(players))
	for i, pid := range players {
		var g []string
		if i < len(gadgets) {
			g = gadgets[i]
		}
		pls[i] = Player{ID: pid, Seat: i, Hits: 0, Alive: true, Gadgets: g}
	}
	return &State{
		Players:        pls,
		Bullets:        bullets,
		BulletIndex:    0,
		TotalBullets:   len(bullets),
		CurrentTurn:    0,
		TurnGadgetUsed: false,
		Phase:          "playing",
	}
}

func stateJSON(s *State) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

// ---------- InitState tests ----------

func TestInitState_2Players(t *testing.T) {
	e := &Engine{}
	state, events, err := e.InitState(nil, []uint{10, 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 || events[0].EventType != "game_start" {
		t.Fatal("expected game_start event")
	}
	var s State
	if err := json.Unmarshal(state, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(s.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(s.Players))
	}
	if s.TotalBullets != 12 {
		t.Errorf("expected 12 bullets, got %d", s.TotalBullets)
	}
	if s.Phase != "playing" {
		t.Errorf("expected phase playing, got %q", s.Phase)
	}
	for _, p := range s.Players {
		if len(p.Gadgets) != 2 {
			t.Errorf("player %d should have 2 gadgets, got %d", p.ID, len(p.Gadgets))
		}
	}
}

func TestInitState_3PlayersRejected(t *testing.T) {
	e := &Engine{}
	_, _, err := e.InitState(nil, []uint{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for 3 players")
	}
}

func TestInitState_4PlayersRejected(t *testing.T) {
	e := &Engine{}
	_, _, err := e.InitState(nil, []uint{1, 2, 3, 4})
	if err == nil {
		t.Fatal("expected error for 4 players")
	}
}

func TestInitState_TooFewPlayers(t *testing.T) {
	e := &Engine{}
	_, _, err := e.InitState(nil, []uint{1})
	if err == nil {
		t.Fatal("expected error for 1 player")
	}
}

func TestInitState_TooManyPlayers(t *testing.T) {
	e := &Engine{}
	_, _, err := e.InitState(nil, []uint{1, 2, 3, 4, 5})
	if err == nil {
		t.Fatal("expected error for 5 players")
	}
}

func TestInitState_BulletCounts(t *testing.T) {
	e := &Engine{}
	state, _, _ := e.InitState(nil, []uint{1, 2})
	var s State
	json.Unmarshal(state, &s)
	liveCount := 0
	for _, b := range s.Bullets {
		if b == "live" {
			liveCount++
		}
	}
	if liveCount < 5 {
		t.Errorf("expected at least 5 live rounds, got %d", liveCount)
	}
}

// ---------- Fire tests ----------

func TestFire_LiveRound_Hit(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank", "blank"}, [][]string{{}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(result), &ns)
	if ns.Players[1].Hits != 1 {
		t.Errorf("expected 1 hit on player 2, got %d", ns.Players[1].Hits)
	}
	if ns.CurrentTurn != 1 {
		t.Errorf("expected turn to advance to seat 1, got %d", ns.CurrentTurn)
	}

	// Verify fire event
	found := false
	for _, ev := range result.Events {
		if ev.EventType == "fire" {
			found = true
		}
	}
	if !found {
		t.Error("expected fire event")
	}
}

func TestFire_BlankAtSelf_ExtraTurn(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"blank", "live"}, [][]string{{}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 0})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(result), &ns)
	// Blank at self → extra turn, CurrentTurn should stay 0
	if ns.CurrentTurn != 0 {
		t.Errorf("expected extra turn (seat 0), got seat %d", ns.CurrentTurn)
	}
	if ns.Players[0].Hits != 0 {
		t.Errorf("blank should not cause a hit, got %d", ns.Players[0].Hits)
	}
}

func TestFire_BlankAtOther_TurnAdvances(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"blank", "live"}, [][]string{{}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(result), &ns)
	if ns.CurrentTurn != 1 {
		t.Errorf("expected turn to advance to seat 1, got %d", ns.CurrentTurn)
	}
}

func TestFire_LiveRound_SecondHitDoesNotEliminate(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank", "live"}, [][]string{{}, {}})
	s.Players[1].Hits = 1 // already has 1 hit
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(result), &ns)
	if ns.Players[1].Hits != 2 {
		t.Errorf("expected player 2 to have 2 hits, got %d", ns.Players[1].Hits)
	}
	if !ns.Players[1].Alive {
		t.Error("player 2 should still be alive at 2 hits")
	}

	foundElim := false
	for _, ev := range result.Events {
		if ev.EventType == "elimination" {
			foundElim = true
		}
	}
	if foundElim {
		t.Error("did not expect elimination event at 2 hits")
	}
	if isGameOver(result) {
		t.Error("did not expect game over at 2 hits")
	}
}

func TestElimination_At3Hits(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank", "live"}, [][]string{{}, {}})
	s.Players[1].Hits = 2 // already has 2 hits
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(result), &ns)
	if ns.Players[1].Alive {
		t.Error("player 2 should be eliminated at 3 hits")
	}

	foundElim := false
	for _, ev := range result.Events {
		if ev.EventType == "elimination" {
			foundElim = true
		}
	}
	if !foundElim {
		t.Error("expected elimination event")
	}
}

func TestGameOver_OnePlayerRemains(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{}, {}})
	s.Players[1].Hits = 2
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !isGameOver(result) {
		t.Fatal("expected game over")
	}
	var ns State
	json.Unmarshal(lastState(result), &ns)
	if ns.Winner == nil || *ns.Winner != 1 {
		t.Error("expected player 1 to win")
	}
	if ns.Phase != "finished" {
		t.Errorf("expected phase finished, got %q", ns.Phase)
	}
}

func TestGameOver_AllBulletsUsed(t *testing.T) {
	e := &Engine{}
	// 2 bullets: both blank → resolve by hits
	s := buildState([]uint{1, 2}, []string{"blank", "blank"}, [][]string{{}, {}})
	s.Players[0].Hits = 1
	raw := stateJSON(s)

	// Player 1 fires blank at player 2 → turn advances
	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw = lastState(result)

	// Player 2 fires last blank at player 1
	action, _ = json.Marshal(map[string]any{"type": "fire", "target": 0})
	result, err = e.ApplyAction(raw, 2, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !isGameOver(result) {
		t.Fatal("expected game over when all bullets used")
	}
	var ns State
	json.Unmarshal(lastState(result), &ns)
	// Player 2 has 0 hits, player 1 has 1 hit → player 2 wins
	if ns.Winner == nil || *ns.Winner != 2 {
		t.Errorf("expected player 2 to win (fewer hits), winner=%v", ns.Winner)
	}
}

// ---------- Gadget tests ----------

func TestGadget_FishChips_ReduceHit(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "live"}, [][]string{{"fish_chips"}, {}})
	s.Players[0].Hits = 1
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "fish_chips"})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(result), &ns)
	if ns.Players[0].Hits != 0 {
		t.Errorf("expected 0 hits after fish_chips, got %d", ns.Players[0].Hits)
	}
	if len(ns.Players[0].Gadgets) != 0 {
		t.Errorf("expected gadget removed, got %v", ns.Players[0].Gadgets)
	}
	if ns.CurrentTurn != 0 {
		t.Errorf("expected turn to stay on the same player after gadget use, got %d", ns.CurrentTurn)
	}
	if !ns.TurnGadgetUsed {
		t.Error("expected turn_gadget_used to be true after gadget use")
	}
}

func TestGadget_FishChips_MinZero(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{"fish_chips"}, {}})
	// Player has 0 hits, using fish_chips should keep at 0
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "fish_chips"})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(result), &ns)
	if ns.Players[0].Hits != 0 {
		t.Errorf("hits should stay at 0, got %d", ns.Players[0].Hits)
	}
}

func TestGadget_Goggles_PeekBullet(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{"goggles"}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "goggles"})
	result, err := e.ApplyAction(raw, 1, action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(result), &ns)
	if ns.LastPeek == nil || *ns.LastPeek != "live" {
		t.Errorf("expected peek to show 'live', got %v", ns.LastPeek)
	}
	if ns.PeekPlayerID == nil || *ns.PeekPlayerID != 1 {
		t.Errorf("expected peek player to be 1, got %v", ns.PeekPlayerID)
	}
	if len(ns.Players[0].Gadgets) != 0 {
		t.Errorf("expected gadget removed, got %v", ns.Players[0].Gadgets)
	}
}

func TestGadget_Goggles_PlayerView(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{"goggles"}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "goggles"})
	result, _ := e.ApplyAction(raw, 1, action)
	afterState := lastState(result)

	// Player 1 should see the peek
	view1, _ := e.GetPlayerView(afterState, 1)
	var pv1 playerView
	json.Unmarshal(view1, &pv1)
	if pv1.LastPeek == nil || *pv1.LastPeek != "live" {
		t.Error("player 1 should see peek result")
	}
	if !pv1.TurnGadgetUsed {
		t.Error("player 1 should see that the gadget has already been used this turn")
	}

	// Player 2 should NOT see the peek
	view2, _ := e.GetPlayerView(afterState, 2)
	var pv2 playerView
	json.Unmarshal(view2, &pv2)
	if pv2.LastPeek != nil {
		t.Error("player 2 should not see peek result")
	}
	if !pv2.TurnGadgetUsed {
		t.Error("spectating players should see that the turn is waiting for a shot")
	}
}

func TestPendingActions_DefaultAllowsGadgetOrFire(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{"goggles"}, {}})

	actions, err := e.GetPendingActions(stateJSON(s))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 pending action, got %d", len(actions))
	}
	if actions[0].Prompt != "Choose an action: fire at yourself, fire at another player, or use one gadget before your mandatory shot." {
		t.Errorf("unexpected prompt: %q", actions[0].Prompt)
	}
}

func TestPendingActions_AfterGadgetRequiresShot(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{"goggles"}, {}})
	s.TurnGadgetUsed = true

	actions, err := e.GetPendingActions(stateJSON(s))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 pending action, got %d", len(actions))
	}
	if actions[0].Prompt != "Choose a target to fire. You already used one gadget this turn." {
		t.Errorf("unexpected prompt: %q", actions[0].Prompt)
	}
}

func TestTurnModel_GadgetThenFireAdvancesTurn(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{"fish_chips"}, {}})
	s.Players[0].Hits = 1

	gadgetAction, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "fish_chips"})
	gadgetResult, err := e.ApplyAction(stateJSON(s), 1, gadgetAction)
	if err != nil {
		t.Fatalf("unexpected gadget error: %v", err)
	}

	fireAction, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	fireResult, err := e.ApplyAction(lastState(gadgetResult), 1, fireAction)
	if err != nil {
		t.Fatalf("unexpected fire error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(fireResult), &ns)
	if ns.CurrentTurn != 1 {
		t.Errorf("expected turn to advance after the mandatory shot, got %d", ns.CurrentTurn)
	}
	if ns.TurnGadgetUsed {
		t.Error("expected turn_gadget_used to reset after the shot")
	}
}

func TestTurnModel_GadgetThenBlankSelfResetsTurnState(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"blank", "live"}, [][]string{{"goggles"}, {}})

	gadgetAction, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "goggles"})
	gadgetResult, err := e.ApplyAction(stateJSON(s), 1, gadgetAction)
	if err != nil {
		t.Fatalf("unexpected gadget error: %v", err)
	}

	fireAction, _ := json.Marshal(map[string]any{"type": "fire", "target": 0})
	fireResult, err := e.ApplyAction(lastState(gadgetResult), 1, fireAction)
	if err != nil {
		t.Fatalf("unexpected fire error: %v", err)
	}

	var ns State
	json.Unmarshal(lastState(fireResult), &ns)
	if ns.CurrentTurn != 0 {
		t.Errorf("expected blank self-shot to keep the current player, got %d", ns.CurrentTurn)
	}
	if ns.TurnGadgetUsed {
		t.Error("expected turn_gadget_used to reset for the extra turn")
	}
}

func TestTurnModel_CannotUseSecondGadgetSameTurn(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{"goggles", "fish_chips"}, {}})

	gadgetAction, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "goggles"})
	gadgetResult, err := e.ApplyAction(stateJSON(s), 1, gadgetAction)
	if err != nil {
		t.Fatalf("unexpected gadget error: %v", err)
	}

	secondGadgetAction, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "fish_chips"})
	_, err = e.ApplyAction(lastState(gadgetResult), 1, secondGadgetAction)
	if err == nil {
		t.Fatal("expected error for using a second gadget in the same turn")
	}
}

// ---------- Validation tests ----------

func TestInvalid_WrongTurn(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 0})
	_, err := e.ApplyAction(raw, 2, action) // player 2 but it's player 1's turn
	if err == nil {
		t.Fatal("expected error for wrong turn")
	}
}

func TestInvalid_DeadTarget(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2, 3}, []string{"live"}, [][]string{{}, {}, {}})
	s.Players[1].Alive = false
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	_, err := e.ApplyAction(raw, 1, action)
	if err == nil {
		t.Fatal("expected error for firing at eliminated player")
	}
}

func TestInvalid_NoGadget(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "gadget", "gadget": "fish_chips"})
	_, err := e.ApplyAction(raw, 1, action)
	if err == nil {
		t.Fatal("expected error for using gadget player doesn't have")
	}
}

func TestInvalid_GameOver(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{}, {}})
	s.Phase = "finished"
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 1})
	_, err := e.ApplyAction(raw, 1, action)
	if err == nil {
		t.Fatal("expected error for action on finished game")
	}
}

func TestInvalid_BadActionType(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "dance"})
	_, err := e.ApplyAction(raw, 1, action)
	if err == nil {
		t.Fatal("expected error for unknown action type")
	}
}

func TestInvalid_InvalidTargetSeat(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{}, {}})
	raw := stateJSON(s)

	action, _ := json.Marshal(map[string]any{"type": "fire", "target": 5})
	_, err := e.ApplyAction(raw, 1, action)
	if err == nil {
		t.Fatal("expected error for invalid target seat")
	}
}

// ---------- View tests ----------

func TestGetPlayerView_HidesOtherGadgets(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{"fish_chips"}, {"goggles"}})
	raw := stateJSON(s)

	view, _ := e.GetPlayerView(raw, 1)
	var pv playerView
	json.Unmarshal(view, &pv)

	// Player 1 sees own gadgets
	if len(pv.Players[0].Gadgets) != 1 || pv.Players[0].Gadgets[0] != "fish_chips" {
		t.Errorf("player 1 should see own gadgets, got %v", pv.Players[0].Gadgets)
	}
	// Player 2's gadgets are hidden (only count shown)
	if len(pv.Players[1].Gadgets) != 0 {
		t.Errorf("player 2's gadgets should be hidden, got %v", pv.Players[1].Gadgets)
	}
	if pv.Players[1].GadgetCount != 1 {
		t.Errorf("player 2 gadget count should be 1, got %d", pv.Players[1].GadgetCount)
	}
}

func TestGetPlayerView_HidesBulletOrder(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{}, {}})
	raw := stateJSON(s)

	view, _ := e.GetPlayerView(raw, 1)
	// Ensure bullet order is not present
	var m map[string]any
	json.Unmarshal(view, &m)
	if _, ok := m["bullets"]; ok {
		t.Error("player view should not contain bullet order")
	}
}

func TestGetSpectatorView(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{"fish_chips"}, {}})
	raw := stateJSON(s)

	view, err := e.GetSpectatorView(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sv spectatorView
	json.Unmarshal(view, &sv)
	if len(sv.Players) != 2 {
		t.Errorf("expected 2 players in spectator view, got %d", len(sv.Players))
	}
}

func TestGetGodView(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live", "blank"}, [][]string{{"fish_chips"}, {"goggles"}})
	raw := stateJSON(s)

	view, _ := e.GetGodView(raw)
	var gs State
	json.Unmarshal(view, &gs)
	if len(gs.Bullets) != 2 {
		t.Errorf("god view should show all bullets, got %d", len(gs.Bullets))
	}
	if len(gs.Players[0].Gadgets) != 1 {
		t.Errorf("god view should show all gadgets, got %v", gs.Players[0].Gadgets)
	}
}

func TestGetPendingActions(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{"live"}, [][]string{{}, {}})
	raw := stateJSON(s)

	actions, err := e.GetPendingActions(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 pending action, got %d", len(actions))
	}
	if actions[0].PlayerID != 1 {
		t.Errorf("expected pending action for player 1, got %d", actions[0].PlayerID)
	}
	if actions[0].ActionType != "turn" {
		t.Errorf("expected action type 'turn', got %q", actions[0].ActionType)
	}
}

func TestGetPendingActions_GameOver(t *testing.T) {
	e := &Engine{}
	s := buildState([]uint{1, 2}, []string{}, [][]string{{}, {}})
	s.Phase = "finished"
	raw := stateJSON(s)

	actions, _ := e.GetPendingActions(raw)
	if len(actions) != 0 {
		t.Errorf("expected no pending actions after game over, got %d", len(actions))
	}
}

// ---------- Syncronym & event model ----------

func TestSyncronym(t *testing.T) {
	e := &Engine{}
	if e.Syncronym() != "cr" {
		t.Errorf("expected syncronym 'cr', got %q", e.Syncronym())
	}
}

func TestNewEventModel(t *testing.T) {
	e := &Engine{}
	m := e.NewEventModel()
	if m.TableName() != "cr_game_events" {
		t.Errorf("expected table name 'cr_game_events', got %q", m.TableName())
	}
}

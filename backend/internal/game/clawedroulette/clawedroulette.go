package clawedroulette

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"

	"github.com/clawarena/clawarena/internal/game"
)

func init() {
	game.Register("clawed_roulette", &Engine{})
}

// Player represents a participant in the roulette game.
type Player struct {
	ID      uint     `json:"id"`
	Seat    int      `json:"seat"`
	Hits    int      `json:"hits"`
	Alive   bool     `json:"alive"`
	Gadgets []string `json:"gadgets"` // "fish_chips" or "goggles"
}

// State is the internal Clawed Roulette game state.
type State struct {
	Players      []Player `json:"players"`
	Bullets      []string `json:"bullets"`                  // "live" or "blank" — remaining bullets
	BulletIndex  int      `json:"bullet_index"`             // next bullet to fire
	TotalBullets int      `json:"total_bullets"`
	CurrentTurn  int      `json:"current_turn"`             // seat index of current player
	Phase        string   `json:"phase"`                    // "playing" or "finished"
	Winner       *uint    `json:"winner,omitempty"`
	IsDraw       bool     `json:"is_draw"`
	LastPeek     *string  `json:"last_peek,omitempty"`      // result of goggles (private)
	PeekPlayerID *uint    `json:"peek_player_id,omitempty"` // who peeked
}

// Engine implements game.GameEngine for Clawed Roulette.
type Engine struct{}

func (e *Engine) Syncronym() string                                  { return "cr" }
func (e *Engine) NewEventModel() game.GameEventRecord                { return &CrGameEvent{} }
func (e *Engine) GetPhaseTimeout(_ json.RawMessage) *game.PhaseTimeout { return nil }

func parseState(raw json.RawMessage) (*State, error) {
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func marshalState(s *State) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

// InitState sets up a new game with exactly 2 players.
func (e *Engine) InitState(config json.RawMessage, players []uint) (json.RawMessage, []game.GameEvent, error) {
	if len(players) != 2 {
		return nil, nil, errors.New("clawed_roulette requires exactly 2 players")
	}

	// Build 12 bullets: random mix with blanks < 8 (i.e. at least 5 live).
	totalBullets := 12
	numBlanks := rand.Intn(8) // 0..7 blanks
	numLive := totalBullets - numBlanks
	bullets := make([]string, totalBullets)
	for i := 0; i < numLive; i++ {
		bullets[i] = "live"
	}
	for i := numLive; i < totalBullets; i++ {
		bullets[i] = "blank"
	}
	rand.Shuffle(len(bullets), func(i, j int) { bullets[i], bullets[j] = bullets[j], bullets[i] })

	// Build gadget pool: 1 fish_chips + 1 goggles per player, shuffle, deal 2 each.
	n := len(players)
	pool := make([]string, 0, n*2)
	for i := 0; i < n; i++ {
		pool = append(pool, "fish_chips")
		pool = append(pool, "goggles")
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	pls := make([]Player, n)
	for i, pid := range players {
		pls[i] = Player{
			ID:      pid,
			Seat:    i,
			Hits:    0,
			Alive:   true,
			Gadgets: []string{pool[i*2], pool[i*2+1]},
		}
	}

	startSeat := rand.Intn(n)

	s := &State{
		Players:      pls,
		Bullets:      bullets,
		BulletIndex:  0,
		TotalBullets: totalBullets,
		CurrentTurn:  startSeat,
		Phase:        "playing",
	}
	stateJSON := marshalState(s)

	// Event details: player info + bullet composition (counts only).
	type playerInfo struct {
		ID   uint `json:"id"`
		Seat int  `json:"seat"`
	}
	pInfos := make([]playerInfo, n)
	for i, p := range pls {
		pInfos[i] = playerInfo{ID: p.ID, Seat: p.Seat}
	}
	details, _ := json.Marshal(map[string]any{
		"players":     pInfos,
		"live_count":  numLive,
		"blank_count": numBlanks,
		"first_seat":  startSeat,
	})

	events := []game.GameEvent{{
		Source:     "system",
		EventType:  "game_start",
		Details:    details,
		StateAfter: stateJSON,
		Visibility: "public",
	}}

	return stateJSON, events, nil
}

// ---------- views ----------

type playerViewPlayer struct {
	ID           uint     `json:"id"`
	Seat         int      `json:"seat"`
	Hits         int      `json:"hits"`
	Alive        bool     `json:"alive"`
	Gadgets      []string `json:"gadgets,omitempty"`
	GadgetCount  int      `json:"gadget_count"`
}

type playerView struct {
	Players       []playerViewPlayer  `json:"players"`
	BulletIndex   int                 `json:"bullet_index"`
	TotalBullets  int                 `json:"total_bullets"`
	CurrentTurn   int                 `json:"current_turn"`
	Phase         string              `json:"phase"`
	Winner        *uint               `json:"winner,omitempty"`
	IsDraw        bool                `json:"is_draw"`
	LastPeek      *string             `json:"last_peek,omitempty"`
}

func (e *Engine) GetPlayerView(raw json.RawMessage, playerID uint) (json.RawMessage, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	pv := playerView{
		BulletIndex:  s.BulletIndex,
		TotalBullets: s.TotalBullets,
		CurrentTurn:  s.CurrentTurn,
		Phase:        s.Phase,
		Winner:       s.Winner,
		IsDraw:       s.IsDraw,
	}
	for _, p := range s.Players {
		pvp := playerViewPlayer{
			ID:          p.ID,
			Seat:        p.Seat,
			Hits:        p.Hits,
			Alive:       p.Alive,
			GadgetCount: len(p.Gadgets),
		}
		if p.ID == playerID {
			pvp.Gadgets = p.Gadgets
		}
		pv.Players = append(pv.Players, pvp)
	}
	if s.PeekPlayerID != nil && *s.PeekPlayerID == playerID {
		pv.LastPeek = s.LastPeek
	}
	return json.Marshal(pv)
}

type spectatorViewPlayer struct {
	ID    uint `json:"id"`
	Seat  int  `json:"seat"`
	Hits  int  `json:"hits"`
	Alive bool `json:"alive"`
}

type spectatorView struct {
	Players      []spectatorViewPlayer `json:"players"`
	BulletIndex  int                   `json:"bullet_index"`
	TotalBullets int                   `json:"total_bullets"`
	CurrentTurn  int                   `json:"current_turn"`
	Phase        string                `json:"phase"`
	Winner       *uint                 `json:"winner,omitempty"`
	IsDraw       bool                  `json:"is_draw"`
}

func (e *Engine) GetSpectatorView(raw json.RawMessage) (json.RawMessage, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	sv := spectatorView{
		BulletIndex:  s.BulletIndex,
		TotalBullets: s.TotalBullets,
		CurrentTurn:  s.CurrentTurn,
		Phase:        s.Phase,
		Winner:       s.Winner,
		IsDraw:       s.IsDraw,
	}
	for _, p := range s.Players {
		sv.Players = append(sv.Players, spectatorViewPlayer{
			ID: p.ID, Seat: p.Seat, Hits: p.Hits, Alive: p.Alive,
		})
	}
	return json.Marshal(sv)
}

func (e *Engine) GetGodView(raw json.RawMessage) (json.RawMessage, error) {
	return raw, nil
}

// ---------- pending actions ----------

func (e *Engine) GetPendingActions(raw json.RawMessage) ([]game.PendingAction, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	if s.Phase == "finished" {
		return nil, nil
	}
	cp := s.Players[s.CurrentTurn]
	var targets []int
	for _, p := range s.Players {
		if p.Alive {
			targets = append(targets, p.Seat)
		}
	}
	return []game.PendingAction{{
		PlayerID:     cp.ID,
		ActionType:   "turn",
		Prompt:       "Choose an action: fire at yourself, fire at another player, or use a gadget.",
		ValidTargets: targets,
	}}, nil
}

// ---------- apply action ----------

type actionPayload struct {
	Type   string `json:"type"`   // "fire" or "gadget"
	Target *int   `json:"target"` // seat index for fire
	Gadget string `json:"gadget"` // gadget name
}

func (e *Engine) ApplyAction(raw json.RawMessage, playerID uint, actionRaw json.RawMessage) (*game.ApplyResult, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	if s.Phase == "finished" {
		return nil, errors.New("game is already over")
	}

	cp := &s.Players[s.CurrentTurn]
	if cp.ID != playerID {
		return nil, errors.New("not your turn")
	}

	var act actionPayload
	if err := json.Unmarshal(actionRaw, &act); err != nil {
		return nil, fmt.Errorf("invalid action: %w", err)
	}

	// Clear previous peek
	s.LastPeek = nil
	s.PeekPlayerID = nil

	var events []game.GameEvent

	switch act.Type {
	case "fire":
		evts, err := e.handleFire(s, cp, act)
		if err != nil {
			return nil, err
		}
		events = append(events, evts...)
	case "gadget":
		evts, err := e.handleGadget(s, cp, act)
		if err != nil {
			return nil, err
		}
		events = append(events, evts...)
	default:
		return nil, fmt.Errorf("unknown action type: %q", act.Type)
	}

	return &game.ApplyResult{Events: events}, nil
}

func (e *Engine) handleFire(s *State, shooter *Player, act actionPayload) ([]game.GameEvent, error) {
	if act.Target == nil {
		return nil, errors.New("fire action requires a target seat")
	}
	targetSeat := *act.Target
	if targetSeat < 0 || targetSeat >= len(s.Players) {
		return nil, fmt.Errorf("invalid target seat: %d", targetSeat)
	}
	target := &s.Players[targetSeat]
	if !target.Alive {
		return nil, errors.New("target player is eliminated")
	}
	if s.BulletIndex >= len(s.Bullets) {
		return nil, errors.New("no bullets remaining")
	}

	bullet := s.Bullets[s.BulletIndex]
	s.BulletIndex++

	var events []game.GameEvent
	selfShot := shooter.Seat == targetSeat

	if bullet == "live" {
		target.Hits++
		eliminated := target.Hits >= 2

		if eliminated {
			target.Alive = false
		}

		stateJSON := marshalState(s)
		fireDetails, _ := json.Marshal(map[string]any{
			"bullet":     "live",
			"target":     targetSeat,
			"self_shot":  selfShot,
			"target_hits": target.Hits,
		})
		events = append(events, game.GameEvent{
			Source:     "agent",
			EventType:  "fire",
			Actor:     &game.EventEntity{AgentID: &shooter.ID, Seat: intPtr(shooter.Seat)},
			Target:    &game.EventEntity{AgentID: &target.ID, Seat: intPtr(target.Seat)},
			Details:    fireDetails,
			StateAfter: stateJSON,
			Visibility: "public",
		})

		if eliminated {
			elimDetails, _ := json.Marshal(map[string]any{
				"eliminated_id":   target.ID,
				"eliminated_seat": target.Seat,
			})
			events = append(events, game.GameEvent{
				Source:     "system",
				EventType:  "elimination",
				Target:    &game.EventEntity{AgentID: &target.ID, Seat: intPtr(target.Seat)},
				Details:    elimDetails,
				StateAfter: marshalState(s),
				Visibility: "public",
			})
		}

		// Check game end
		if ended, result := checkGameEnd(s); ended {
			s.Phase = "finished"
			endState := marshalState(s)
			// Update last event's state
			events[len(events)-1].StateAfter = endState

			endDetails, _ := json.Marshal(map[string]any{
				"winner":  s.Winner,
				"is_draw": s.IsDraw,
			})
			events = append(events, game.GameEvent{
				Source:     "system",
				EventType:  "game_over",
				Details:    endDetails,
				StateAfter: endState,
				Visibility: "public",
				GameOver:   true,
				Result:     result,
			})
			return events, nil
		}

		advanceTurn(s)
		// Update the last event's state to reflect the turn change
		events[len(events)-1].StateAfter = marshalState(s)

	} else {
		// blank
		stateJSON := marshalState(s)
		fireDetails, _ := json.Marshal(map[string]any{
			"bullet":    "blank",
			"target":    targetSeat,
			"self_shot": selfShot,
		})

		if selfShot {
			// Extra turn: don't advance
		} else {
			advanceTurn(s)
		}

		// Check if all bullets used
		if ended, result := checkGameEnd(s); ended {
			s.Phase = "finished"
			endState := marshalState(s)
			events = append(events, game.GameEvent{
				Source:     "agent",
				EventType:  "fire",
				Actor:     &game.EventEntity{AgentID: &shooter.ID, Seat: intPtr(shooter.Seat)},
				Target:    &game.EventEntity{AgentID: &s.Players[targetSeat].ID, Seat: intPtr(targetSeat)},
				Details:    fireDetails,
				StateAfter: endState,
				Visibility: "public",
			})
			endDetails, _ := json.Marshal(map[string]any{
				"winner":  s.Winner,
				"is_draw": s.IsDraw,
			})
			events = append(events, game.GameEvent{
				Source:     "system",
				EventType:  "game_over",
				Details:    endDetails,
				StateAfter: endState,
				Visibility: "public",
				GameOver:   true,
				Result:     result,
			})
			return events, nil
		}

		finalState := marshalState(s)
		_ = stateJSON // computed before turn advance
		events = append(events, game.GameEvent{
			Source:     "agent",
			EventType:  "fire",
			Actor:     &game.EventEntity{AgentID: &shooter.ID, Seat: intPtr(shooter.Seat)},
			Target:    &game.EventEntity{AgentID: &s.Players[targetSeat].ID, Seat: intPtr(targetSeat)},
			Details:    fireDetails,
			StateAfter: finalState,
			Visibility: "public",
		})
	}

	return events, nil
}

func (e *Engine) handleGadget(s *State, player *Player, act actionPayload) ([]game.GameEvent, error) {
	if act.Gadget != "fish_chips" && act.Gadget != "goggles" {
		return nil, fmt.Errorf("unknown gadget: %q", act.Gadget)
	}

	// Verify player has the gadget
	idx := -1
	for i, g := range player.Gadgets {
		if g == act.Gadget {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("you don't have gadget %q", act.Gadget)
	}

	// Remove gadget from hand
	player.Gadgets = append(player.Gadgets[:idx], player.Gadgets[idx+1:]...)

	var events []game.GameEvent

	switch act.Gadget {
	case "fish_chips":
		if player.Hits > 0 {
			player.Hits--
		}
		advanceTurn(s)
		details, _ := json.Marshal(map[string]any{
			"gadget":    "fish_chips",
			"player_id": player.ID,
			"hits_after": player.Hits,
		})
		events = append(events, game.GameEvent{
			Source:     "agent",
			EventType:  "gadget_use",
			Actor:     &game.EventEntity{AgentID: &player.ID, Seat: intPtr(player.Seat)},
			Details:    details,
			StateAfter: marshalState(s),
			Visibility: "public",
		})

	case "goggles":
		if s.BulletIndex < len(s.Bullets) {
			peek := s.Bullets[s.BulletIndex]
			s.LastPeek = &peek
			s.PeekPlayerID = uintPtr(player.ID)
		}
		advanceTurn(s)
		details, _ := json.Marshal(map[string]any{
			"gadget":    "goggles",
			"player_id": player.ID,
		})
		events = append(events, game.GameEvent{
			Source:     "agent",
			EventType:  "gadget_use",
			Actor:     &game.EventEntity{AgentID: &player.ID, Seat: intPtr(player.Seat)},
			Details:    details,
			StateAfter: marshalState(s),
			Visibility: "public",
		})
	}

	return events, nil
}

// ---------- helpers ----------

func advanceTurn(s *State) {
	n := len(s.Players)
	for i := 1; i <= n; i++ {
		next := (s.CurrentTurn + i) % n
		if s.Players[next].Alive {
			s.CurrentTurn = next
			return
		}
	}
}

func checkGameEnd(s *State) (bool, *game.GameResult) {
	// All bullets used → resolve by fewest hits among alive players
	if s.BulletIndex >= len(s.Bullets) {
		return resolveByHits(s)
	}
	// Count alive players
	var alive []Player
	for _, p := range s.Players {
		if p.Alive {
			alive = append(alive, p)
		}
	}
	if len(alive) == 1 {
		s.Winner = uintPtr(alive[0].ID)
		return true, &game.GameResult{WinnerIDs: []uint{alive[0].ID}}
	}
	if len(alive) == 0 {
		s.IsDraw = true
		return true, &game.GameResult{WinnerIDs: []uint{}}
	}
	return false, nil
}

func resolveByHits(s *State) (bool, *game.GameResult) {
	var alive []Player
	for _, p := range s.Players {
		if p.Alive {
			alive = append(alive, p)
		}
	}
	if len(alive) == 0 {
		s.IsDraw = true
		return true, &game.GameResult{WinnerIDs: []uint{}}
	}
	if len(alive) == 1 {
		s.Winner = uintPtr(alive[0].ID)
		return true, &game.GameResult{WinnerIDs: []uint{alive[0].ID}}
	}

	// Find fewest hits
	minHits := alive[0].Hits
	for _, p := range alive[1:] {
		if p.Hits < minHits {
			minHits = p.Hits
		}
	}
	var winners []uint
	for _, p := range alive {
		if p.Hits == minHits {
			winners = append(winners, p.ID)
		}
	}
	if len(winners) == 1 {
		s.Winner = uintPtr(winners[0])
	} else {
		s.IsDraw = true
	}
	return true, &game.GameResult{WinnerIDs: winners}
}

func intPtr(v int) *int    { return &v }
func uintPtr(v uint) *uint { return &v }

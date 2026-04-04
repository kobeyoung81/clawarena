package clawedwolf

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"

	"github.com/clawarena/clawarena/internal/game"
)

func init() {
	game.Register("clawedwolf", &Engine{})
}

const (
	PhaseNightClawedWolf  = "night_clawedwolf"
	PhaseNightWolfDiscuss = "night_wolf_discuss"
	PhaseNightSeer        = "night_seer"
	PhaseNightGuard       = "night_guard"
	PhaseDayAnnounce      = "day_announce"
	PhaseDayDiscuss       = "day_discuss"
	PhaseDayVote          = "day_vote"
	PhaseDayResult        = "day_result"
	PhaseFinished         = "finished"

	maxWolfVoteRounds = 3

	RoleClawedWolf = "clawedwolf"
	RoleSeer       = "seer"
	RoleGuard      = "guard"
	RoleVillager   = "villager"
)

type Player struct {
	ID        uint   `json:"id"`
	Seat      int    `json:"seat"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Alive     bool   `json:"alive"`
	LastWords string `json:"last_words,omitempty"`
}

type State struct {
	Players          []Player       `json:"players"`
	Phase            string         `json:"phase"`
	Round            int            `json:"round"`
	PhaseActions     map[string]int `json:"phase_actions"`
	NightKillTarget  *int           `json:"night_kill_target"`
	NightGuardTarget *int           `json:"night_guard_target"`
	LastGuardTarget  *int           `json:"last_guard_target"`
	SeerResults      map[int]string `json:"seer_results"`
	DaySpeeches      []Speech       `json:"day_speeches"`
	DayVotes         map[string]int `json:"day_votes"`
	Eliminated       []int          `json:"eliminated"`
	SpeakerIndex     int            `json:"speaker_index"`
	SpeakStartSeat   int            `json:"speak_start_seat"`
	WolfSpeeches     []Speech       `json:"wolf_speeches,omitempty"`
	WolfVoteRound    int            `json:"wolf_vote_round"`
	Winner           *string        `json:"winner,omitempty"`
}

type Speech struct {
	Seat    int    `json:"seat"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type Engine struct{}

// ---------------------------------------------------------------------------
// Helper utilities
// ---------------------------------------------------------------------------

func intPtr(v int) *int       { return &v }
func uintPtr(v uint) *uint    { return &v }

// wolfVoteRound returns the current wolf vote round (1-based), treating 0 as 1
// for backward compatibility with states serialized before this field existed.
func (s *State) wolfVoteRound() int {
	if s.WolfVoteRound < 1 {
		return 1
	}
	return s.WolfVoteRound
}

func stateSnapshot(s *State) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func parseState(raw json.RawMessage) (*State, error) {
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	if s.PhaseActions == nil {
		s.PhaseActions = map[string]int{}
	}
	if s.SeerResults == nil {
		s.SeerResults = map[int]string{}
	}
	if s.DayVotes == nil {
		s.DayVotes = map[string]int{}
	}
	return &s, nil
}

// ---------------------------------------------------------------------------
// New interface methods
// ---------------------------------------------------------------------------

func (e *Engine) Syncronym() string { return "cw" }

func (e *Engine) NewEventModel() game.GameEventRecord { return &CwGameEvent{} }

func (e *Engine) GetPhaseTimeout(_ json.RawMessage) *game.PhaseTimeout { return nil }

// ---------------------------------------------------------------------------
// InitState
// ---------------------------------------------------------------------------

func (e *Engine) InitState(config json.RawMessage, players []uint) (json.RawMessage, []game.GameEvent, error) {
	if len(players) != 6 {
		return nil, nil, errors.New("clawedwolf requires exactly 6 players")
	}

	// Extract player names from config
	var cfg struct {
		PlayerNames map[string]string `json:"player_names"`
	}
	json.Unmarshal(config, &cfg)

	roles := []string{RoleClawedWolf, RoleClawedWolf, RoleSeer, RoleGuard, RoleVillager, RoleVillager}
	perm := rand.Perm(6)
	ps := make([]Player, 6)
	for i, pid := range players {
		name := ""
		if cfg.PlayerNames != nil {
			name = cfg.PlayerNames[fmt.Sprintf("%d", pid)]
		}
		ps[i] = Player{ID: pid, Seat: i, Name: name, Role: roles[perm[i]], Alive: true}
	}

	s := State{
		Players:        ps,
		Phase:          PhaseNightClawedWolf,
		Round:          1,
		PhaseActions:   map[string]int{},
		SeerResults:    map[int]string{},
		DayVotes:       map[string]int{},
		SpeakStartSeat: 0,
		WolfVoteRound:  1,
	}

	stateJSON := stateSnapshot(&s)

	// Seed events: game_start (public) + roles_assigned per player (private)
	events := []game.GameEvent{
		{
			Source:     "system",
			EventType:  "game_start",
			Details:    mustJSON(map[string]any{"player_count": len(players)}),
			StateAfter: stateJSON,
			Visibility: "public",
		},
	}

	// Public phase_change for the first night
	events = append(events, game.GameEvent{
		Source:     "system",
		EventType:  "phase_change",
		Details:    mustJSON(map[string]any{"phase": s.Phase, "round": s.Round}),
		StateAfter: stateJSON,
		Visibility: "public",
	})

	for _, p := range s.Players {
		events = append(events, game.GameEvent{
			Source:     "system",
			EventType:  "roles_assigned",
			Target:     &game.EventEntity{AgentID: uintPtr(p.ID), Seat: intPtr(p.Seat)},
			Details:    mustJSON(map[string]any{"role": p.Role}),
			StateAfter: stateJSON,
			Visibility: fmt.Sprintf("player:%d", p.ID),
		})
	}

	return stateJSON, events, nil
}

// ---------------------------------------------------------------------------
// View functions
// ---------------------------------------------------------------------------

func (e *Engine) GetPlayerView(raw json.RawMessage, playerID uint) (json.RawMessage, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}

	type publicPlayer struct {
		ID    uint   `json:"id"`
		Seat  int    `json:"seat"`
		Name  string `json:"name,omitempty"`
		Alive bool   `json:"alive"`
		Role  string `json:"role,omitempty"`
	}

	// Find the requesting player
	var myPlayer *Player
	for i := range s.Players {
		if s.Players[i].ID == playerID {
			myPlayer = &s.Players[i]
			break
		}
	}

	pubPlayers := make([]publicPlayer, len(s.Players))
	for i, p := range s.Players {
		pp := publicPlayer{ID: p.ID, Seat: p.Seat, Name: p.Name, Alive: p.Alive}
		if !p.Alive {
			pp.Role = p.Role // dead players' roles are public
		}
		if myPlayer != nil && myPlayer.Role == RoleClawedWolf && p.Role == RoleClawedWolf {
			pp.Role = p.Role // wolves see each other
		}
		pubPlayers[i] = pp
	}

	view := map[string]interface{}{
		"players":         pubPlayers,
		"phase":           s.Phase,
		"round":           s.Round,
		"speeches":        s.DaySpeeches,
		"current_speaker": currentSpeakerSeat(s),
		"winner":          s.Winner,
	}

	if myPlayer != nil {
		view["your_role"] = myPlayer.Role
		view["your_seat"] = myPlayer.Seat
		view["your_alive"] = myPlayer.Alive
		if myPlayer.Role == RoleSeer {
			view["seer_results"] = s.SeerResults
		}
		if myPlayer.Role == RoleClawedWolf {
			view["wolf_speeches"] = s.WolfSpeeches
			view["wolf_vote_round"] = s.WolfVoteRound
		}
	}

	// Add public vote results if in day_result or later
	if s.Phase == PhaseDayResult || s.Phase == PhaseFinished {
		view["day_votes"] = s.DayVotes
	}

	return json.Marshal(view)
}

func (e *Engine) GetSpectatorView(raw json.RawMessage) (json.RawMessage, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	type publicPlayer struct {
		ID    uint   `json:"id"`
		Seat  int    `json:"seat"`
		Name  string `json:"name,omitempty"`
		Alive bool   `json:"alive"`
		Role  string `json:"role,omitempty"`
	}
	pubPlayers := make([]publicPlayer, len(s.Players))
	for i, p := range s.Players {
		pp := publicPlayer{ID: p.ID, Seat: p.Seat, Name: p.Name, Alive: p.Alive}
		if !p.Alive {
			pp.Role = p.Role
		}
		pubPlayers[i] = pp
	}
	view := map[string]interface{}{
		"players":         pubPlayers,
		"phase":           s.Phase,
		"round":           s.Round,
		"speeches":        s.DaySpeeches,
		"current_speaker": currentSpeakerSeat(s),
		"day_votes":       s.DayVotes,
		"winner":          s.Winner,
	}
	return json.Marshal(view)
}

func (e *Engine) GetGodView(raw json.RawMessage) (json.RawMessage, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	view := map[string]interface{}{
		"players":            s.Players,
		"phase":              s.Phase,
		"round":              s.Round,
		"speeches":           s.DaySpeeches,
		"day_votes":          s.DayVotes,
		"seer_results":       s.SeerResults,
		"night_kill_target":  s.NightKillTarget,
		"night_guard_target": s.NightGuardTarget,
		"phase_actions":      s.PhaseActions,
		"wolf_speeches":      s.WolfSpeeches,
		"wolf_vote_round":    s.WolfVoteRound,
		"winner":             s.Winner,
	}
	return json.Marshal(view)
}

// ---------------------------------------------------------------------------
// currentSpeakerSeat returns the seat of the player who is currently speaking
// during the day_discuss phase, or nil if not applicable.
// ---------------------------------------------------------------------------

func currentSpeakerSeat(s *State) *int {
	if s.Phase != PhaseDayDiscuss {
		return nil
	}
	spoken := map[int]bool{}
	for _, sp := range s.DaySpeeches {
		spoken[sp.Seat] = true
	}
	for i := 0; i < len(s.Players); i++ {
		seat := (s.SpeakStartSeat + s.SpeakerIndex + i) % len(s.Players)
		p := playerBySeat(s, seat)
		if p != nil && p.Alive && !spoken[seat] {
			return &seat
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// GetPendingActions (unchanged logic)
// ---------------------------------------------------------------------------

func (e *Engine) GetPendingActions(raw json.RawMessage) ([]game.PendingAction, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	if s.Winner != nil {
		return nil, nil
	}
	return pendingActionsForPhase(s), nil
}

func pendingActionsForPhase(s *State) []game.PendingAction {
	var actions []game.PendingAction
	aliveSeats := alivePlayerSeats(s)

	switch s.Phase {
	case PhaseNightClawedWolf:
		for _, p := range s.Players {
			if p.Alive && p.Role == RoleClawedWolf {
				if _, done := s.PhaseActions[fmt.Sprintf("%d", p.Seat)]; !done {
					targets := targetsExcluding(s, []int{})
					actions = append(actions, game.PendingAction{
						PlayerID:     p.ID,
						ActionType:   "kill_vote",
						Prompt:       fmt.Sprintf("Choose a player to kill tonight (vote round %d/%d).", s.wolfVoteRound(), maxWolfVoteRounds),
						ValidTargets: targets,
					})
				}
			}
		}

	case PhaseNightWolfDiscuss:
		spoken := map[int]bool{}
		for _, sp := range s.WolfSpeeches {
			spoken[sp.Seat] = true
		}
		for _, p := range s.Players {
			if p.Alive && p.Role == RoleClawedWolf && !spoken[p.Seat] {
				actions = append(actions, game.PendingAction{
					PlayerID:   p.ID,
					ActionType: "wolf_speak",
					Prompt:     fmt.Sprintf("Discuss with your partner who to kill (round %d/%d). You disagreed on the target.", s.wolfVoteRound()-1, maxWolfVoteRounds),
				})
				break
			}
		}

	case PhaseNightSeer:
		for _, p := range s.Players {
			if p.Alive && p.Role == RoleSeer {
				if _, done := s.PhaseActions["seer"]; !done {
					targets := targetsExcluding(s, []int{p.Seat})
					actions = append(actions, game.PendingAction{
						PlayerID:     p.ID,
						ActionType:   "investigate",
						Prompt:       "Choose a player to investigate.",
						ValidTargets: targets,
					})
				}
			}
		}

	case PhaseNightGuard:
		for _, p := range s.Players {
			if p.Alive && p.Role == RoleGuard {
				if _, done := s.PhaseActions["guard"]; !done {
					var exclude []int
					if s.LastGuardTarget != nil {
						exclude = append(exclude, *s.LastGuardTarget)
					}
					targets := targetsExcluding(s, exclude)
					actions = append(actions, game.PendingAction{
						PlayerID:     p.ID,
						ActionType:   "protect",
						Prompt:       "Choose a player to protect tonight.",
						ValidTargets: targets,
					})
				}
			}
		}

	case PhaseDayDiscuss:
		spoken := map[int]bool{}
		for _, sp := range s.DaySpeeches {
			spoken[sp.Seat] = true
		}
		for i := 0; i < len(s.Players); i++ {
			seat := (s.SpeakStartSeat + s.SpeakerIndex + i) % len(s.Players)
			_ = aliveSeats
			p := playerBySeat(s, seat)
			if p != nil && p.Alive && !spoken[seat] {
				actions = append(actions, game.PendingAction{
					PlayerID:   p.ID,
					ActionType: "speak",
					Prompt:     "Share your thoughts. Try to identify the clawed wolves.",
				})
				break
			}
		}

	case PhaseDayVote:
		for _, p := range s.Players {
			if p.Alive {
				if _, done := s.DayVotes[fmt.Sprintf("%d", p.Seat)]; !done {
					targets := targetsExcluding(s, []int{p.Seat})
					actions = append(actions, game.PendingAction{
						PlayerID:     p.ID,
						ActionType:   "vote",
						Prompt:       "Vote to eliminate a player (or abstain with target_seat: -1).",
						ValidTargets: append(targets, -1),
					})
				}
			}
		}
	}
	return actions
}

// ---------------------------------------------------------------------------
// Utility helpers (unchanged)
// ---------------------------------------------------------------------------

func alivePlayerSeats(s *State) []int {
	var seats []int
	for _, p := range s.Players {
		if p.Alive {
			seats = append(seats, p.Seat)
		}
	}
	return seats
}

func targetsExcluding(s *State, exclude []int) []int {
	excl := map[int]bool{}
	for _, e := range exclude {
		excl[e] = true
	}
	var targets []int
	for _, p := range s.Players {
		if p.Alive && !excl[p.Seat] {
			targets = append(targets, p.Seat)
		}
	}
	return targets
}

func playerBySeat(s *State, seat int) *Player {
	for i := range s.Players {
		if s.Players[i].Seat == seat {
			return &s.Players[i]
		}
	}
	return nil
}

func nameOfSeat(s *State, seat int) string {
	p := playerBySeat(s, seat)
	if p != nil && p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("Seat %d", seat)
}

// ---------------------------------------------------------------------------
// ApplyAction
// ---------------------------------------------------------------------------

type actionPayload struct {
	Type       string `json:"type"`
	TargetSeat *int   `json:"target_seat"`
	Message    string `json:"message"`
}

func (e *Engine) ApplyAction(raw json.RawMessage, playerID uint, actionRaw json.RawMessage) (*game.ApplyResult, error) {
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	if s.Winner != nil {
		return nil, errors.New("game is already over")
	}

	var action actionPayload
	if err := json.Unmarshal(actionRaw, &action); err != nil {
		return nil, fmt.Errorf("invalid action: %w", err)
	}

	// Find the acting player
	var actor *Player
	for i := range s.Players {
		if s.Players[i].ID == playerID {
			actor = &s.Players[i]
			break
		}
	}
	if actor == nil {
		return nil, errors.New("player not found")
	}
	if !actor.Alive {
		return nil, errors.New("dead players cannot act")
	}

	var events []game.GameEvent

	switch s.Phase {
	case PhaseNightClawedWolf:
		if action.Type != "kill_vote" || actor.Role != RoleClawedWolf {
			return nil, errors.New("invalid action for this phase")
		}
		if action.TargetSeat == nil {
			return nil, errors.New("target_seat is required")
		}
		target := playerBySeat(s, *action.TargetSeat)
		if target == nil || !target.Alive {
			return nil, errors.New("invalid target")
		}
		s.PhaseActions[fmt.Sprintf("%d", actor.Seat)] = *action.TargetSeat

		// Agent kill_vote event (snapshot after recording the vote)
		events = append(events, game.GameEvent{
			Source:    "agent",
			EventType: "kill_vote",
			Actor: &game.EventEntity{
				AgentID: uintPtr(actor.ID),
				Seat:    intPtr(actor.Seat),
				Team:    "wolf",
			},
			Target: &game.EventEntity{
				Seat: action.TargetSeat,
			},
			StateAfter: stateSnapshot(s),
			Visibility: "team:wolf",
		})

		// Check if all alive wolves have voted
		aliveWolves := aliveByRole(s, RoleClawedWolf)
		if len(s.PhaseActions) >= len(aliveWolves) {
			// Collect distinct votes
			votes := map[int]bool{}
			for _, p := range s.Players {
				if p.Alive && p.Role == RoleClawedWolf {
					if v, ok := s.PhaseActions[fmt.Sprintf("%d", p.Seat)]; ok {
						votes[v] = true
					}
				}
			}

			if len(votes) == 1 {
				// Wolves agree — set target and advance
				for target := range votes {
					s.NightKillTarget = &target
				}
				s.PhaseActions = map[string]int{}
				s.WolfSpeeches = nil
				s.WolfVoteRound = 1
				s.Phase = advanceNightPhase(s, PhaseNightSeer)

				events = append(events, game.GameEvent{
					Source:     "system",
					EventType:  "phase_change",
					Details:    mustJSON(map[string]any{"phase": s.Phase, "round": s.Round}),
					StateAfter: stateSnapshot(s),
					Visibility: "public",
				})
			} else if s.wolfVoteRound() >= maxWolfVoteRounds {
				// Max rounds reached — random pick from the votes
				targets := make([]int, 0, len(votes))
				for t := range votes {
					targets = append(targets, t)
				}
				picked := targets[rand.Intn(len(targets))]
				s.NightKillTarget = &picked
				s.PhaseActions = map[string]int{}
				s.WolfSpeeches = nil
				s.WolfVoteRound = 1
				s.Phase = advanceNightPhase(s, PhaseNightSeer)

				events = append(events, game.GameEvent{
					Source:     "system",
					EventType:  "wolf_vote_random",
					Details:    mustJSON(map[string]any{"picked_seat": picked, "candidates": targets}),
					StateAfter: stateSnapshot(s),
					Visibility: "team:wolf",
				})
				events = append(events, game.GameEvent{
					Source:     "system",
					EventType:  "phase_change",
					Details:    mustJSON(map[string]any{"phase": s.Phase, "round": s.Round}),
					StateAfter: stateSnapshot(s),
					Visibility: "public",
				})
			} else {
				// Disagree — enter wolf discussion phase
				s.PhaseActions = map[string]int{}
				s.WolfSpeeches = nil
				s.Phase = PhaseNightWolfDiscuss

				events = append(events, game.GameEvent{
					Source:     "system",
					EventType:  "wolf_vote_disagree",
					Details:    mustJSON(map[string]any{"vote_round": s.wolfVoteRound(), "max_rounds": maxWolfVoteRounds}),
					StateAfter: stateSnapshot(s),
					Visibility: "team:wolf",
				})
			}
		}

	case PhaseNightWolfDiscuss:
		if action.Type != "wolf_speak" || actor.Role != RoleClawedWolf {
			return nil, errors.New("invalid action for this phase")
		}
		s.WolfSpeeches = append(s.WolfSpeeches, Speech{
			Seat:    actor.Seat,
			Name:    actor.Name,
			Message: action.Message,
		})

		events = append(events, game.GameEvent{
			Source:    "agent",
			EventType: "wolf_speak",
			Actor: &game.EventEntity{
				AgentID: uintPtr(actor.ID),
				Seat:    intPtr(actor.Seat),
				Team:    "wolf",
			},
			Details:    mustJSON(map[string]any{"content": action.Message}),
			StateAfter: stateSnapshot(s),
			Visibility: "team:wolf",
		})

		// Check if all alive wolves have spoken
		spoken := map[int]bool{}
		for _, sp := range s.WolfSpeeches {
			spoken[sp.Seat] = true
		}
		allSpoken := true
		for _, p := range s.Players {
			if p.Alive && p.Role == RoleClawedWolf && !spoken[p.Seat] {
				allSpoken = false
				break
			}
		}
		if allSpoken {
			// Move to next vote round
			s.WolfVoteRound++
			s.WolfSpeeches = nil
			s.PhaseActions = map[string]int{}
			s.Phase = PhaseNightClawedWolf

			events = append(events, game.GameEvent{
				Source:     "system",
				EventType:  "phase_change",
				Details:    mustJSON(map[string]any{"phase": s.Phase, "round": s.Round, "wolf_vote_round": s.WolfVoteRound}),
				StateAfter: stateSnapshot(s),
				Visibility: "team:wolf",
			})
		}

	case PhaseNightSeer:
		if action.Type != "investigate" || actor.Role != RoleSeer {
			return nil, errors.New("invalid action for this phase")
		}
		if action.TargetSeat == nil {
			return nil, errors.New("target_seat is required")
		}
		target := playerBySeat(s, *action.TargetSeat)
		if target == nil || !target.Alive {
			return nil, errors.New("invalid target")
		}
		alignment := "good"
		if target.Role == RoleClawedWolf {
			alignment = "evil"
		}
		s.SeerResults[*action.TargetSeat] = alignment
		s.PhaseActions["seer"] = *action.TargetSeat

		// Agent investigate event
		events = append(events, game.GameEvent{
			Source:    "agent",
			EventType: "investigate",
			Actor: &game.EventEntity{
				AgentID: uintPtr(actor.ID),
				Seat:    intPtr(actor.Seat),
				Role:    RoleSeer,
			},
			Target: &game.EventEntity{
				Seat: action.TargetSeat,
			},
			Details:    mustJSON(map[string]any{"alignment": alignment}),
			StateAfter: stateSnapshot(s),
			Visibility: fmt.Sprintf("player:%d", playerID),
		})

		s.Phase = advanceNightPhase(s, PhaseNightGuard)

		// Phase change event
		events = append(events, game.GameEvent{
			Source:    "system",
			EventType: "phase_change",
			Details:   mustJSON(map[string]any{"phase": s.Phase, "round": s.Round}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		})

	case PhaseNightGuard:
		if action.Type != "protect" || actor.Role != RoleGuard {
			return nil, errors.New("invalid action for this phase")
		}
		if action.TargetSeat == nil {
			return nil, errors.New("target_seat is required")
		}
		if s.LastGuardTarget != nil && *s.LastGuardTarget == *action.TargetSeat {
			return nil, errors.New("cannot protect the same player consecutively")
		}
		target := playerBySeat(s, *action.TargetSeat)
		if target == nil || !target.Alive {
			return nil, errors.New("invalid target")
		}
		s.NightGuardTarget = action.TargetSeat
		s.PhaseActions["guard"] = *action.TargetSeat

		// Agent protect event
		events = append(events, game.GameEvent{
			Source:    "agent",
			EventType: "protect",
			Actor: &game.EventEntity{
				AgentID: uintPtr(actor.ID),
				Seat:    intPtr(actor.Seat),
				Role:    RoleGuard,
			},
			Target: &game.EventEntity{
				Seat: action.TargetSeat,
			},
			StateAfter: stateSnapshot(s),
			Visibility: fmt.Sprintf("player:%d", playerID),
		})

		// Resolve night, then advance to day
		events = append(events, resolveNight(s)...)

		// Only transition to day if the game hasn't ended
		if s.Winner == nil {
			s.Phase = PhaseDayDiscuss
			s.SpeakerIndex = 0

			// Phase change to day_discuss
			events = append(events, game.GameEvent{
				Source:    "system",
				EventType: "phase_change",
				Details:   mustJSON(map[string]any{"phase": s.Phase, "round": s.Round}),
				StateAfter: stateSnapshot(s),
				Visibility: "public",
			})
		}

	case PhaseDayDiscuss:
		if action.Type != "speak" {
			return nil, errors.New("invalid action for this phase")
		}
		s.DaySpeeches = append(s.DaySpeeches, Speech{
			Seat:    actor.Seat,
			Name:    actor.Name,
			Message: action.Message,
		})

		// Agent speak event
		events = append(events, game.GameEvent{
			Source:    "agent",
			EventType: "speak",
			Actor: &game.EventEntity{
				AgentID: uintPtr(actor.ID),
				Seat:    intPtr(actor.Seat),
			},
			Details:    mustJSON(map[string]any{"content": action.Message}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		})

		// Check if all alive players have spoken
		spoken := map[int]bool{}
		for _, sp := range s.DaySpeeches {
			spoken[sp.Seat] = true
		}
		allSpoken := true
		for _, p := range s.Players {
			if p.Alive && !spoken[p.Seat] {
				allSpoken = false
				break
			}
		}
		if allSpoken {
			s.Phase = PhaseDayVote
			s.DayVotes = map[string]int{}

			// Phase change event
			events = append(events, game.GameEvent{
				Source:    "system",
				EventType: "phase_change",
				Details:   mustJSON(map[string]any{"phase": s.Phase, "round": s.Round}),
				StateAfter: stateSnapshot(s),
				Visibility: "public",
			})
		}

	case PhaseDayVote:
		if action.Type != "vote" {
			return nil, errors.New("invalid action for this phase")
		}
		targetVal := -1
		if action.TargetSeat != nil {
			targetVal = *action.TargetSeat
		}
		if targetVal >= 0 {
			target := playerBySeat(s, targetVal)
			if target == nil || !target.Alive {
				return nil, errors.New("invalid vote target")
			}
			if targetVal == actor.Seat {
				return nil, errors.New("cannot vote for yourself")
			}
		}
		s.DayVotes[fmt.Sprintf("%d", actor.Seat)] = targetVal

		// Agent vote event
		voteEvt := game.GameEvent{
			Source:    "agent",
			EventType: "vote",
			Actor: &game.EventEntity{
				AgentID: uintPtr(actor.ID),
				Seat:    intPtr(actor.Seat),
			},
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		}
		if targetVal >= 0 {
			voteEvt.Target = &game.EventEntity{Seat: intPtr(targetVal)}
		}
		events = append(events, voteEvt)

		// Check if all alive players have voted
		aliveCount := countAlive(s)
		if len(s.DayVotes) >= aliveCount {
			events = append(events, resolveVote(s)...)
			if s.Winner != nil {
				s.Phase = PhaseFinished
			} else {
				// Start next round
				s.Round++
				s.DaySpeeches = nil
				s.DayVotes = map[string]int{}
				s.PhaseActions = map[string]int{}
				s.NightKillTarget = nil
				s.NightGuardTarget = nil
				s.WolfSpeeches = nil
				s.WolfVoteRound = 1
				s.SpeakStartSeat = (s.SpeakStartSeat + 1) % len(s.Players)
				s.SpeakerIndex = 0
				s.Phase = PhaseNightClawedWolf

				events = append(events, game.GameEvent{
					Source:    "system",
					EventType: "phase_change",
					Details:   mustJSON(map[string]any{"phase": s.Phase, "round": s.Round}),
					StateAfter: stateSnapshot(s),
					Visibility: "public",
				})
			}
		}

	default:
		return nil, fmt.Errorf("no actions expected in phase: %s", s.Phase)
	}

	return &game.ApplyResult{Events: events}, nil
}

// ---------------------------------------------------------------------------
// Night phase advancement
// ---------------------------------------------------------------------------

func advanceNightPhase(s *State, next string) string {
	switch next {
	case PhaseNightSeer:
		if !hasAliveRole(s, RoleSeer) {
			return advanceNightPhase(s, PhaseNightGuard)
		}
	case PhaseNightGuard:
		if !hasAliveRole(s, RoleGuard) {
			// Guard is dead: resolve night immediately and go to day
			// (resolveNight is called by the caller after phase advancement,
			// but when skipping guard we need to handle it here.)
			// We return PhaseDayDiscuss; the caller must call resolveNight.
			return PhaseDayDiscuss
		}
	}
	return next
}

func hasAliveRole(s *State, role string) bool {
	for _, p := range s.Players {
		if p.Alive && p.Role == role {
			return true
		}
	}
	return false
}

func aliveByRole(s *State, role string) []Player {
	var result []Player
	for _, p := range s.Players {
		if p.Alive && p.Role == role {
			result = append(result, p)
		}
	}
	return result
}

func countAlive(s *State) int {
	count := 0
	for _, p := range s.Players {
		if p.Alive {
			count++
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// resolveNight — applies the night kill with guard protection check.
// Mutates state in place and returns events with progressive snapshots.
// ---------------------------------------------------------------------------

func resolveNight(s *State) []game.GameEvent {
	var events []game.GameEvent
	if s.NightKillTarget == nil {
		// No kill target: emit a night_resolve event with null killed_seat
		events = append(events, game.GameEvent{
			Source:    "system",
			EventType: "night_resolve",
			Details:   mustJSON(map[string]any{"killed_seat": nil, "guarded": s.NightGuardTarget != nil}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		})
		s.LastGuardTarget = s.NightGuardTarget
		return events
	}

	target := *s.NightKillTarget
	saved := s.NightGuardTarget != nil && *s.NightGuardTarget == target

	if saved {
		// Guard save event — do not reveal who was saved (public information)
		events = append(events, game.GameEvent{
			Source:    "system",
			EventType: "guard_save",
			Details:   mustJSON(map[string]any{"guarded": true}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		})

		// Night resolve event (nobody died)
		events = append(events, game.GameEvent{
			Source:    "system",
			EventType: "night_resolve",
			Details:   mustJSON(map[string]any{"killed_seat": nil, "guarded": true}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		})
	} else {
		p := playerBySeat(s, target)
		if p != nil && p.Alive {
			// Kill the player — mutate state first
			for i := range s.Players {
				if s.Players[i].Seat == target {
					s.Players[i].Alive = false
					break
				}
			}
			s.Eliminated = append(s.Eliminated, target)

			// Night resolve event
			events = append(events, game.GameEvent{
				Source:    "system",
				EventType: "night_resolve",
				Details:   mustJSON(map[string]any{"killed_seat": target, "guarded": false}),
				StateAfter: stateSnapshot(s),
				Visibility: "public",
			})

			// Death event
			events = append(events, game.GameEvent{
				Source:    "system",
				EventType: "death",
				Target: &game.EventEntity{
					AgentID: uintPtr(p.ID),
					Seat:    intPtr(target),
				},
				Details:    mustJSON(map[string]any{"cause": "night_kill", "role_reveal": p.Role}),
				StateAfter: stateSnapshot(s),
				Visibility: "public",
			})
		}
	}

	s.LastGuardTarget = s.NightGuardTarget
	checkWinCondition(s, &events)
	return events
}

// ---------------------------------------------------------------------------
// resolveVote — eliminates the player with the most votes.
// Mutates state in place and returns events with progressive snapshots.
// ---------------------------------------------------------------------------

func resolveVote(s *State) []game.GameEvent {
	var events []game.GameEvent
	tally := map[int]int{}
	for _, target := range s.DayVotes {
		if target >= 0 {
			tally[target]++
		}
	}

	// Find max votes
	maxVotes := 0
	for _, count := range tally {
		if count > maxVotes {
			maxVotes = count
		}
	}

	// Find all with max votes
	var candidates []int
	for seat, count := range tally {
		if count == maxVotes {
			candidates = append(candidates, seat)
		}
	}

	if maxVotes == 0 || len(candidates) != 1 {
		// No consensus
		events = append(events, game.GameEvent{
			Source:    "system",
			EventType: "vote_result",
			Details:   mustJSON(map[string]any{"tally": tally, "eliminated": false}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		})
		return events
	}

	eliminated := candidates[0]
	p := playerBySeat(s, eliminated)
	if p != nil && p.Alive {
		// Kill the player — mutate state first
		for i := range s.Players {
			if s.Players[i].Seat == eliminated {
				s.Players[i].Alive = false
				break
			}
		}
		s.Eliminated = append(s.Eliminated, eliminated)

		// Vote result event
		events = append(events, game.GameEvent{
			Source:    "system",
			EventType: "vote_result",
			Target: &game.EventEntity{
				AgentID: uintPtr(p.ID),
				Seat:    intPtr(eliminated),
			},
			Details:    mustJSON(map[string]any{"tally": tally, "eliminated": true}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		})

		// Death event
		events = append(events, game.GameEvent{
			Source:    "system",
			EventType: "death",
			Target: &game.EventEntity{
				AgentID: uintPtr(p.ID),
				Seat:    intPtr(eliminated),
			},
			Details:    mustJSON(map[string]any{"cause": "vote_elimination", "role_reveal": p.Role}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
		})
	}

	checkWinCondition(s, &events)
	return events
}

// ---------------------------------------------------------------------------
// Win condition check — appends game_over event if the game is over.
// ---------------------------------------------------------------------------

func checkWinCondition(s *State, events *[]game.GameEvent) {
	aliveWolves := 0
	aliveGood := 0
	for _, p := range s.Players {
		if p.Alive {
			if p.Role == RoleClawedWolf {
				aliveWolves++
			} else {
				aliveGood++
			}
		}
	}

	if aliveWolves == 0 {
		winner := "good"
		s.Winner = &winner
		s.Phase = PhaseFinished

		var winnerIDs []uint
		for _, p := range s.Players {
			if p.Role != RoleClawedWolf {
				winnerIDs = append(winnerIDs, p.ID)
			}
		}

		*events = append(*events, game.GameEvent{
			Source:    "system",
			EventType: "game_over",
			Details:   mustJSON(map[string]any{"winner_team": "good", "winner_ids": winnerIDs}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
			GameOver:  true,
			Result: &game.GameResult{
				WinnerIDs:  winnerIDs,
				WinnerTeam: "good",
			},
		})
	} else if aliveWolves >= aliveGood {
		winner := "evil"
		s.Winner = &winner
		s.Phase = PhaseFinished

		var winnerIDs []uint
		for _, p := range s.Players {
			if p.Role == RoleClawedWolf {
				winnerIDs = append(winnerIDs, p.ID)
			}
		}

		*events = append(*events, game.GameEvent{
			Source:    "system",
			EventType: "game_over",
			Details:   mustJSON(map[string]any{"winner_team": "evil", "winner_ids": winnerIDs}),
			StateAfter: stateSnapshot(s),
			Visibility: "public",
			GameOver:  true,
			Result: &game.GameResult{
				WinnerIDs:  winnerIDs,
				WinnerTeam: "evil",
			},
		})
	}
}

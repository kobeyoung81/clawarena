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
	PhaseNightClawedWolf = "night_clawedwolf"
	PhaseNightSeer     = "night_seer"
	PhaseNightGuard    = "night_guard"
	PhaseDayAnnounce   = "day_announce"
	PhaseDayDiscuss    = "day_discuss"
	PhaseDayVote       = "day_vote"
	PhaseDayResult     = "day_result"
	PhaseFinished      = "finished"

	RoleClawedWolf = "clawedwolf"
	RoleSeer     = "seer"
	RoleGuard    = "guard"
	RoleVillager = "villager"
)

type Player struct {
	ID        uint   `json:"id"`
	Seat      int    `json:"seat"`
	Role      string `json:"role"`
	Alive     bool   `json:"alive"`
	LastWords string `json:"last_words,omitempty"`
}

type State struct {
	Players          []Player          `json:"players"`
	Phase            string            `json:"phase"`
	Round            int               `json:"round"`
	PhaseActions     map[string]int    `json:"phase_actions"` // seat -> target_seat (for night votes)
	NightKillTarget  *int              `json:"night_kill_target"`
	NightGuardTarget *int              `json:"night_guard_target"`
	LastGuardTarget  *int              `json:"last_guard_target"`
	SeerResults      map[int]string    `json:"seer_results"` // seat -> "good"/"evil"
	DaySpeeches      []Speech          `json:"day_speeches"`
	DayVotes         map[string]int    `json:"day_votes"` // voter seat -> target seat (-1=abstain)
	Events           []game.GameEvent  `json:"events"`
	Eliminated       []int             `json:"eliminated"`
	SpeakerIndex     int               `json:"speaker_index"` // which alive player speaks next
	SpeakStartSeat   int               `json:"speak_start_seat"`
	Winner           *string           `json:"winner,omitempty"`
}

type Speech struct {
	Seat    int    `json:"seat"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type Engine struct{}

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

func (e *Engine) InitState(_ json.RawMessage, players []uint) (json.RawMessage, error) {
	if len(players) != 6 {
		return nil, errors.New("clawedwolf requires exactly 6 players")
	}
	roles := []string{RoleClawedWolf, RoleClawedWolf, RoleSeer, RoleGuard, RoleVillager, RoleVillager}
	perm := rand.Perm(6)
	ps := make([]Player, 6)
	for i, pid := range players {
		ps[i] = Player{ID: pid, Seat: i, Role: roles[perm[i]], Alive: true}
	}
	s := State{
		Players:      ps,
		Phase:        PhaseNightClawedWolf,
		Round:        1,
		PhaseActions: map[string]int{},
		SeerResults:  map[int]string{},
		DayVotes:     map[string]int{},
		Events: []game.GameEvent{{
			Type:       "game_start",
			Message:    "Game started. Night 1 begins.",
			Visibility: "public",
		}},
		SpeakStartSeat: 0,
	}
	return json.Marshal(s)
}

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
		pp := publicPlayer{ID: p.ID, Seat: p.Seat, Alive: p.Alive}
		if !p.Alive {
			pp.Role = p.Role // dead players' roles are public
		}
		if myPlayer != nil && myPlayer.Role == RoleClawedWolf && p.Role == RoleClawedWolf {
			pp.Role = p.Role // wolves see each other
		}
		pubPlayers[i] = pp
	}

	view := map[string]interface{}{
		"players":  pubPlayers,
		"phase":    s.Phase,
		"round":    s.Round,
		"events":   s.Events,
		"speeches": s.DaySpeeches,
		"winner":   s.Winner,
	}

	if myPlayer != nil {
		view["your_role"] = myPlayer.Role
		view["your_seat"] = myPlayer.Seat
		view["your_alive"] = myPlayer.Alive
		if myPlayer.Role == RoleSeer {
			view["seer_results"] = s.SeerResults
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
		Alive bool   `json:"alive"`
		Role  string `json:"role,omitempty"`
	}
	pubPlayers := make([]publicPlayer, len(s.Players))
	for i, p := range s.Players {
		pp := publicPlayer{ID: p.ID, Seat: p.Seat, Alive: p.Alive}
		if !p.Alive {
			pp.Role = p.Role
		}
		pubPlayers[i] = pp
	}
	view := map[string]interface{}{
		"players":  pubPlayers,
		"phase":    s.Phase,
		"round":    s.Round,
		"events":   s.Events,
		"speeches": s.DaySpeeches,
		"winner":   s.Winner,
	}
	return json.Marshal(view)
}

func (e *Engine) GetGodView(raw json.RawMessage) (json.RawMessage, error) {
	// Return full state with all roles revealed
	s, err := parseState(raw)
	if err != nil {
		return nil, err
	}
	view := map[string]interface{}{
		"players":           s.Players,
		"phase":             s.Phase,
		"round":             s.Round,
		"events":            s.Events,
		"speeches":          s.DaySpeeches,
		"day_votes":         s.DayVotes,
		"seer_results":      s.SeerResults,
		"night_kill_target": s.NightKillTarget,
		"night_guard_target": s.NightGuardTarget,
		"phase_actions":     s.PhaseActions,
		"winner":            s.Winner,
	}
	return json.Marshal(view)
}

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
						Prompt:       "Choose a player to kill tonight.",
						ValidTargets: targets,
					})
				}
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
		// Find next speaker (round-robin from speak_start_seat among alive players)
		spoken := map[int]bool{}
		for _, sp := range s.DaySpeeches {
			spoken[sp.Seat] = true
		}
		// Find next alive player who hasn't spoken
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

type actionPayload struct {
	Type       string  `json:"type"`
	TargetSeat *int    `json:"target_seat"`
	Message    string  `json:"message"`
}

func (e *Engine) ApplyAction(raw json.RawMessage, playerID uint, actionRaw json.RawMessage) (game.ActionResult, error) {
	s, err := parseState(raw)
	if err != nil {
		return game.ActionResult{}, err
	}
	if s.Winner != nil {
		return game.ActionResult{}, errors.New("game is already over")
	}

	var action actionPayload
	if err := json.Unmarshal(actionRaw, &action); err != nil {
		return game.ActionResult{}, fmt.Errorf("invalid action: %w", err)
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
		return game.ActionResult{}, errors.New("player not found")
	}
	if !actor.Alive {
		return game.ActionResult{}, errors.New("dead players cannot act")
	}

	var newEvents []game.GameEvent

	switch s.Phase {
	case PhaseNightClawedWolf:
		if action.Type != "kill_vote" || actor.Role != RoleClawedWolf {
			return game.ActionResult{}, errors.New("invalid action for this phase")
		}
		if action.TargetSeat == nil {
			return game.ActionResult{}, errors.New("target_seat is required")
		}
		target := playerBySeat(s, *action.TargetSeat)
		if target == nil || !target.Alive {
			return game.ActionResult{}, errors.New("invalid target")
		}
		s.PhaseActions[fmt.Sprintf("%d", actor.Seat)] = *action.TargetSeat

		// Check if all alive wolves have voted
		aliveWolves := aliveByRole(s, RoleClawedWolf)
		if len(s.PhaseActions) >= len(aliveWolves) {
			// Resolve: first wolf's choice wins on disagreement
			firstVote := -1
			for _, p := range s.Players {
				if p.Alive && p.Role == RoleClawedWolf {
					if v, ok := s.PhaseActions[fmt.Sprintf("%d", p.Seat)]; ok {
						if firstVote == -1 {
							firstVote = v
						}
						// Check if all agree
					}
				}
			}
			s.NightKillTarget = &firstVote
			s.PhaseActions = map[string]int{}
			s.Phase = advanceNightPhase(s, PhaseNightSeer)
		}

	case PhaseNightSeer:
		if action.Type != "investigate" || actor.Role != RoleSeer {
			return game.ActionResult{}, errors.New("invalid action for this phase")
		}
		if action.TargetSeat == nil {
			return game.ActionResult{}, errors.New("target_seat is required")
		}
		target := playerBySeat(s, *action.TargetSeat)
		if target == nil || !target.Alive {
			return game.ActionResult{}, errors.New("invalid target")
		}
		alignment := "good"
		if target.Role == RoleClawedWolf {
			alignment = "evil"
		}
		s.SeerResults[*action.TargetSeat] = alignment
		newEvents = append(newEvents, game.GameEvent{
			Type:       "seer_result",
			Message:    fmt.Sprintf("You investigated seat %d: they are %s.", *action.TargetSeat, alignment),
			Visibility: fmt.Sprintf("player:%d", playerID),
		})
		s.PhaseActions["seer"] = *action.TargetSeat
		s.Phase = advanceNightPhase(s, PhaseNightGuard)

	case PhaseNightGuard:
		if action.Type != "protect" || actor.Role != RoleGuard {
			return game.ActionResult{}, errors.New("invalid action for this phase")
		}
		if action.TargetSeat == nil {
			return game.ActionResult{}, errors.New("target_seat is required")
		}
		if s.LastGuardTarget != nil && *s.LastGuardTarget == *action.TargetSeat {
			return game.ActionResult{}, errors.New("cannot protect the same player consecutively")
		}
		target := playerBySeat(s, *action.TargetSeat)
		if target == nil || !target.Alive {
			return game.ActionResult{}, errors.New("invalid target")
		}
		s.NightGuardTarget = action.TargetSeat
		s.PhaseActions["guard"] = *action.TargetSeat
		// Advance to day announce — resolve night
		newEvents = append(newEvents, resolveNight(s)...)
		s.Phase = PhaseDayAnnounce
		// Immediately auto-advance day_announce to day_discuss
		s.Phase = PhaseDayDiscuss
		s.SpeakerIndex = 0

	case PhaseDayDiscuss:
		if action.Type != "speak" {
			return game.ActionResult{}, errors.New("invalid action for this phase")
		}
		s.DaySpeeches = append(s.DaySpeeches, Speech{
			Seat:    actor.Seat,
			Message: action.Message,
		})
		newEvents = append(newEvents, game.GameEvent{
			Type:       "speech",
			Message:    fmt.Sprintf("Seat %d: %s", actor.Seat, action.Message),
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
		}

	case PhaseDayVote:
		if action.Type != "vote" {
			return game.ActionResult{}, errors.New("invalid action for this phase")
		}
		targetVal := -1
		if action.TargetSeat != nil {
			targetVal = *action.TargetSeat
		}
		if targetVal >= 0 {
			target := playerBySeat(s, targetVal)
			if target == nil || !target.Alive {
				return game.ActionResult{}, errors.New("invalid vote target")
			}
			if targetVal == actor.Seat {
				return game.ActionResult{}, errors.New("cannot vote for yourself")
			}
		}
		s.DayVotes[fmt.Sprintf("%d", actor.Seat)] = targetVal

		// Check if all alive players have voted
		aliveCount := countAlive(s)
		if len(s.DayVotes) >= aliveCount {
			newEvents = append(newEvents, resolveVote(s)...)
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
				s.SpeakStartSeat = (s.SpeakStartSeat + 1) % len(s.Players)
				s.SpeakerIndex = 0
				s.Phase = PhaseNightClawedWolf
				newEvents = append(newEvents, game.GameEvent{
					Type:       "phase_change",
					Message:    fmt.Sprintf("Night %d begins.", s.Round),
					Visibility: "public",
				})
			}
		}

	default:
		return game.ActionResult{}, fmt.Errorf("no actions expected in phase: %s", s.Phase)
	}

	s.Events = append(s.Events, newEvents...)

	newStateRaw, err := json.Marshal(s)
	if err != nil {
		return game.ActionResult{}, err
	}

	result := game.ActionResult{
		NewState: newStateRaw,
		Events:   newEvents,
		GameOver: s.Winner != nil,
	}

	if s.Winner != nil {
		team := *s.Winner
		var winnerIDs []uint
		for _, p := range s.Players {
			if (team == "good" && p.Role != RoleClawedWolf) ||
				(team == "evil" && p.Role == RoleClawedWolf) {
				winnerIDs = append(winnerIDs, p.ID)
			}
		}
		result.Result = &game.GameResult{
			WinnerIDs:  winnerIDs,
			WinnerTeam: team,
		}
	}

	return result, nil
}

// advanceNightPhase advances to the next night phase, skipping if the role is dead.
func advanceNightPhase(s *State, next string) string {
	switch next {
	case PhaseNightSeer:
		if !hasAliveRole(s, RoleSeer) {
			return advanceNightPhase(s, PhaseNightGuard)
		}
	case PhaseNightGuard:
		if !hasAliveRole(s, RoleGuard) {
			resolveNight(s)
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

// resolveNight applies the night kill (with guard protection check).
func resolveNight(s *State) []game.GameEvent {
	var events []game.GameEvent
	if s.NightKillTarget == nil {
		return events
	}
	target := *s.NightKillTarget
	// Guard save?
	saved := s.NightGuardTarget != nil && *s.NightGuardTarget == target
	if saved {
		events = append(events, game.GameEvent{
			Type:       "guard_save",
			Message:    fmt.Sprintf("Someone was attacked but protected! Seat %d survived.", target),
			Visibility: "public",
		})
	} else {
		p := playerBySeat(s, target)
		if p != nil && p.Alive {
			p.Alive = false
			s.Eliminated = append(s.Eliminated, target)
			events = append(events, game.GameEvent{
				Type:       "death",
				Message:    fmt.Sprintf("Seat %d was killed during the night. They were a %s.", target, p.Role),
				Visibility: "public",
			})
			// Update in slice
			for i := range s.Players {
				if s.Players[i].Seat == target {
					s.Players[i].Alive = false
					break
				}
			}
		}
	}
	s.LastGuardTarget = s.NightGuardTarget
	checkWinCondition(s, &events)
	return events
}

// resolveVote eliminates the player with the most votes.
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
		events = append(events, game.GameEvent{
			Type:       "vote_result",
			Message:    "No consensus reached. Nobody is eliminated.",
			Visibility: "public",
		})
		return events
	}

	eliminated := candidates[0]
	p := playerBySeat(s, eliminated)
	if p != nil && p.Alive {
		for i := range s.Players {
			if s.Players[i].Seat == eliminated {
				s.Players[i].Alive = false
				break
			}
		}
		s.Eliminated = append(s.Eliminated, eliminated)
		events = append(events, game.GameEvent{
			Type:       "vote_result",
			Message:    fmt.Sprintf("Seat %d was voted out. They were a %s.", eliminated, p.Role),
			Visibility: "public",
		})
	}
	checkWinCondition(s, &events)
	return events
}

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
		*events = append(*events, game.GameEvent{
			Type:       "game_over",
			Message:    "All clawed wolves eliminated! Good team wins!",
			Visibility: "public",
		})
	} else if aliveWolves >= aliveGood {
		winner := "evil"
		s.Winner = &winner
		*events = append(*events, game.GameEvent{
			Type:       "game_over",
			Message:    "Clawed wolves outnumber the good players! Evil team wins!",
			Visibility: "public",
		})
	}
}

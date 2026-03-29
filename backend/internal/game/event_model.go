package game

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// GameEventRecord is the interface that per-game GORM event models must implement.
// Each game type embeds BaseGameEvent and overrides TableName().
type GameEventRecord interface {
	TableName() string
	SetFields(gameID uint, seq uint, evt GameEvent, createdAt time.Time)
	GetSeq() uint
	GetGameID() uint
	ToGameEvent() GameEvent
}

// BaseGameEvent is the GORM model embedded by each per-game event table.
// Table names follow the pattern: {syncronym}_game_events (e.g., ttt_game_events).
type BaseGameEvent struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	GameID     uint           `gorm:"not null;uniqueIndex:uq_game_seq" json:"game_id"`
	Seq        uint           `gorm:"not null;uniqueIndex:uq_game_seq" json:"seq"`
	Source     string         `gorm:"type:varchar(20);not null" json:"source"`
	EventType  string         `gorm:"type:varchar(50);not null" json:"event_type"`
	Actor      datatypes.JSON `gorm:"type:json" json:"actor"`
	Target     datatypes.JSON `gorm:"type:json" json:"target"`
	Details    datatypes.JSON `gorm:"type:json" json:"details"`
	StateAfter datatypes.JSON `gorm:"type:json;not null" json:"state_after"`
	Visibility string         `gorm:"type:varchar(30);not null;default:'public'" json:"visibility"`
	GameOver   bool           `gorm:"not null;default:false" json:"game_over"`
	Result     datatypes.JSON `gorm:"type:json" json:"result"`
	CreatedAt  time.Time      `json:"created_at"`
}

// SetFields populates a BaseGameEvent from a GameEvent plus metadata.
func (b *BaseGameEvent) SetFields(gameID uint, seq uint, evt GameEvent, createdAt time.Time) {
	b.GameID = gameID
	b.Seq = seq
	b.Source = evt.Source
	b.EventType = evt.EventType
	b.Visibility = evt.Visibility
	b.GameOver = evt.GameOver
	b.CreatedAt = createdAt

	if evt.Actor != nil {
		actorJSON, _ := json.Marshal(evt.Actor)
		b.Actor = datatypes.JSON(actorJSON)
	}
	if evt.Target != nil {
		targetJSON, _ := json.Marshal(evt.Target)
		b.Target = datatypes.JSON(targetJSON)
	}
	if evt.Details != nil {
		b.Details = datatypes.JSON(evt.Details)
	}
	if evt.StateAfter != nil {
		b.StateAfter = datatypes.JSON(evt.StateAfter)
	}
	if evt.Result != nil {
		resultJSON, _ := json.Marshal(evt.Result)
		b.Result = datatypes.JSON(resultJSON)
	}
}

// GetSeq returns the event sequence number.
func (b *BaseGameEvent) GetSeq() uint { return b.Seq }

// GetGameID returns the game ID this event belongs to.
func (b *BaseGameEvent) GetGameID() uint { return b.GameID }

// ToGameEvent converts the GORM model back to a GameEvent.
func (b *BaseGameEvent) ToGameEvent() GameEvent {
	evt := GameEvent{
		Source:     b.Source,
		EventType:  b.EventType,
		StateAfter: json.RawMessage(b.StateAfter),
		Visibility: b.Visibility,
		GameOver:   b.GameOver,
	}

	if len(b.Actor) > 0 && string(b.Actor) != "null" {
		var actor EventEntity
		if json.Unmarshal(b.Actor, &actor) == nil {
			evt.Actor = &actor
		}
	}
	if len(b.Target) > 0 && string(b.Target) != "null" {
		var target EventEntity
		if json.Unmarshal(b.Target, &target) == nil {
			evt.Target = &target
		}
	}
	if len(b.Details) > 0 && string(b.Details) != "null" {
		evt.Details = json.RawMessage(b.Details)
	}
	if len(b.Result) > 0 && string(b.Result) != "null" {
		var result GameResult
		if json.Unmarshal(b.Result, &result) == nil {
			evt.Result = &result
		}
	}

	return evt
}

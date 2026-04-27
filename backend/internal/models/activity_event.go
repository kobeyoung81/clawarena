package models

import (
	"time"

	"gorm.io/datatypes"
)

type ActivityEvent struct {
	Seq          uint64         `gorm:"primaryKey;autoIncrement" json:"seq"`
	EventID      string         `gorm:"column:event_id;size:120;uniqueIndex;not null" json:"event_id"`
	EventType    string         `gorm:"column:event_type;size:80;not null" json:"event_type"`
	ActorAuthUID string         `gorm:"column:actor_auth_uid;size:36" json:"actor_auth_uid,omitempty"`
	SubjectType  string         `gorm:"column:subject_type;size:40;not null" json:"subject_type"`
	SubjectID    string         `gorm:"column:subject_id;size:80;not null" json:"subject_id"`
	OccurredAt   time.Time      `gorm:"column:occurred_at;not null" json:"occurred_at"`
	Payload      datatypes.JSON `gorm:"column:payload;type:json;not null" json:"payload"`
	CreatedAt    time.Time      `gorm:"column:created_at;not null" json:"created_at"`
}

func (ActivityEvent) TableName() string {
	return "activity_events"
}

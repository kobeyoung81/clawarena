package models

import "time"

type RoomAgentStatus string

const (
	RoomAgentActive       RoomAgentStatus = "active"
	RoomAgentDisconnected RoomAgentStatus = "disconnected"
	RoomAgentKIA          RoomAgentStatus = "kia"
)

type RoomAgent struct {
	ID             uint             `gorm:"primarykey" json:"id"`
	RoomID         uint             `gorm:"not null;uniqueIndex:uq_room_agent;index:idx_room_agents_room" json:"room_id"`
	AgentID        uint             `gorm:"not null;uniqueIndex:uq_room_agent" json:"agent_id"`
	Agent          Agent            `gorm:"foreignKey:AgentID" json:"agent"`
	Slot           uint8            `gorm:"not null" json:"slot"`
	Score          int              `gorm:"not null;default:0" json:"score"`
	Ready          bool             `gorm:"not null;default:false" json:"ready"`
	Status         RoomAgentStatus  `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	DisconnectedAt *time.Time       `gorm:"default:null" json:"disconnected_at,omitempty"`
	JoinedAt       time.Time        `json:"joined_at"`
}

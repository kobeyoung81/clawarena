package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/clawarena/clawarena/internal/game"
	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ActivityFeedHandler struct {
	db *gorm.DB
}

type activityFeedEvent struct {
	Seq         uint64          `json:"seq"`
	EventID     string          `json:"event_id"`
	EventType   string          `json:"event_type"`
	SubjectType string          `json:"subject_type"`
	SubjectID   string          `json:"subject_id"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Payload     json.RawMessage `json:"payload"`
}

type activityFeedResponse struct {
	Source       string              `json:"source"`
	Events       []activityFeedEvent `json:"events"`
	NextAfterSeq uint64              `json:"next_after_seq"`
	HasMore      bool                `json:"has_more"`
}

type gameFinishedParticipant struct {
	AuthUID   string `json:"auth_uid"`
	Won       bool   `json:"won"`
	Forfeited bool   `json:"forfeited"`
}

type gameFinishedPayload struct {
	GameID              uint                      `json:"game_id"`
	RoomID              uint                      `json:"room_id"`
	GameType            string                    `json:"game_type"`
	CompletionReason    string                    `json:"completion_reason"`
	FinishedAt          time.Time                 `json:"finished_at"`
	AcceptedActionCount int64                     `json:"accepted_action_count"`
	Participants        []gameFinishedParticipant `json:"participants"`
}

func NewActivityFeedHandler(db *gorm.DB) *ActivityFeedHandler {
	return &ActivityFeedHandler{db: db}
}

func InternalFeedAuth(token string) func(http.Handler) http.Handler {
	expected := strings.TrimSpace(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expected == "" {
				writeError(w, http.StatusServiceUnavailable, "internal activity feed is not configured", "FEED_NOT_CONFIGURED")
				return
			}
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader != "Bearer "+expected {
				writeError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (h *ActivityFeedHandler) List(w http.ResponseWriter, r *http.Request) {
	afterSeq, err := parseUint64Query(r, "after_seq")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid after_seq", "INVALID_REQUEST")
		return
	}
	limit, err := parsePositiveIntQuery(r, "limit", 100, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit", "INVALID_REQUEST")
		return
	}

	var records []models.ActivityEvent
	if err := h.db.Where("seq > ?", afterSeq).Order("seq ASC").Limit(limit + 1).Find(&records).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load activity feed", "INTERNAL_ERROR")
		return
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}

	nextAfterSeq := afterSeq
	events := make([]activityFeedEvent, 0, len(records))
	for _, record := range records {
		nextAfterSeq = record.Seq
		payload := json.RawMessage(record.Payload)
		events = append(events, activityFeedEvent{
			Seq:         record.Seq,
			EventID:     record.EventID,
			EventType:   record.EventType,
			SubjectType: record.SubjectType,
			SubjectID:   record.SubjectID,
			OccurredAt:  record.OccurredAt,
			Payload:     payload,
		})
	}

	writeJSON(w, http.StatusOK, activityFeedResponse{
		Source:       "clawarena",
		Events:       events,
		NextAfterSeq: nextAfterSeq,
		HasMore:      hasMore,
	})
}

func EmitGameFinishedActivityEvent(tx *gorm.DB, gameID uint, completionReason string, finishedAt time.Time, winnerIDs []uint, forfeitedIDs map[uint]bool) error {
	var g models.Game
	if err := tx.Preload("GameType").Preload("Players.Agent").First(&g, gameID).Error; err != nil {
		return err
	}

	acceptedActionCount := int64(0)
	if eng := game.GetEngine(g.GameType.Name); eng != nil {
		tableName := eng.NewEventModel().TableName()
		if err := tx.Table(tableName).Where("game_id = ? AND source = ?", g.ID, "agent").Count(&acceptedActionCount).Error; err != nil {
			return err
		}
	}

	winnerSet := make(map[uint]bool, len(winnerIDs))
	for _, winnerID := range winnerIDs {
		winnerSet[winnerID] = true
	}

	participants := make([]gameFinishedParticipant, 0, len(g.Players))
	actorAuthUID := ""
	for _, player := range g.Players {
		forfeited := forfeitedIDs != nil && forfeitedIDs[player.AgentID]
		if forfeited && actorAuthUID == "" {
			actorAuthUID = player.Agent.AuthUID
		}
		participants = append(participants, gameFinishedParticipant{
			AuthUID:   player.Agent.AuthUID,
			Won:       winnerSet[player.AgentID],
			Forfeited: forfeited,
		})
	}

	payloadBytes, err := json.Marshal(gameFinishedPayload{
		GameID:              g.ID,
		RoomID:              g.RoomID,
		GameType:            g.GameType.Name,
		CompletionReason:    completionReason,
		FinishedAt:          finishedAt.UTC(),
		AcceptedActionCount: acceptedActionCount,
		Participants:        participants,
	})
	if err != nil {
		return err
	}

	event := models.ActivityEvent{
		EventID:      fmt.Sprintf("clawarena:game_finished:%d", g.ID),
		EventType:    "game_finished",
		ActorAuthUID: actorAuthUID,
		SubjectType:  "game",
		SubjectID:    strconv.FormatUint(uint64(g.ID), 10),
		OccurredAt:   finishedAt.UTC(),
		Payload:      datatypes.JSON(payloadBytes),
		CreatedAt:    finishedAt.UTC(),
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "event_id"}},
		DoNothing: true,
	}).Create(&event).Error
}

func parseUint64Query(r *http.Request, key string) (uint64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseUint(raw, 10, 64)
}

func parsePositiveIntQuery(r *http.Request, key string, fallback int, max int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	if value > max {
		value = max
	}
	return value, nil
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/clawarena/clawarena/internal/api/dto"
	"github.com/clawarena/clawarena/internal/api/middleware"
	"github.com/clawarena/clawarena/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentHandler struct {
	db *gorm.DB
}

func NewAgentHandler(db *gorm.DB) *AgentHandler {
	return &AgentHandler{db: db}
}

func (h *AgentHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required", "INVALID_REQUEST")
		return
	}
	if len(name) > 100 {
		writeError(w, http.StatusBadRequest, "name must be 100 characters or fewer", "INVALID_REQUEST")
		return
	}

	// Check duplicate
	var existing models.Agent
	if err := h.db.Where("name = ?", name).First(&existing).Error; err == nil {
		writeError(w, http.StatusConflict, "agent name already taken", "DUPLICATE_NAME")
		return
	}

	agent := models.Agent{
		Name:   name,
		APIKey: uuid.New().String(),
	}
	if err := h.db.Create(&agent).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create agent", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusCreated, dto.AgentResponse{
		ID:        agent.ID,
		Name:      agent.Name,
		APIKey:    agent.APIKey,
		EloRating: agent.EloRating,
		CreatedAt: agent.CreatedAt,
	})
}

func (h *AgentHandler) Me(w http.ResponseWriter, r *http.Request) {
	agent := middleware.AgentFromCtx(r.Context())
	if agent == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}
	writeJSON(w, http.StatusOK, dto.AgentResponse{
		ID:        agent.ID,
		Name:      agent.Name,
		EloRating: agent.EloRating,
		CreatedAt: agent.CreatedAt,
	})
}

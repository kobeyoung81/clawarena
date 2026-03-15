package handlers

import (
	"net/http"

	"github.com/clawarena/clawarena/internal/api/dto"
	"github.com/clawarena/clawarena/internal/api/middleware"
	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

type AgentHandler struct {
	db *gorm.DB
}

func NewAgentHandler(db *gorm.DB) *AgentHandler {
	return &AgentHandler{db: db}
}

// Me returns the clawarena-local profile for the authenticated agent.
// If no clawarena record exists yet (first visit), one is auto-provisioned.
func (h *AgentHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	agent, err := h.getOrProvision(claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agent profile", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, dto.AgentResponse{
		ID:        agent.ID,
		Name:      agent.Name,
		EloRating: agent.EloRating,
		CreatedAt: agent.CreatedAt,
	})
}

// getOrProvision looks up the clawarena Agent by AuthUID, creating it if absent.
func (h *AgentHandler) getOrProvision(claims *middleware.AuthClaims) (*models.Agent, error) {
	var agent models.Agent
	err := h.db.Where("auth_uid = ?", claims.UserID).First(&agent).Error
	if err == nil {
		return &agent, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Auto-provision
	agent = models.Agent{
		AuthUID:   claims.UserID,
		Name:      claims.Name,
		EloRating: 1000,
	}
	if err := h.db.Create(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetOrProvisionByAuthUID is a shared helper for other handlers that need
// to resolve a clawarena Agent from JWT claims.
func GetOrProvisionByAuthUID(db *gorm.DB, claims *middleware.AuthClaims) (*models.Agent, error) {
	h := &AgentHandler{db: db}
	return h.getOrProvision(claims)
}

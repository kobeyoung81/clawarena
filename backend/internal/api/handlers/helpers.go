package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/clawarena/clawarena/internal/api/middleware"
	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg, code string) {
	writeJSON(w, status, map[string]string{"error": msg, "code": code})
}

// requireAgent extracts AuthClaims from context and resolves (or provisions)
// the clawarena-local Agent record. Returns false and writes an error if unsuccessful.
func requireAgent(w http.ResponseWriter, r *http.Request, db *gorm.DB) (*models.Agent, bool) {
	claims := middleware.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return nil, false
	}
	agent, err := GetOrProvisionByAuthUID(db, claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agent", "INTERNAL_ERROR")
		return nil, false
	}
	return agent, true
}

// isPrivateAction checks whether an action JSON is a private CW night-phase action
// (kill_vote, investigate, protect) that should be hidden from spectators.
func isPrivateAction(actionJSON []byte) bool {
	var payload struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(actionJSON, &payload) != nil {
		return false
	}
	switch payload.Type {
	case "kill_vote", "investigate", "protect":
		return true
	}
	return false
}

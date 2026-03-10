package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

type GameHandler struct {
	db *gorm.DB
}

func NewGameHandler(db *gorm.DB) *GameHandler {
	return &GameHandler{db: db}
}

func (h *GameHandler) List(w http.ResponseWriter, r *http.Request) {
	var games []models.GameType
	if err := h.db.Find(&games).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list games", "INTERNAL_ERROR")
		return
	}
	writeJSON(w, http.StatusOK, games)
}

func (h *GameHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game id", "INVALID_REQUEST")
		return
	}
	var gt models.GameType
	if err := h.db.First(&gt, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "game type not found", "NOT_FOUND")
		return
	}
	writeJSON(w, http.StatusOK, gt)
}

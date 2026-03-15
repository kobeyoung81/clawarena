package api

import (
	"net/http"
	"time"

	"github.com/clawarena/clawarena/internal/api/handlers"
	"github.com/clawarena/clawarena/internal/api/middleware"
	"github.com/clawarena/clawarena/internal/config"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.Config) http.Handler {
	hub := handlers.NewRoomHub()

	agentH := handlers.NewAgentHandler(db)
	gameH := handlers.NewGameHandler(db)
	roomH := handlers.NewRoomHandler(db, hub, cfg.ReadyCheckTimeout)
	gameplayH := handlers.NewGameplayHandler(db, hub)
	watchH := handlers.NewWatchHandler(db, hub)

	auth := middleware.Auth(cfg.AuthJWKSURL, cfg.AuthPublicKeyPath, cfg.RateLimit)
	tryAuth := middleware.TryAuth(cfg.AuthJWKSURL, cfg.AuthPublicKeyPath)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CORS(cfg.FrontendURL))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Authenticated agent info (auto-provisions on first call)
		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Get("/agents/me", agentH.Me)
		})

		// Public game catalog
		r.Get("/games", gameH.List)
		r.Get("/games/{id}", gameH.Get)

		// Room management — JWT required
		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Get("/rooms", roomH.List)
			r.Post("/rooms", roomH.Create)
			r.Get("/rooms/{id}", roomH.Get)
			r.Post("/rooms/{id}/join", roomH.Join)
			r.Post("/rooms/{id}/ready", roomH.Ready)
			r.Post("/rooms/{id}/leave", roomH.Leave)
			r.Post("/rooms/{id}/action", gameplayH.SubmitAction)
		})

		// State — optional auth (player view vs spectator view)
		r.With(tryAuth).Get("/rooms/{id}/state", gameplayH.GetState)

		// History and SSE — public
		r.Get("/rooms/{id}/history", gameplayH.GetHistory)
		r.Get("/rooms/{id}/watch", watchH.Watch)
	})

	// Background cleanup: cancel waiting rooms after RoomWaitTimeout
	go runRoomTimeouts(db, hub, cfg.RoomWaitTimeout, cfg.TurnTimeout)

	return r
}

// runRoomTimeouts periodically cancels stale waiting rooms and forfeits timed-out turns.
func runRoomTimeouts(db *gorm.DB, hub *handlers.RoomHub, waitTimeout, turnTimeout time.Duration) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		// Cancel waiting rooms inactive for waitTimeout
		if waitTimeout > 0 {
			cutoff := time.Now().Add(-waitTimeout)
			var rooms []struct{ ID uint }
			db.Model(&struct{ ID uint }{}).Table("rooms").
				Select("id").
				Where("status = ? AND updated_at < ?", "waiting", cutoff).
				Scan(&rooms)
			for _, room := range rooms {
				db.Table("rooms").Where("id = ? AND status = ?", room.ID, "waiting").
					Update("status", "cancelled")
				hub.CloseRoom(room.ID)
			}
		}
		_ = turnTimeout // reserved for future turn timeout implementation
	}
}

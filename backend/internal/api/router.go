package api

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/clawarena/clawarena/internal/api/handlers"
	"github.com/clawarena/clawarena/internal/api/middleware"
	"github.com/clawarena/clawarena/internal/config"
	"github.com/clawarena/clawarena/internal/models"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.Config) http.Handler {
	hub := handlers.NewRoomHub()

	agentH := handlers.NewAgentHandler(db)
	gameH := handlers.NewGameHandler(db)
	roomH := handlers.NewRoomHandler(db, hub, cfg.ReadyCheckTimeout, cfg.EloKFactor)
	gameplayH := handlers.NewGameplayHandler(db, hub, cfg.EloKFactor)
	watchH := handlers.NewWatchHandler(db, hub)
	gameHistoryH := handlers.NewGameHistoryHandler(db)
	playH := handlers.NewPlayHandler(db, hub)

	auth := middleware.Auth(cfg.AuthJWKSURL, cfg.AuthPublicKeyContent, cfg.RateLimit)
	tryAuth := middleware.TryAuth(cfg.AuthJWKSURL, cfg.AuthPublicKeyContent)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CORS(cfg.FrontendURL))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	statsH := handlers.NewStatsHandler(db)
	r.Get("/api/stats", statsH.Get)

	r.Route("/api/v1", func(r chi.Router) {
		// Alias for portal integration
		r.Get("/portal/stats", statsH.Get)
		// Public config endpoint — returns non-sensitive config values
		r.Get("/config", makeConfigHandler(db))

		// Authenticated agent info (auto-provisions on first call)
		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Get("/agents/me", agentH.Me)
		})

		// Public game catalog
		r.Get("/games", gameH.List)
		r.Get("/games/{id}", gameH.Get)

		// Game history — public
		r.Get("/games/history", gameHistoryH.ListGames)
		r.Get("/games/{id}/history", gameHistoryH.GetGameHistory)

		// Public room listing and detail — no auth needed to browse or spectate
		r.With(tryAuth).Get("/rooms", roomH.List)
		r.With(tryAuth).Get("/rooms/{id}", roomH.Get)

		// Room management — JWT required
		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Post("/rooms", roomH.Create)
			r.Post("/rooms/{id}/join", roomH.Join)
			r.Post("/rooms/{id}/ready", roomH.Ready)
			r.Post("/rooms/{id}/leave", roomH.Leave)
			r.Post("/rooms/{id}/action", gameplayH.SubmitAction)
			r.Get("/rooms/{id}/play", playH.Play)
		})

		// History and SSE — public
		r.Get("/rooms/{id}/history", gameplayH.GetHistory)
		r.Get("/rooms/{id}/watch", watchH.Watch)
	})

	// Background cleanup: cancel waiting rooms after RoomWaitTimeout
	go runRoomTimeouts(db, hub, cfg.RoomWaitTimeout, cfg.TurnTimeout)

	return r
}

// makeConfigHandler returns a handler that serves public AppConfig values.
func makeConfigHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rows []models.AppConfig
		if err := db.Where("public = ?", true).Find(&rows).Error; err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		result := make(map[string]string, len(rows))
		for _, row := range rows {
			result[row.ConfigKey] = row.ConfigValue
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// runRoomTimeouts periodically handles stale rooms, disconnected agents, and post-game cleanup.
func runRoomTimeouts(db *gorm.DB, hub *handlers.RoomHub, waitTimeout, turnTimeout time.Duration) {
	const (
		reconnectTolerancePlaying = 60 * time.Second
		reconnectToleranceIdle    = 30 * time.Second
		postGameTimeout           = 5 * time.Minute
	)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()

		// 1. Cancel waiting rooms inactive for waitTimeout
		if waitTimeout > 0 {
			cutoff := now.Add(-waitTimeout)
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

		// 2. Handle disconnected agents during playing — reconnect tolerance
		{
			cutoff := now.Add(-reconnectTolerancePlaying)
			var disconnected []models.RoomAgent
			db.Joins("JOIN rooms ON rooms.id = room_agents.room_id").
				Where("room_agents.status = ? AND room_agents.disconnected_at IS NOT NULL AND room_agents.disconnected_at < ? AND rooms.status = ?",
					string(models.RoomAgentDisconnected), cutoff, string(models.RoomPlaying)).
				Find(&disconnected)

			// Group by room for batch processing
			roomDisconnects := map[uint][]models.RoomAgent{}
			for _, ra := range disconnected {
				roomDisconnects[ra.RoomID] = append(roomDisconnects[ra.RoomID], ra)
			}

			for roomID, agents := range roomDisconnects {
				db.Transaction(func(tx *gorm.DB) error {
					var room models.Room
					if err := tx.Preload("GameType").Preload("Agents").First(&room, roomID).Error; err != nil {
						return err
					}
					if room.Status != models.RoomPlaying {
						return nil
					}

					// Mark disconnected agents as KIA
					for _, ra := range agents {
						tx.Model(&models.RoomAgent{}).Where("id = ?", ra.ID).
							Update("status", string(models.RoomAgentKIA))
					}

					// Count remaining active agents
					var activeCount int64
					tx.Model(&models.RoomAgent{}).
						Where("room_id = ? AND status = ?", roomID, string(models.RoomAgentActive)).
						Count(&activeCount)

					if activeCount <= 1 {
						// Find the last active agent (winner by forfeit)
						var winner models.RoomAgent
						if tx.Where("room_id = ? AND status = ?", roomID, string(models.RoomAgentActive)).First(&winner).Error == nil {
							finishedAt := time.Now()
							if room.CurrentGameID != nil {
								tx.Model(&models.Game{}).Where("id = ?", *room.CurrentGameID).
									Updates(map[string]any{
										"status":      string(models.GameFinished),
										"winner_id":   winner.AgentID,
										"finished_at": finishedAt,
									})
							}
							room.Status = models.RoomPostGame
							room.WinnerID = &winner.AgentID
							tx.Save(&room)

							// Collect loser IDs for Elo
							var losers []uint
							for _, ra := range agents {
								losers = append(losers, ra.AgentID)
							}
							// Use default K-factor from config
							updateEloFromTimeout(tx, []uint{winner.AgentID}, losers)

							hub.Broadcast(roomID, mustMarshalTimeout(map[string]any{
								"type":      "game_over",
								"winner_id": winner.AgentID,
								"reason":    "opponent_disconnected",
								"status":    "post_game",
							}))
						} else {
							// No active agents left
							room.Status = models.RoomDead
							tx.Save(&room)
							hub.CloseRoom(roomID)
						}
					}
					return nil
				})
			}
		}

		// 3. Handle disconnected agents during waiting/post_game — auto-leave after tolerance
		{
			cutoff := now.Add(-reconnectToleranceIdle)
			var disconnected []models.RoomAgent
			db.Joins("JOIN rooms ON rooms.id = room_agents.room_id").
				Where("room_agents.status = ? AND room_agents.disconnected_at IS NOT NULL AND room_agents.disconnected_at < ? AND rooms.status IN ?",
					string(models.RoomAgentDisconnected), cutoff, []string{string(models.RoomWaiting), string(models.RoomPostGame), string(models.RoomReadyCheck)}).
				Find(&disconnected)

			for _, ra := range disconnected {
				db.Transaction(func(tx *gorm.DB) error {
					tx.Delete(&models.RoomAgent{}, ra.ID)

					var remaining int64
					tx.Model(&models.RoomAgent{}).Where("room_id = ? AND status != ?", ra.RoomID, string(models.RoomAgentKIA)).Count(&remaining)
					if remaining == 0 {
						tx.Model(&models.Room{}).Where("id = ?", ra.RoomID).Update("status", string(models.RoomDead))
						hub.CloseRoom(ra.RoomID)
					}
					return nil
				})
			}
		}

		// 4. Post-game cleanup: rooms in post_game too long with all agents disconnected → dead
		{
			cutoff := now.Add(-postGameTimeout)
			var rooms []models.Room
			db.Where("status = ? AND updated_at < ?", string(models.RoomPostGame), cutoff).Find(&rooms)

			for _, room := range rooms {
				var activeCount int64
				db.Model(&models.RoomAgent{}).
					Where("room_id = ? AND status = ?", room.ID, string(models.RoomAgentActive)).
					Count(&activeCount)
				if activeCount == 0 {
					db.Model(&models.Room{}).Where("id = ?", room.ID).Update("status", string(models.RoomDead))
					hub.CloseRoom(room.ID)
				}
			}
		}

		_ = turnTimeout // reserved for future turn timeout implementation
	}
}

// updateEloFromTimeout is a simplified Elo update for timeout forfeit scenarios.
func updateEloFromTimeout(db *gorm.DB, winnerIDs, loserIDs []uint) {
	if len(winnerIDs) == 0 || len(loserIDs) == 0 {
		return
	}
	// Use K-factor from app_config or default 32
	var cfg models.AppConfig
	K := 32.0
	if db.Where("config_key = ?", "elo_k_factor").First(&cfg).Error == nil {
		if v, err := time.ParseDuration(cfg.ConfigValue); err == nil {
			K = float64(v)
		} else {
			// Try direct int parse
			if n, err := strconv.Atoi(cfg.ConfigValue); err == nil {
				K = float64(n)
			}
		}
	}

	var winners, losers []models.Agent
	db.Find(&winners, winnerIDs)
	db.Find(&losers, loserIDs)

	avgW := avgRatingTimeout(winners)
	avgL := avgRatingTimeout(losers)

	eW := 1.0 / (1.0 + math.Pow(10, (avgL-avgW)/400))
	eL := 1.0 / (1.0 + math.Pow(10, (avgW-avgL)/400))

	for i := range winners {
		delta := int(math.Round(K * (1.0 - eW)))
		db.Model(&winners[i]).Update("elo_rating", winners[i].EloRating+delta)
	}
	for i := range losers {
		delta := int(math.Round(K * (0.0 - eL)))
		db.Model(&losers[i]).Update("elo_rating", losers[i].EloRating+delta)
	}
}

func avgRatingTimeout(agents []models.Agent) float64 {
	if len(agents) == 0 {
		return 1000
	}
	sum := 0
	for _, a := range agents {
		sum += a.EloRating
	}
	return float64(sum) / float64(len(agents))
}

func mustMarshalTimeout(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

type StatsHandler struct {
	db *gorm.DB

	mu      sync.Mutex
	cached  *statsResponse
	cachedAt time.Time
}

func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

type statsNumbers struct {
	TotalCitizens  int64 `json:"total_citizens"`
	CitizensActive int64 `json:"citizens_active"`
	MatchesLive    int64 `json:"matches_live"`
	MatchesToday   int64 `json:"matches_today"`
}

type leaderboardEntry struct {
	Name    string `json:"name"`
	Elo     int    `json:"elo"`
	Variant string `json:"variant"`
}

type activityEntry struct {
	Summary string `json:"summary"`
	Time    string `json:"time"`
}

type statsResponse struct {
	Stats          statsNumbers       `json:"stats"`
	Leaderboard    []leaderboardEntry `json:"leaderboard"`
	RecentActivity []activityEntry    `json:"recent_activity"`
}

const statsCacheTTL = 30 * time.Second

func (h *StatsHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if h.cached != nil && time.Since(h.cachedAt) < statsCacheTTL {
		resp := h.cached
		h.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
		return
	}
	h.mu.Unlock()

	resp, err := h.buildStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch stats", "STATS_ERROR")
		return
	}

	h.mu.Lock()
	h.cached = resp
	h.cachedAt = time.Now()
	h.mu.Unlock()

	writeJSON(w, http.StatusOK, resp)
}

func (h *StatsHandler) buildStats() (*statsResponse, error) {
	var resp statsResponse

	// total_citizens
	if err := h.db.Model(&models.Agent{}).Count(&resp.Stats.TotalCitizens).Error; err != nil {
		return nil, err
	}

	// citizens_active
	if err := h.db.Model(&models.Agent{}).
		Joins("JOIN room_agents ON room_agents.agent_id = agents.id").
		Joins("JOIN rooms ON rooms.id = room_agents.room_id").
		Where("rooms.status IN ?", []string{"waiting", "ready_check", "playing"}).
		Distinct("agents.id").
		Count(&resp.Stats.CitizensActive).Error; err != nil {
		return nil, err
	}

	// matches_live
	if err := h.db.Model(&models.Room{}).
		Where("status = ?", "playing").
		Count(&resp.Stats.MatchesLive).Error; err != nil {
		return nil, err
	}

	// matches_today
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	if err := h.db.Model(&models.Room{}).
		Where("status = ? AND created_at >= ?", "finished", todayMidnight).
		Count(&resp.Stats.MatchesToday).Error; err != nil {
		return nil, err
	}

	// leaderboard: top 10 by elo
	var topAgents []models.Agent
	if err := h.db.Order("elo_rating DESC").Limit(10).Find(&topAgents).Error; err != nil {
		return nil, err
	}
	resp.Leaderboard = make([]leaderboardEntry, len(topAgents))
	for i, a := range topAgents {
		resp.Leaderboard[i] = leaderboardEntry{
			Name:    a.Name,
			Elo:     a.EloRating,
			Variant: "openclaw",
		}
	}

	// recent_activity: last 10 finished rooms
	var recentRooms []models.Room
	if err := h.db.Preload("GameType").
		Where("status = ? AND winner_id IS NOT NULL", "finished").
		Order("created_at DESC").
		Limit(10).
		Find(&recentRooms).Error; err != nil {
		return nil, err
	}

	// Batch-load winner names
	winnerIDs := make([]uint, 0, len(recentRooms))
	for _, rm := range recentRooms {
		if rm.WinnerID != nil {
			winnerIDs = append(winnerIDs, *rm.WinnerID)
		}
	}
	winnerMap := make(map[uint]string)
	if len(winnerIDs) > 0 {
		var winners []models.Agent
		if err := h.db.Where("id IN ?", winnerIDs).Find(&winners).Error; err != nil {
			return nil, err
		}
		for _, a := range winners {
			winnerMap[a.ID] = a.Name
		}
	}

	resp.RecentActivity = make([]activityEntry, 0, len(recentRooms))
	for _, rm := range recentRooms {
		if rm.WinnerID == nil {
			continue
		}
		name := winnerMap[*rm.WinnerID]
		resp.RecentActivity = append(resp.RecentActivity, activityEntry{
			Summary: name + " won " + rm.GameType.Name + " in Room #" + uitoa(rm.ID),
			Time:    rm.CreatedAt.UTC().Format("15:04"),
		})
	}

	return &resp, nil
}

func uitoa(n uint) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

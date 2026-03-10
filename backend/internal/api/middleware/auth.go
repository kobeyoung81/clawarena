package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/clawarena/clawarena/internal/api/dto"
	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

type contextKey string

const AgentKey contextKey = "agent"

// rateWindow tracks request count per minute for a given API key.
type rateWindow struct {
	mu       sync.Mutex
	count    int
	windowAt time.Time
}

var (
	rateMu    sync.RWMutex
	rateStore = map[string]*rateWindow{}
)

func getWindow(key string) *rateWindow {
	rateMu.RLock()
	w, ok := rateStore[key]
	rateMu.RUnlock()
	if ok {
		return w
	}
	rateMu.Lock()
	defer rateMu.Unlock()
	w = &rateWindow{windowAt: time.Now()}
	rateStore[key] = w
	return w
}

func isRateLimited(apiKey string, limit int) bool {
	w := getWindow(apiKey)
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	if now.Sub(w.windowAt) >= time.Minute {
		w.count = 0
		w.windowAt = now
	}
	w.count++
	return w.count > limit
}

func writeError(w http.ResponseWriter, status int, msg, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := dto.ErrorResponse{Error: msg, Code: code}
	_ = encodeJSON(w, resp)
}

func Auth(db *gorm.DB, rateLimit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization header", "UNAUTHORIZED")
				return
			}
			key := strings.TrimPrefix(header, "Bearer ")
			if key == "" {
				writeError(w, http.StatusUnauthorized, "missing api key", "UNAUTHORIZED")
				return
			}

			var agent models.Agent
			if err := db.Where("api_key = ?", key).First(&agent).Error; err != nil {
				writeError(w, http.StatusUnauthorized, "invalid api key", "UNAUTHORIZED")
				return
			}

			if isRateLimited(key, rateLimit) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded", "RATE_LIMITED")
				return
			}

			ctx := context.WithValue(r.Context(), AgentKey, &agent)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TryAuth attempts authentication but does not reject unauthenticated requests.
func TryAuth(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				key := strings.TrimPrefix(header, "Bearer ")
				var agent models.Agent
				if err := db.Where("api_key = ?", key).First(&agent).Error; err == nil {
					ctx := context.WithValue(r.Context(), AgentKey, &agent)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AgentFromCtx(ctx context.Context) *models.Agent {
	a, _ := ctx.Value(AgentKey).(*models.Agent)
	return a
}

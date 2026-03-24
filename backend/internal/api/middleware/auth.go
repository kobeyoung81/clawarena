package middleware

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/clawarena/clawarena/internal/api/dto"
)

type contextKey string

const AgentKey contextKey = "auth_claims"

// AuthClaims holds the lightweight identity extracted from a JWT.
type AuthClaims struct {
	UserID string // JWT sub (auth service user ID, e.g. "usr_...")
	Type   string // "human" | "agent"
	Name   string
}

// jwkSet is the minimal JWKS structure we parse.
type jwkSet struct {
	Keys []struct {
		N   string `json:"n"`
		E   string `json:"e"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	} `json:"keys"`
}

// jwtClaims is the minimal JWT claims we care about.
type jwtClaims struct {
	Sub  string `json:"sub"`
	Type string `json:"type"`
	Name string `json:"name"`
	Exp  int64  `json:"exp"`
}

// jwksCache fetches and caches the public key from the JWKS endpoint.
type jwksCache struct {
	mu         sync.RWMutex
	publicKey  *rsa.PublicKey
	fetchedAt  time.Time
	url        string
	keyContent string // optional PEM content stored directly in the database
}

func newJWKSCache(jwksURL, keyContent string) *jwksCache {
	return &jwksCache{url: jwksURL, keyContent: keyContent}
}

func (c *jwksCache) getKey() (*rsa.PublicKey, error) {
	c.mu.RLock()
	key := c.publicKey
	age := time.Since(c.fetchedAt)
	c.mu.RUnlock()

	if key != nil && age < time.Hour {
		return key, nil
	}

	return c.refresh()
}

func (c *jwksCache) refresh() (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Inline PEM content takes precedence (dev/testing)
	if c.keyContent != "" {
		key, err := parsePEMPublicKey([]byte(c.keyContent))
		if err == nil {
			c.publicKey = key
			c.fetchedAt = time.Now()
			return key, nil
		}
		log.Printf("[auth] warn: failed to parse inline public key: %v", err)
	}

	// Fetch JWKS from URL
	if c.url == "" {
		return nil, fmt.Errorf("no JWKS URL or public key content configured")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(c.url)
	if err != nil {
		if c.publicKey != nil {
			return c.publicKey, nil // use stale key if fetch fails
		}
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	for _, k := range jwks.Keys {
		if k.Alg != "RS256" {
			continue
		}
		pubKey, err := jwkToRSA(k.N, k.E)
		if err != nil {
			continue
		}
		c.publicKey = pubKey
		c.fetchedAt = time.Now()
		return pubKey, nil
	}

	return nil, fmt.Errorf("no RS256 key found in JWKS")
}

func jwkToRSA(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

func parsePEMPublicKey(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}
	return rsaKey, nil
}

func verifyJWT(publicKey *rsa.PublicKey, tokenString string) (*jwtClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	signingInput := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(signingInput))

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid signature")
	}

	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], sigBytes); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid claims")
	}

	var claims jwtClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims JSON")
	}

	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// rateWindow tracks request count per minute for a given user.
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

func isRateLimited(userID string, limit int) bool {
	w := getWindow(userID)
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
	b, _ := json.Marshal(resp)
	w.Write(b)
}

// Auth validates a Bearer JWT and rejects unauthenticated requests.
func Auth(jwksURL, keyContent string, rateLimit int) func(http.Handler) http.Handler {
	cache := newJWKSCache(jwksURL, keyContent)
	// Pre-warm the cache
	go func() {
		if _, err := cache.refresh(); err != nil {
			log.Printf("[auth] warn: could not pre-fetch public key: %v", err)
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization header", "UNAUTHORIZED")
				return
			}

			pubKey, err := cache.getKey()
			if err != nil {
				writeError(w, http.StatusServiceUnavailable, "auth service unavailable", "AUTH_UNAVAILABLE")
				return
			}

			claims, err := verifyJWT(pubKey, token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token", "UNAUTHORIZED")
				return
			}

			if isRateLimited(claims.Sub, rateLimit) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded", "RATE_LIMITED")
				return
			}

			ac := &AuthClaims{UserID: claims.Sub, Type: claims.Type, Name: claims.Name}
			ctx := context.WithValue(r.Context(), AgentKey, ac)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TryAuth attempts JWT auth but does not reject unauthenticated requests.
func TryAuth(jwksURL, keyContent string) func(http.Handler) http.Handler {
	cache := newJWKSCache(jwksURL, keyContent)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token != "" {
				if pubKey, err := cache.getKey(); err == nil {
					if claims, err := verifyJWT(pubKey, token); err == nil {
						ac := &AuthClaims{UserID: claims.Sub, Type: claims.Type, Name: claims.Name}
						ctx := context.WithValue(r.Context(), AgentKey, ac)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromCtx retrieves auth claims from the request context.
func ClaimsFromCtx(ctx context.Context) *AuthClaims {
	c, _ := ctx.Value(AgentKey).(*AuthClaims)
	return c
}

// AgentFromCtx is an alias for ClaimsFromCtx for backward compatibility.
func AgentFromCtx(ctx context.Context) *AuthClaims {
	return ClaimsFromCtx(ctx)
}

func extractBearer(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	if cookie, err := r.Cookie("lc_access"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

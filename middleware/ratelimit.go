// middleware/ratelimit.go
package middleware

import (
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Token bucket rate limiter implementation
type TokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
	mu             sync.Mutex
}

func NewTokenBucket(maxTokens, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:         maxTokens,
		maxTokens:      maxTokens,
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefillTime = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Rate limiter storage
type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex

	// Configuration
	maxRequests   int
	windowSeconds int
}

var (
	generalLimiter *RateLimiter
	authLimiter    *RateLimiter
)

func init() {
	// Initialize rate limiters from environment
	generalMaxReq := getEnvInt("RATE_LIMIT_MAX_REQUESTS", 100)          // tokens
	generalWindow := getEnvInt("RATE_LIMIT_WINDOW_MS", 900000) / 1000   // 15 min default
	if generalWindow <= 0 {
		generalWindow = 900 // guard
	}
	authMaxReq := getEnvInt("AUTH_RATE_LIMIT_MAX", 5)
	authWindow := getEnvInt("AUTH_RATE_LIMIT_WINDOW_MS", 300000) / 1000 // 5 min default
	if authWindow <= 0 {
		authWindow = 300
	}

	generalLimiter = NewRateLimiter(generalMaxReq, generalWindow)
	authLimiter = NewRateLimiter(authMaxReq, authWindow)

	// Cleanup old buckets every 10 minutes
	go startCleanupRoutine()
}

func NewRateLimiter(maxRequests, windowSeconds int) *RateLimiter {
	return &RateLimiter{
		buckets:       make(map[string]*TokenBucket),
		maxRequests:   maxRequests,
		windowSeconds: windowSeconds,
	}
}

func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		refillRate := float64(rl.maxRequests) / float64(rl.windowSeconds) // tokens/sec
		bucket = NewTokenBucket(float64(rl.maxRequests), refillRate)
		rl.buckets[key] = bucket
	}
	return bucket
}

func (rl *RateLimiter) Allow(key string) bool {
	bucket := rl.getBucket(key)
	return bucket.Allow()
}

// Cleanup old buckets periodically
func startCleanupRoutine() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cleanupOldBuckets(generalLimiter)
		cleanupOldBuckets(authLimiter)
	}
}

func cleanupOldBuckets(rl *RateLimiter) {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		// Remove buckets that haven't been accessed in 30 minutes
		if now.Sub(bucket.lastRefillTime) > 30*time.Minute {
			delete(rl.buckets, key)
		}
		bucket.mu.Unlock()
	}
}

// Helper functions

func getClientIPFromStd(r *http.Request) string {
	// X-Forwarded-For may contain a list, take the first
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return rip
	}
	// RemoteAddr can be "ip:port"
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func getEnvInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return def
}

func rateLimitDisabled() bool {
	// RATE_LIMIT_ENABLED=false disables limiter
	val := strings.ToLower(strings.TrimSpace(os.Getenv("RATE_LIMIT_ENABLED")))
	return val == "false" || val == "0" || val == "no"
}

// Middleware (net/http)

// RateLimitMiddleware applies general rate limiting (net/http)
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rateLimitDisabled() {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := getClientIPFromStd(r)
		if !generalLimiter.Allow(clientIP) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"success": false, "error": "Rate limit exceeded. Please try again later."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AuthRateLimitMiddleware applies stricter rate limiting for auth endpoints (net/http)
func AuthRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rateLimitDisabled() {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := getClientIPFromStd(r)
		if !authLimiter.Allow(clientIP) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"success": false, "message": "Too many authentication attempts. Please try again in 5 minutes."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Middleware (Fiber)

// FiberRateLimitMiddleware applies general rate limiting for Fiber
func FiberRateLimitMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if rateLimitDisabled() {
			return c.Next()
		}
		// Skip static and health endpoints to reduce dev friction
		path := c.Path()
		if path == "/health" || path == "/api/health" ||
			strings.HasPrefix(path, "/static") ||
			strings.HasPrefix(path, "/css") ||
			strings.HasPrefix(path, "/js") {
			return c.Next()
		}

		clientIP := c.IP()
		if !generalLimiter.Allow(clientIP) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "Rate limit exceeded. Please try again later.",
			})
		}
		return c.Next()
	}
}

// FiberAuthRateLimitMiddleware applies stricter rate limiting to auth endpoints for Fiber
func FiberAuthRateLimitMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if rateLimitDisabled() {
			return c.Next()
		}
		clientIP := c.IP()
		if !authLimiter.Allow(clientIP) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "Too many authentication attempts. Please try again in 5 minutes.",
			})
		}
		return c.Next()
	}
}

// RateLimitConfig for HTTPRateLimit wrapper
type RateLimitConfig struct {
	Requests  int
	Burst     int
	Window    time.Duration
	KeyFunc   func(*http.Request) string
	BlockCode int
}

// HTTPRateLimit is a configurable rate limiter wrapper
func HTTPRateLimit(config RateLimitConfig) func(http.Handler) http.Handler {
	// Create a custom rate limiter with the specified config
	// Use Requests as max tokens, and convert Window to seconds
	windowSeconds := int(config.Window.Seconds())
	if windowSeconds <= 0 {
		windowSeconds = 60 // default to 1 minute
	}
	limiter := NewRateLimiter(config.Requests, windowSeconds)

	keyFunc := config.KeyFunc
	if keyFunc == nil {
		keyFunc = func(r *http.Request) string {
			return getClientIPFromStd(r)
		}
	}

	blockCode := config.BlockCode
	if blockCode == 0 {
		blockCode = http.StatusTooManyRequests
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rateLimitDisabled() {
				next.ServeHTTP(w, r)
				return
			}

			key := keyFunc(r)
			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(blockCode)
				_, _ = w.Write([]byte(`{"success": false, "error": "Rate limit exceeded. Please try again later."}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPKeyFunc returns the client IP as the rate limit key
func IPKeyFunc(r *http.Request) string {
	return getClientIPFromStd(r)
}

// HTTPCORSMiddleware adds CORS headers for specified origins
func HTTPCORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else if len(allowedOrigins) > 0 {
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigins[0])
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HTTPRecoverMiddleware recovers from panics and returns a 500 error
func HTTPRecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v\n", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"success": false, "error": "Internal server error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
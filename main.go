// ~/Documents/CODING/ubible/main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"ubible/database"
	"ubible/handlers"
	"ubible/middleware"
	"ubible/services"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Validate critical environment variables
	validateEnvironment()

	// Initialize database
	database.InitDB()

	// Initialize team handlers
	// handlers.InitTeamHandlers() // TODO: Uncomment after migrating teams.go

	// Load verses from files
	log.Println("Loading verses from files...")
	services.LoadVersesFromFiles()
	services.LoadVersesFromTXT()

	// Initialize cleanup service
	services.InitCleanupService()
	defer func() {
		if cleanupService := services.GetCleanupService(); cleanupService != nil {
			cleanupService.Stop()
		}
	}()

	// HTTP Mux for REST API and HTML/static
	mux := http.NewServeMux()

	// Static files
	// Directories
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./static/css"))))
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./static/js"))))
	mux.Handle("/admin/", http.StripPrefix("/admin/", http.FileServer(http.Dir("./static/admin"))))
	mux.Handle("/verses/", http.StripPrefix("/verses/", http.FileServer(http.Dir("./verses"))))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// HTML routes
	mux.HandleFunc("/", serveFile("./static/index.html"))
	mux.HandleFunc("/quiz", serveFile("./static/quiz.html"))
	mux.HandleFunc("/quiz.html", serveFile("./static/quiz.html"))
	mux.HandleFunc("/practice", serveFile("./static/practice.html"))
	mux.HandleFunc("/practice.html", serveFile("./static/practice.html"))
	mux.HandleFunc("/challenges", serveFile("./static/challenges.html"))
	mux.HandleFunc("/challenges.html", serveFile("./static/challenges.html"))
	mux.HandleFunc("/settings", serveFile("./static/settings.html"))
	mux.HandleFunc("/settings.html", serveFile("./static/settings.html"))
	mux.HandleFunc("/shop", serveFile("./static/shop.html"))
	mux.HandleFunc("/shop.html", serveFile("./static/shop.html"))
	mux.HandleFunc("/login", serveFile("./static/login.html"))
	mux.HandleFunc("/login.html", serveFile("./static/login.html"))
	mux.HandleFunc("/teams", serveFile("./static/teams.html"))
	mux.HandleFunc("/teams.html", serveFile("./static/teams.html"))
	mux.HandleFunc("/ai-theme-maker", serveFile("./static/ai-theme-maker.html"))
	mux.HandleFunc("/ai-theme-maker.html", serveFile("./static/ai-theme-maker.html"))
	mux.HandleFunc("/theme-maker", serveFile("./static/theme-maker.html"))
	mux.HandleFunc("/theme-maker.html", serveFile("./static/theme-maker.html"))

	// Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// API routes (only wiring endpoints already using net/http signatures)
	// Global API prefix helper
	route := func(path string, h http.Handler) {
		mux.Handle(path, h)
	}

	// Rate limits
	globalRL := middleware.HTTPRateLimit(middleware.RateLimitConfig{
		Requests:  300,
		Burst:     60,
		Window:    time.Minute,
		KeyFunc:   middleware.IPKeyFunc,
		BlockCode: http.StatusTooManyRequests,
	})
	authRL := middleware.HTTPRateLimit(middleware.RateLimitConfig{
		Requests:  30,
		Burst:     10,
		Window:    time.Minute,
		KeyFunc:   middleware.IPKeyFunc,
		BlockCode: http.StatusTooManyRequests,
	})

	// CORS
	corsOrigins := getEnv("CORS_ORIGINS", "http://localhost:3000")
	allowed := splitAndTrim(corsOrigins)

	// Helpers to build chains
	chain := func(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
		// Apply in reverse (outermost first in list)
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
	mh := func(method string, fn http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				// Allow preflight through CORS middleware
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.Method != method {
				http.NotFound(w, r)
				return
			}
			fn.ServeHTTP(w, r)
		})
	}

	// Auth routes
	route("/api/auth/guest", chain(
		mh(http.MethodPost, handlers.GuestLogin),
		authRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/auth/login", chain(
		mh(http.MethodPost, handlers.Login),
		authRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/auth/register", chain(
		mh(http.MethodPost, handlers.Register),
		authRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/auth/upgrade", chain(
		middleware.AuthMiddleware(mh(http.MethodPost, handlers.UpgradeGuest)),
		authRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/auth/preferences", chain(
		middleware.AuthMiddleware(mh(http.MethodGet, handlers.GetPreferences)),
		authRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/auth/preferences/save", chain( // if original was POST /preferences reuse same path
		middleware.AuthMiddleware(mh(http.MethodPost, handlers.SavePreferences)),
		authRL,
		middleware.HTTPCORSMiddleware(allowed),
	))

	// Themes (migrated to net/http)
	route("/api/themes", chain(
		mh(http.MethodGet, handlers.GetThemes),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/themes/public", chain(
		mh(http.MethodPost, handlers.CreatePublicTheme),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/themes/generate", chain(
		mh(http.MethodPost, handlers.GenerateTheme),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	// TODO: /api/themes/:id (requires param parsing in handler)

	// Verses (migrated to net/http)
	route("/api/verses", chain(
		mh(http.MethodGet, handlers.GetVerses),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	// TODO: /api/verses/:id

	// Quiz (migrated to net/http)
	route("/api/questions/quiz", chain(
		mh(http.MethodGet, handlers.GetQuizQuestions),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))

	// Practice
	route("/api/practice/cards", chain(
		mh(http.MethodGet, handlers.GetPracticeCards),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))

	// Stats
	route("/api/stats/players", chain(
		mh(http.MethodGet, handlers.GetOnlinePlayersCount),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/stats/last-played", chain(
		mh(http.MethodGet, handlers.GetLastPlayedTime),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))

	// Debug
	route("/api/debug/rooms", chain(
		mh(http.MethodGet, handlers.GetActiveRooms),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	route("/api/debug/sessions", chain(
		mh(http.MethodGet, handlers.GetGameSessions),
		globalRL,
		middleware.HTTPCORSMiddleware(allowed),
	))
	// TODO: /api/debug/rooms/:code

	// Wrap mux with global middlewares
	rootHandler := chain(
		mux,
		globalRL,
		middleware.HTTPLoggerMiddleware,
		middleware.HTTPRecoverMiddleware,
		middleware.HTTPCORSMiddleware(allowed), // also handle OPTIONS at root
	)

	// Start WebSocket server on port 4000 (pure net/http)
	wsPort := getEnv("WS_PORT", "4000")
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/ws", handlers.WebSocketHandler)
	wsServer := &http.Server{
		Addr:    ":" + wsPort,
		Handler: wsMux,
	}
	go func() {
		log.Printf("🌐 WebSocket server starting on port %s", wsPort)
		if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("WebSocket server failed:", err)
		}
	}()

	// Start HTTP/REST server on port 3000
	port := getEnv("PORT", "3000")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      rootHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("🚀 HTTP server starting on port %s", port)
	log.Printf("📊 Environment: %s", getEnv("APP_ENV", "development"))
	log.Printf("🔐 JWT Secret configured: %v", os.Getenv("JWT_SECRET") != "")
	log.Printf("🧹 Guest cleanup: %s", getEnv("GUEST_CLEANUP_ENABLED", "true"))
	log.Printf("🌐 WebSocket available at ws://localhost:%s/ws", wsPort)
	log.Printf("✅ Quiz endpoint available at /api/questions/quiz")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Failed to start HTTP server:", err)
	}
}

// validateEnvironment checks for required environment variables
func validateEnvironment() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable must be set. Generate one with: openssl rand -base64 64")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("FATAL: JWT_SECRET must be at least 32 characters long")
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "production" {
		// Additional production checks
		corsOrigins := os.Getenv("CORS_ORIGINS")
		if corsOrigins == "" || corsOrigins == "http://localhost:3000" {
			log.Println("WARNING: CORS_ORIGINS not properly configured for production")
		}
	}
}

// Helpers

func serveFile(filepath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = jsonEncode(w, v)
}

func jsonEncode(w http.ResponseWriter, v any) error {
	// small local encoder to avoid importing encoding/json everywhere in this file
	type jsonMarshaler interface {
		MarshalJSON() ([]byte, error)
	}
	// fallback to standard library
	return (&jsonEncoder{w: w}).Encode(v)
}

type jsonEncoder struct {
	w http.ResponseWriter
}

func (e *jsonEncoder) Encode(v any) error {
	// defer to stdlib
	return json.NewEncoder(e.w).Encode(v)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
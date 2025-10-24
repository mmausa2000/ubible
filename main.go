// ~/Documents/CODING/ubible/main.go
package main

import (
	"log"
	"net/http"
	"os"
	"time"
	"ubible/database"
	"ubible/handlers"
	"ubible/handlers/admin"
	"ubible/middleware"
	"ubible/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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
	handlers.InitTeamHandlers()

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

	// Initialize matchmaking service
	// services.InitMatchmaking()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
		BodyLimit:    4 * 1024 * 1024, // 4MB
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))

	// CORS configuration
	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// Apply rate limiting to all routes
	app.Use(middleware.FiberRateLimitMiddleware())

	// Serve static files
	app.Static("/", "./static")
	app.Static("/css", "./static/css")
	app.Static("/js", "./static/js")
	app.Static("/admin", "./static/admin")
	app.Static("/verses", "./verses")

	// API Routes
	api := app.Group("/api")

	// Auth routes with stricter rate limiting
	authGroup := api.Group("/auth")
	authGroup.Use(middleware.FiberAuthRateLimitMiddleware())
	authGroup.Post("/guest", handlers.GuestLogin)
	authGroup.Post("/login", handlers.Login)
	authGroup.Post("/register", handlers.Register)
	authGroup.Post("/upgrade", middleware.AuthMiddleware, handlers.UpgradeGuest)
	authGroup.Get("/preferences", middleware.AuthMiddleware, handlers.GetPreferences)
	authGroup.Post("/preferences", middleware.AuthMiddleware, handlers.SavePreferences)

	// Theme routes
	api.Get("/themes", handlers.GetThemes)
	api.Post("/themes", handlers.CreatePublicTheme) // Public theme creation (no auth required)
	api.Post("/themes/generate", handlers.GenerateTheme) // Generate theme from keywords (no auth required)
	api.Get("/themes/:id", handlers.GetTheme)
	api.Put("/themes/:id", middleware.AuthMiddleware, handlers.UpdateTheme)
	api.Delete("/themes/:id", middleware.AuthMiddleware, handlers.DeleteTheme)

	// Verse routes
	api.Get("/verses", handlers.GetVerses)
	api.Get("/verses/:id", handlers.GetVerse)

	// Quiz questions endpoint
	api.Get("/questions/quiz", handlers.GetQuizQuestions)

	// Practice routes
	api.Get("/practice/cards", handlers.GetPracticeCards)

	// Stats routes
	api.Get("/stats/players", handlers.GetOnlinePlayersCount)
	api.Get("/stats/last-played", handlers.GetLastPlayedTime)

	// Game session routes
	api.Get("/game/check-active", handlers.CheckActiveGame)
	api.Post("/game/start", handlers.StartGameSession)
	api.Post("/game/end", handlers.EndGameSession)

	// User routes (require authentication)
	userGroup := api.Group("/users")
	userGroup.Use(middleware.AuthMiddleware)
	userGroup.Get("/me", handlers.GetCurrentUser)
	userGroup.Put("/me", handlers.UpdateCurrentUser)
	userGroup.Get("/stats", handlers.GetUserStats)
	userGroup.Get("/search", handlers.SearchUsers)
	userGroup.Get("/:id", handlers.GetUserProfile)

	// Power-up routes
	powerupGroup := api.Group("/powerups")
	powerupGroup.Use(middleware.AuthMiddleware)
	powerupGroup.Post("/use", handlers.UsePowerUp)
	powerupGroup.Post("/purchase", handlers.PurchasePowerUp)
	powerupGroup.Get("/inventory", handlers.GetPowerUpInventory)

	// Progression routes
	progressionGroup := api.Group("/progression")
	progressionGroup.Use(middleware.AuthMiddleware)
	progressionGroup.Post("/xp", handlers.AwardXP)
	progressionGroup.Post("/game", handlers.RecordGame)
	progressionGroup.Get("/", handlers.GetProgression)
	progressionGroup.Get("/achievements", handlers.GetUserAchievements)

	// Game routes
	gameGroup := api.Group("/games")
	gameGroup.Use(middleware.AuthMiddleware)
	gameGroup.Post("/record", handlers.RecordGame)
	gameGroup.Get("/history", handlers.GetGameHistory)

	// Friend routes
	friendGroup := api.Group("/friends")
	friendGroup.Use(middleware.AuthMiddleware)
	friendGroup.Get("/", handlers.GetFriends)
	friendGroup.Post("/request", handlers.SendFriendRequest)
	friendGroup.Post("/accept", handlers.AcceptFriendRequest)
	// friendGroup.Delete("/:id", handlers.RemoveFriend)
	// friendGroup.Get("/requests", handlers.GetFriendRequests)

	// Leaderboard routes
	leaderboardGroup := api.Group("/leaderboard")
	leaderboardGroup.Get("/", handlers.GetLeaderboard)
	leaderboardGroup.Get("/season", handlers.GetSeasonLeaderboard)
	leaderboardGroup.Get("/user/:id", handlers.GetUserRank)
	leaderboardGroup.Get("/around/:id", handlers.GetLeaderboardAroundUser)

	// Team Portal routes
	teamGroup := api.Group("/teams")
	teamGroup.Use(middleware.AuthMiddleware)

	// Public team routes (no auth required)
	api.Get("/teams/public", handlers.GetPublicTeams)
	api.Get("/teams/popular", handlers.GetPopularTeams)
	teamGroup.Post("/", handlers.CreateTeam)
	teamGroup.Get("/", handlers.GetUserTeams)
	teamGroup.Get("/search", handlers.SearchTeams)
	teamGroup.Get("/:id", handlers.GetTeam)
	teamGroup.Put("/:id", handlers.UpdateTeam)
	teamGroup.Delete("/:id", handlers.DeleteTeam)
	teamGroup.Post("/join", handlers.JoinTeam)
	teamGroup.Post("/:id/leave", handlers.LeaveTeam)
	teamGroup.Get("/:id/members", handlers.GetTeamMembers)
	teamGroup.Delete("/:id/members/:memberId", handlers.RemoveMember)
	teamGroup.Put("/:id/members/:memberId/promote", handlers.PromoteMember)
	teamGroup.Put("/:id/members/:memberId/demote", handlers.DemoteMember)
	teamGroup.Put("/:id/transfer", handlers.TransferOwnership)
	teamGroup.Get("/:id/leaderboard", handlers.GetTeamLeaderboard)
	teamGroup.Get("/:id/stats", handlers.GetTeamStats)
	teamGroup.Get("/:id/check-membership", handlers.CheckMembership)

	// Team Theme routes
	teamGroup.Post("/:id/themes", handlers.CreateTeamTheme)
	teamGroup.Get("/:id/themes", handlers.GetTeamThemes)
	teamGroup.Get("/themes", handlers.GetUserTeamThemes) // Get all themes from user's teams

	// Team Challenge routes
	teamGroup.Post("/:id/challenges", handlers.CreateChallenge)
	teamGroup.Get("/:id/challenges", handlers.GetTeamChallenges)
	teamGroup.Get("/challenges", handlers.GetUserTeamChallenges) // Get all challenges from user's teams

	// Admin routes
	adminGroup := api.Group("/admin")
	adminGroup.Post("/login", admin.Login)
	adminGroup.Post("/logout", admin.Logout)

	// Protected admin routes
	adminProtected := adminGroup.Group("")
	adminProtected.Use(middleware.AdminAuthMiddleware)
	adminProtected.Get("/verify", admin.VerifyToken)
	adminProtected.Get("/users", admin.GetUsers)
	adminProtected.Get("/users/:id", admin.GetUser)
	adminProtected.Put("/users/:id", admin.UpdateUser)
	adminProtected.Delete("/users/:id", admin.DeleteUser)
	adminProtected.Post("/users/:id/ban", admin.BanUser)
	adminProtected.Post("/users/:id/reset-password", admin.ResetUserPassword)
	adminProtected.Get("/analytics", admin.GetAnalytics)
	adminProtected.Post("/cleanup/manual", admin.ManualCleanup)
	adminProtected.Get("/cleanup/stats", admin.GetCleanupStats)

	// Admin theme management
	adminProtected.Get("/themes", admin.GetAllThemes)
	adminProtected.Post("/themes", admin.CreateAdminTheme)
	adminProtected.Put("/themes/:id", admin.UpdateAdminTheme)
	adminProtected.Delete("/themes/:id", admin.DeleteAdminTheme)
	// Admin endpoint for bulk verse theme creation
	adminProtected.Post("/themes/from-verses", handlers.CreateThemeFromVerses)

	// Admin achievement management
	adminProtected.Get("/achievements", admin.GetAchievements)
	adminProtected.Post("/achievements", admin.CreateAchievement)
	adminProtected.Put("/achievements/:id", admin.UpdateAchievement)
	adminProtected.Delete("/achievements/:id", admin.DeleteAchievement)

	// Admin challenge management
	adminProtected.Get("/challenges", admin.GetChallenges)
	adminProtected.Post("/challenges", admin.CreateChallenge)
	adminProtected.Put("/challenges/:id", admin.UpdateChallenge)
	adminProtected.Delete("/challenges/:id", admin.DeleteChallenge)

	// Proxy /ws requests from Fiber to the WebSocket server on port 4000
	// This allows clients to keep using ws://localhost:3000/ws
	app.Get("/ws", func(c *fiber.Ctx) error {
		// Redirect to WebSocket server
		wsPort := getEnv("WS_PORT", "4000")
		wsURL := "ws://localhost:" + wsPort + "/ws"

		// For WebSocket, we need to tell the client the correct URL
		// Since Fiber can't proxy WebSocket, inform user of correct port
		return c.Status(fiber.StatusUpgradeRequired).JSON(fiber.Map{
			"error":   "WebSocket endpoint moved",
			"message": "Please connect to " + wsURL,
			"ws_url":  wsURL,
		})
	})

	// Debug endpoints for troubleshooting multiplayer (remove in production)
	api.Get("/debug/rooms", handlers.GetActiveRooms)
	api.Get("/debug/rooms/:code", handlers.GetRoomByCode)
	api.Get("/debug/sessions", handlers.GetGameSessions)

	// Secure game URL route (Chess.com-style)
	app.Get("/game/:gameID", handlers.HandleGameURL)

	// HTML routes
	app.Get("/", serveFile("./static/index.html"))
	app.Get("/quiz", serveFile("./static/quiz.html"))
	app.Get("/quiz.html", serveFile("./static/quiz.html"))
	app.Get("/practice", serveFile("./static/practice.html"))
	app.Get("/practice.html", serveFile("./static/practice.html"))
	app.Get("/challenges", serveFile("./static/challenges.html"))
	app.Get("/challenges.html", serveFile("./static/challenges.html"))
	app.Get("/settings", serveFile("./static/settings.html"))
	app.Get("/settings.html", serveFile("./static/settings.html"))
	app.Get("/shop", serveFile("./static/shop.html"))
	app.Get("/shop.html", serveFile("./static/shop.html"))
	app.Get("/login", serveFile("./static/login.html"))
	app.Get("/login.html", serveFile("./static/login.html"))
	app.Get("/teams", serveFile("./static/teams.html"))
	app.Get("/teams.html", serveFile("./static/teams.html"))
	app.Get("/ai-theme-maker", serveFile("./static/ai-theme-maker.html"))
	app.Get("/ai-theme-maker.html", serveFile("./static/ai-theme-maker.html"))
	app.Get("/theme-maker", serveFile("./static/theme-maker.html"))
	app.Get("/theme-maker.html", serveFile("./static/theme-maker.html"))

	// Admin HTML routes
	app.Get("/admin", serveFile("./static/admin/index.html"))
	app.Get("/admin/login", serveFile("./static/admin/login.html"))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

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

	// Start Fiber HTTP/REST server on port 3000
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 HTTP server starting on port %s", port)
	log.Printf("📊 Environment: %s", getEnv("APP_ENV", "development"))
	log.Printf("🔐 JWT Secret configured: %v", os.Getenv("JWT_SECRET") != "")
	log.Printf("🧹 Guest cleanup: %s", getEnv("GUEST_CLEANUP_ENABLED", "true"))
	log.Printf("🌐 WebSocket available at ws://localhost:%s/ws", wsPort)
	log.Printf("✅ Quiz endpoint available at /api/questions/quiz")

	if err := app.Listen(":" + port); err != nil {
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

// Helper functions
func serveFile(filepath string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendFile(filepath)
	}
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Don't expose internal errors in production
	if os.Getenv("APP_ENV") == "production" && code == 500 {
		message = "An error occurred. Please try again later."
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

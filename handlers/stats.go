package handlers

import (
	"time"
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetOnlinePlayersCount returns the number of currently online players
func GetOnlinePlayersCount(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Database not available",
		})
	}

	// Update current user's activity if authenticated
	userID := c.Locals("userId")
	if userID != nil {
		now := time.Now()
		db.Model(&models.User{}).Where("id = ?", userID).Update("last_activity", now)
	}

	// Count users who have been active in the last 5 minutes
	// This is a simple way to determine "online" status
	cutoffTime := time.Now().Add(-5 * time.Minute)

	var count int64
	err := db.Model(&models.User{}).Where("last_activity > ?", cutoffTime).Count(&count).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get online players count",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"count":   count,
	})
}

// GetLastPlayedTime returns the last played time for the current user
func GetLastPlayedTime(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Database not available",
		})
	}

	// Get user ID from token (if authenticated)
	userID := c.Locals("userId")
	if userID == nil {
		// Return "Never" for unauthenticated users
		return c.JSON(fiber.Map{
			"success":    true,
			"lastPlayed": "Never",
		})
	}

	var user models.User
	err := db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get user data",
		})
	}

	// Check if user has last_activity set
	if user.LastActivity == nil {
		return c.JSON(fiber.Map{
			"success":    true,
			"lastPlayed": "Never",
		})
	}

	// Format the last activity time
	lastPlayed := user.LastActivity.Format("Jan 2, 2006 at 3:04 PM")

	return c.JSON(fiber.Map{
		"success":    true,
		"lastPlayed": lastPlayed,
	})
}

// CheckActiveGame checks if user has an active game session
func CheckActiveGame(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Database not available",
		})
	}

	userID := c.Locals("userId")
	if userID == nil {
		return c.JSON(fiber.Map{
			"success":    true,
			"hasActive":  false,
			"sessionID":  nil,
		})
	}

	var user models.User
	err := db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get user data",
		})
	}

	// Check if there's an active session and if it's still valid (within last 30 minutes)
	if user.ActiveGameSession != nil && user.GameStartedAt != nil {
		timeSinceStart := time.Since(*user.GameStartedAt)
		if timeSinceStart < 30*time.Minute {
			return c.JSON(fiber.Map{
				"success":    true,
				"hasActive":  true,
				"sessionID":  *user.ActiveGameSession,
				"startedAt":  user.GameStartedAt,
			})
		} else {
			// Clear stale session
			db.Model(&user).Updates(map[string]interface{}{
				"active_game_session": nil,
				"game_started_at":     nil,
			})
		}
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"hasActive": false,
		"sessionID": nil,
	})
}

// StartGameSession creates a new game session for the user
func StartGameSession(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Database not available",
		})
	}

	userID := c.Locals("userId")
	if userID == nil {
		// Generate a guest session ID
		sessionID := uuid.New().String()
		return c.JSON(fiber.Map{
			"success":   true,
			"sessionID": sessionID,
		})
	}

	var user models.User
	err := db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get user data",
		})
	}

	// Check if already has active session
	if user.ActiveGameSession != nil && user.GameStartedAt != nil {
		timeSinceStart := time.Since(*user.GameStartedAt)
		if timeSinceStart < 30*time.Minute {
			return c.Status(409).JSON(fiber.Map{
				"success":    false,
				"error":      "You are already playing a game",
				"sessionID":  *user.ActiveGameSession,
				"startedAt":  user.GameStartedAt,
			})
		}
	}

	// Create new session
	sessionID := uuid.New().String()
	now := time.Now()

	err = db.Model(&user).Updates(map[string]interface{}{
		"active_game_session": sessionID,
		"game_started_at":     now,
	}).Error

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create game session",
		})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"sessionID": sessionID,
		"startedAt": now,
	})
}

// EndGameSession clears the active game session
func EndGameSession(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Database not available",
		})
	}

	userID := c.Locals("userId")
	if userID == nil {
		return c.JSON(fiber.Map{
			"success": true,
		})
	}

	err := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"active_game_session": nil,
		"game_started_at":     nil,
	}).Error

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to end game session",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}


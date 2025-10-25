package handlers

import (
	"net/http"
	"time"
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"ubible/utils"

	"github.com/google/uuid"
)

// GetOnlinePlayersCount returns the number of currently online players
func GetOnlinePlayersCount(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	if db == nil {
		utils.JSONError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Update current user's activity if authenticated
	userID, err := middleware.GetUserID(r)
	if err == nil {
		now := time.Now()
		db.Model(&models.User{}).Where("id = ?", userID).Update("last_activity", now)
	}

	// Count users who have been active in the last 5 minutes
	cutoffTime := time.Now().Add(-5 * time.Minute)

	var count int64
	err = db.Model(&models.User{}).Where("last_activity > ?", cutoffTime).Count(&count).Error
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to get online players count")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   count,
	})
}

// GetLastPlayedTime returns the last played time for the current user
func GetLastPlayedTime(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	if db == nil {
		utils.JSONError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Get user ID from token (if authenticated)
	userID, err := middleware.GetUserID(r)
	if err != nil {
		// Return "Never" for unauthenticated users
		utils.JSON(w, http.StatusOK, map[string]interface{}{
			"success":    true,
			"lastPlayed": "Never",
		})
		return
	}

	var user models.User
	err = db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to get user data")
		return
	}

	// Check if user has last_activity set
	if user.LastActivity == nil {
		utils.JSON(w, http.StatusOK, map[string]interface{}{
			"success":    true,
			"lastPlayed": "Never",
		})
		return
	}

	// Format the last activity time
	lastPlayed := user.LastActivity.Format("Jan 2, 2006 at 3:04 PM")

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"lastPlayed": lastPlayed,
	})
}

// CheckActiveGame checks if user has an active game session
func CheckActiveGame(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	if db == nil {
		utils.JSONError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSON(w, http.StatusOK, map[string]interface{}{
			"success":   true,
			"hasActive": false,
			"sessionID": nil,
		})
		return
	}

	var user models.User
	err = db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to get user data")
		return
	}

	// Check if there's an active session and if it's still valid (within last 30 minutes)
	if user.ActiveGameSession != nil && user.GameStartedAt != nil {
		timeSinceStart := time.Since(*user.GameStartedAt)
		if timeSinceStart < 30*time.Minute {
			utils.JSON(w, http.StatusOK, map[string]interface{}{
				"success":   true,
				"hasActive": true,
				"sessionID": *user.ActiveGameSession,
				"startedAt": user.GameStartedAt,
			})
			return
		} else {
			// Clear stale session
			db.Model(&user).Updates(map[string]interface{}{
				"active_game_session": nil,
				"game_started_at":     nil,
			})
		}
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"hasActive": false,
		"sessionID": nil,
	})
}

// StartGameSession creates a new game session for the user
func StartGameSession(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	if db == nil {
		utils.JSONError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	userID, err := middleware.GetUserID(r)
	if err != nil {
		// Generate a guest session ID
		sessionID := uuid.New().String()
		utils.JSON(w, http.StatusOK, map[string]interface{}{
			"success":   true,
			"sessionID": sessionID,
		})
		return
	}

	var user models.User
	err = db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to get user data")
		return
	}

	// Check if already has active session
	if user.ActiveGameSession != nil && user.GameStartedAt != nil {
		timeSinceStart := time.Since(*user.GameStartedAt)
		if timeSinceStart < 30*time.Minute {
			utils.JSON(w, http.StatusConflict, map[string]interface{}{
				"success":   false,
				"error":     "You are already playing a game",
				"sessionID": *user.ActiveGameSession,
				"startedAt": user.GameStartedAt,
			})
			return
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
		utils.JSONError(w, http.StatusInternalServerError, "Failed to create game session")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"sessionID": sessionID,
		"startedAt": now,
	})
}

// EndGameSession clears the active game session
func EndGameSession(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	if db == nil {
		utils.JSONError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
		return
	}

	err = db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"active_game_session": nil,
		"game_started_at":     nil,
	}).Error

	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to end game session")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

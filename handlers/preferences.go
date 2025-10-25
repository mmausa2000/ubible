package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"ubible/utils"
)

type PreferencesRequest struct {
	SelectedThemes    []int `json:"selected_themes"`
	QuizTimeLimit     int   `json:"quiz_time_limit"`
	QuizQuestionCount int   `json:"quiz_question_count"`
}

// SavePreferences saves user's quiz preferences
func SavePreferences(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()

	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req PreferencesRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if req.QuizTimeLimit < 10 || req.QuizTimeLimit > 300 {
		utils.JSONError(w, http.StatusBadRequest, "Time limit must be between 10 and 300 seconds")
		return
	}

	if req.QuizQuestionCount < 1 || req.QuizQuestionCount > 100 {
		utils.JSONError(w, http.StatusBadRequest, "Question count must be between 1 and 100")
		return
	}

	if len(req.SelectedThemes) > 5 {
		utils.JSONError(w, http.StatusBadRequest, "Maximum 5 themes allowed")
		return
	}

	// Convert selected themes to JSON
	themesJSON, err := json.Marshal(req.SelectedThemes)
	if err != nil {
		log.Printf("Error marshaling themes: %v", err)
		utils.JSONError(w, http.StatusInternalServerError, "Failed to save preferences")
		return
	}

	// Update user preferences
	result := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"selected_themes":     string(themesJSON),
		"quiz_time_limit":     req.QuizTimeLimit,
		"quiz_question_count": req.QuizQuestionCount,
	})

	if result.Error != nil {
		log.Printf("Error saving preferences: %v", result.Error)
		utils.JSONError(w, http.StatusInternalServerError, "Failed to save preferences")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Preferences saved successfully",
	})
}

// GetPreferences retrieves user's quiz preferences
func GetPreferences(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()

	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.JSONError(w, http.StatusNotFound, "User not found")
		return
	}

	// Parse selected themes from JSON
	var selectedThemes []int
	if user.SelectedThemes != "" {
		if err := json.Unmarshal([]byte(user.SelectedThemes), &selectedThemes); err != nil {
			log.Printf("Error parsing selected themes: %v", err)
			selectedThemes = []int{}
		}
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success":             true,
		"selected_themes":     selectedThemes,
		"quiz_time_limit":     user.QuizTimeLimit,
		"quiz_question_count": user.QuizQuestionCount,
	})
}

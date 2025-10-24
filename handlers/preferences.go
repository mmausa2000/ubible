package handlers

import (
	"encoding/json"
	"log"
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/fiber/v2"
)

type PreferencesRequest struct {
	SelectedThemes    []int `json:"selected_themes"`
	QuizTimeLimit     int   `json:"quiz_time_limit"`
	QuizQuestionCount int   `json:"quiz_question_count"`
}

// SavePreferences saves user's quiz preferences
func SavePreferences(c *fiber.Ctx) error {
	db := database.GetDB()

	// Get user ID from JWT token (middleware sets "userId")
	userIDRaw := c.Locals("userId")
	if userIDRaw == nil {
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// Handle both float64 (from JWT claims) and uint
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid user ID type",
		})
	}

	var req PreferencesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate
	if req.QuizTimeLimit < 10 || req.QuizTimeLimit > 300 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Time limit must be between 10 and 300 seconds",
		})
	}

	if req.QuizQuestionCount < 1 || req.QuizQuestionCount > 100 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Question count must be between 1 and 100",
		})
	}

	if len(req.SelectedThemes) > 5 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Maximum 5 themes allowed",
		})
	}

	// Convert selected themes to JSON
	themesJSON, err := json.Marshal(req.SelectedThemes)
	if err != nil {
		log.Printf("Error marshaling themes: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to save preferences",
		})
	}

	// Update user preferences
	result := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"selected_themes":      string(themesJSON),
		"quiz_time_limit":      req.QuizTimeLimit,
		"quiz_question_count":  req.QuizQuestionCount,
	})

	if result.Error != nil {
		log.Printf("Error saving preferences: %v", result.Error)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to save preferences",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Preferences saved successfully",
	})
}

// GetPreferences retrieves user's quiz preferences
func GetPreferences(c *fiber.Ctx) error {
	db := database.GetDB()

	// Get user ID from JWT token (middleware sets "userId")
	userIDRaw := c.Locals("userId")
	if userIDRaw == nil {
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// Handle both float64 (from JWT claims) and uint
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid user ID type",
		})
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "User not found",
		})
	}

	// Parse selected themes from JSON
	var selectedThemes []int
	if user.SelectedThemes != "" {
		if err := json.Unmarshal([]byte(user.SelectedThemes), &selectedThemes); err != nil {
			log.Printf("Error parsing selected themes: %v", err)
			selectedThemes = []int{}
		}
	}

	return c.JSON(fiber.Map{
		"success":              true,
		"selected_themes":      selectedThemes,
		"quiz_time_limit":      user.QuizTimeLimit,
		"quiz_question_count":  user.QuizQuestionCount,
	})
}

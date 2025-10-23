// handlers/themes.go
package handlers

import (
	"ubible/database"
	"ubible/models"
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
)

// GetThemes returns all active themes with question counts (Public endpoint)
func GetThemes(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		log.Println("Database not initialized in GetThemes")
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Database not available",
		})
	}

	var themes []models.Theme
	// Only return active themes for public endpoint
	if err := db.Where("is_active = ?", true).Find(&themes).Error; err != nil {
		log.Printf("Error fetching themes: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch themes",
		})
	}

	// Build response with question counts
	themesData := make([]fiber.Map, len(themes))
	for i, theme := range themes {
		var questionCount int64
		db.Model(&models.Question{}).Where("theme_id = ?", theme.ID).Count(&questionCount)

		themesData[i] = fiber.Map{
			"id":             theme.ID,
			"name":           theme.Name,
			"description":    theme.Description,
			"icon":           theme.Icon,
			"color":          theme.Color,
			"question_count": questionCount,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"themes":  themesData,
		"total":   len(themesData),
	})
}

// GetTheme returns a single theme with its questions
func GetTheme(c *fiber.Ctx) error {
	themeID := c.Params("id")
	db := database.GetDB()

	var theme models.Theme
	if err := db.First(&theme, themeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Theme not found",
		})
	}

	// Get questions for this theme
	var questions []models.Question
	if err := db.Where("theme_id = ?", themeID).Find(&questions).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch questions",
		})
	}

	// Transform questions to include parsed options
	questionsData := make([]fiber.Map, len(questions))
	for i, q := range questions {
		var wrongAnswers []string
		if q.WrongAnswers != "" {
			json.Unmarshal([]byte(q.WrongAnswers), &wrongAnswers)
		}

		options := append([]string{q.CorrectAnswer}, wrongAnswers...)

		questionsData[i] = fiber.Map{
			"id":             q.ID,
			"text":           q.Text,
			"correct_answer": q.CorrectAnswer,
			"options":        options,
			"reference":      q.Reference,
			"difficulty":     q.Difficulty,
		}
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"theme":     theme,
		"questions": questionsData,
		"count":     len(questionsData),
	})
}

// CreateTheme creates a new theme (requires auth)
func CreateTheme(c *fiber.Ctx) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Theme name is required",
		})
	}

	db := database.GetDB()

	// Check for duplicate
	var existing models.Theme
	if err := db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{
			"success": false,
			"error":   "Theme with this name already exists",
		})
	}

	theme := models.Theme{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		IsActive:    true,
		IsDefault:   false,
	}

	if err := db.Create(&theme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create theme",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Theme created successfully",
		"theme":   theme,
	})
}

// UpdateTheme updates an existing theme (requires auth)
func UpdateTheme(c *fiber.Ctx) error {
	themeID := c.Params("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	db := database.GetDB()

	var theme models.Theme
	if err := db.First(&theme, themeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Theme not found",
		})
	}

	// Update fields if provided
	if req.Name != "" {
		theme.Name = req.Name
	}
	if req.Description != "" {
		theme.Description = req.Description
	}
	if req.Icon != "" {
		theme.Icon = req.Icon
	}
	if req.Color != "" {
		theme.Color = req.Color
	}
	if req.IsActive != nil {
		theme.IsActive = *req.IsActive
	}

	if err := db.Save(&theme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update theme",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Theme updated successfully",
		"theme":   theme,
	})
}

// DeleteTheme deletes a theme (requires auth)
func DeleteTheme(c *fiber.Ctx) error {
	themeID := c.Params("id")
	db := database.GetDB()

	var theme models.Theme
	if err := db.First(&theme, themeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Theme not found",
		})
	}

	// Don't allow deleting default themes
	if theme.IsDefault {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Cannot delete default themes",
		})
	}

	// Delete associated questions first
	db.Where("theme_id = ?", themeID).Delete(&models.Question{})

	// Delete theme
	if err := db.Delete(&theme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete theme",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Theme deleted successfully",
	})
}
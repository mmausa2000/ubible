// handlers/admin/themes.go
package admin

import (
	"ubible/database"
	"ubible/models"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// GetAllThemes returns all themes including inactive ones
func GetAllThemes(c *fiber.Ctx) error {
	db := database.GetDB()
	
	var themes []models.Theme
	if err := db.Preload("Questions").Find(&themes).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch themes",
		})
	}

	// Add question count to each theme
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
			"is_active":      theme.IsActive,
			"is_default":     theme.IsDefault,
			"unlock_cost":    theme.UnlockCost,
			"question_count": questionCount,
			"created_at":     theme.CreatedAt,
			"updated_at":     theme.UpdatedAt,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"themes":  themesData,
		"total":   len(themes),
	})
}

// CreateAdminTheme creates a new theme with questions
func CreateAdminTheme(c *fiber.Ctx) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		IsActive    bool   `json:"is_active"`
		IsDefault   bool   `json:"is_default"`
		UnlockCost  int    `json:"unlock_cost"`
		Questions   []struct {
			Text          string   `json:"text"`
			CorrectAnswer string   `json:"correct_answer"`
			WrongAnswers  []string `json:"wrong_answers"`
			Reference     string   `json:"reference"`
			Difficulty    string   `json:"difficulty"`
		} `json:"questions"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Theme name is required",
		})
	}

	db := database.GetDB()

	// Check if theme with same name exists
	var existing models.Theme
	if err := db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{
			"error": "Theme with this name already exists",
		})
	}

	// Create theme
	theme := models.Theme{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		IsActive:    req.IsActive,
		IsDefault:   req.IsDefault,
		UnlockCost:  req.UnlockCost,
	}

	if err := db.Create(&theme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create theme",
		})
	}

	// Create questions
	for _, q := range req.Questions {
		wrongAnswersJSON, _ := json.Marshal(q.WrongAnswers)
		
		question := models.Question{
			ThemeID:       theme.ID,
			Text:          q.Text,
			CorrectAnswer: q.CorrectAnswer,
			WrongAnswers:  string(wrongAnswersJSON),
			Reference:     q.Reference,
			Difficulty:    q.Difficulty,
		}

		if err := db.Create(&question).Error; err != nil {
			// Log error but continue
			continue
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Theme created successfully",
		"theme":   theme,
	})
}

// UpdateAdminTheme updates an existing theme
func UpdateAdminTheme(c *fiber.Ctx) error {
	themeID := c.Params("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		IsActive    bool   `json:"is_active"`
		IsDefault   bool   `json:"is_default"`
		UnlockCost  int    `json:"unlock_cost"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	db := database.GetDB()

	var theme models.Theme
	if err := db.First(&theme, themeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Theme not found",
		})
	}

	// Update fields
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
	theme.IsActive = req.IsActive
	theme.IsDefault = req.IsDefault
	theme.UnlockCost = req.UnlockCost

	if err := db.Save(&theme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update theme",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Theme updated successfully",
		"theme":   theme,
	})
}

// DeleteAdminTheme deletes a theme and its questions
func DeleteAdminTheme(c *fiber.Ctx) error {
	themeID := c.Params("id")

	db := database.GetDB()

	var theme models.Theme
	if err := db.First(&theme, themeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Theme not found",
		})
	}

	// Don't allow deleting default themes
	if theme.IsDefault {
		return c.Status(403).JSON(fiber.Map{
			"error": "Cannot delete default themes",
		})
	}

	// Check if theme is being used
	var attemptCount int64
	db.Model(&models.Attempt{}).Where("theme_id = ?", theme.ID).Count(&attemptCount)

	if attemptCount > 0 {
		return c.Status(409).JSON(fiber.Map{
			"error":   "Cannot delete theme that has been used in games",
			"message": "Consider deactivating the theme instead",
			"attempts": attemptCount,
		})
	}

	// Delete all questions first
	if err := db.Where("theme_id = ?", theme.ID).Delete(&models.Question{}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete theme questions",
		})
	}

	// Delete theme
	if err := db.Delete(&theme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete theme",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Theme deleted successfully",
	})
}

// AddQuestionToTheme adds a question to an existing theme
func AddQuestionToTheme(c *fiber.Ctx) error {
	themeID := c.Params("id")

	var req struct {
		Text          string   `json:"text"`
		CorrectAnswer string   `json:"correct_answer"`
		WrongAnswers  []string `json:"wrong_answers"`
		Reference     string   `json:"reference"`
		Difficulty    string   `json:"difficulty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Text == "" || req.CorrectAnswer == "" || len(req.WrongAnswers) < 2 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Text, correct answer, and at least 2 wrong answers are required",
		})
	}

	db := database.GetDB()

	// Verify theme exists
	var theme models.Theme
	if err := db.First(&theme, themeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Theme not found",
		})
	}

	wrongAnswersJSON, _ := json.Marshal(req.WrongAnswers)
	
	question := models.Question{
		ThemeID:       theme.ID,
		Text:          req.Text,
		CorrectAnswer: req.CorrectAnswer,
		WrongAnswers:  string(wrongAnswersJSON),
		Reference:     req.Reference,
		Difficulty:    req.Difficulty,
	}

	if err := db.Create(&question).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to add question",
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "Question added successfully",
		"question": question,
	})
}

// BulkAddQuestions adds multiple questions to a theme
func BulkAddQuestions(c *fiber.Ctx) error {
	themeID := c.Params("id")

	var req struct {
		Questions []struct {
			Text          string   `json:"text"`
			CorrectAnswer string   `json:"correct_answer"`
			WrongAnswers  []string `json:"wrong_answers"`
			Reference     string   `json:"reference"`
			Difficulty    string   `json:"difficulty"`
		} `json:"questions"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	db := database.GetDB()

	// Verify theme exists
	var theme models.Theme
	if err := db.First(&theme, themeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Theme not found",
		})
	}

	successCount := 0
	failCount := 0

	for _, q := range req.Questions {
		if q.Text == "" || q.CorrectAnswer == "" || len(q.WrongAnswers) < 2 {
			failCount++
			continue
		}

		wrongAnswersJSON, _ := json.Marshal(q.WrongAnswers)
		
		question := models.Question{
			ThemeID:       theme.ID,
			Text:          q.Text,
			CorrectAnswer: q.CorrectAnswer,
			WrongAnswers:  string(wrongAnswersJSON),
			Reference:     q.Reference,
			Difficulty:    q.Difficulty,
		}

		if err := db.Create(&question).Error; err != nil {
			failCount++
			continue
		}
		successCount++
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"message":       "Bulk add completed",
		"added":         successCount,
		"failed":        failCount,
		"total":         len(req.Questions),
	})
}

// GetThemeQuestions returns all questions for a theme
func GetThemeQuestions(c *fiber.Ctx) error {
	themeID := c.Params("id")

	db := database.GetDB()

	// Verify theme exists
	var theme models.Theme
	if err := db.First(&theme, themeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Theme not found",
		})
	}

	var questions []models.Question
	if err := db.Where("theme_id = ?", themeID).Find(&questions).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch questions",
		})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"theme":     theme,
		"questions": questions,
		"total":     len(questions),
	})
}

// UpdateQuestion updates a specific question
func UpdateQuestion(c *fiber.Ctx) error {
	questionID := c.Params("questionId")

	var req struct {
		Text          string   `json:"text"`
		CorrectAnswer string   `json:"correct_answer"`
		WrongAnswers  []string `json:"wrong_answers"`
		Reference     string   `json:"reference"`
		Difficulty    string   `json:"difficulty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	db := database.GetDB()

	var question models.Question
	if err := db.First(&question, questionID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Question not found",
		})
	}

	// Update fields
	if req.Text != "" {
		question.Text = req.Text
	}
	if req.CorrectAnswer != "" {
		question.CorrectAnswer = req.CorrectAnswer
	}
	if len(req.WrongAnswers) >= 2 {
		wrongAnswersJSON, _ := json.Marshal(req.WrongAnswers)
		question.WrongAnswers = string(wrongAnswersJSON)
	}
	if req.Reference != "" {
		question.Reference = req.Reference
	}
	if req.Difficulty != "" {
		question.Difficulty = req.Difficulty
	}

	if err := db.Save(&question).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update question",
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "Question updated successfully",
		"question": question,
	})
}

// DeleteQuestion deletes a specific question
func DeleteQuestion(c *fiber.Ctx) error {
	questionID := c.Params("questionId")

	db := database.GetDB()

	var question models.Question
	if err := db.First(&question, questionID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Question not found",
		})
	}

	if err := db.Delete(&question).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete question",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Question deleted successfully",
	})
}

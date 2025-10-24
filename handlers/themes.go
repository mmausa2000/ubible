// handlers/themes.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
	// Only return active themes for public endpoint, and preload creator info
	if err := db.Preload("Creator").Where("is_active = ?", true).Find(&themes).Error; err != nil {
		log.Printf("Error fetching themes: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch themes",
		})
	}

	// Build response with question counts and creator info
	themesData := make([]fiber.Map, len(themes))
	for i, theme := range themes {
		var questionCount int64
		db.Model(&models.Question{}).Where("theme_id = ?", theme.ID).Count(&questionCount)

		createdBy := ""
		if theme.Creator != nil {
			createdBy = theme.Creator.Username
		}

		themesData[i] = fiber.Map{
			"id":             theme.ID,
			"name":           theme.Name,
			"description":    theme.Description,
			"icon":           theme.Icon,
			"color":          theme.Color,
			"question_count": questionCount,
			"created_by":     createdBy,
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

// CreateTheme creates a new theme with verses (requires auth)
func CreateTheme(c *fiber.Ctx) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		Verses      []struct {
			Reference string `json:"reference"`
			Text      string `json:"text"`
		} `json:"verses"`
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

	if len(req.Verses) < 5 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "At least 5 verses are required",
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

	// Create questions for each verse
	successCount := 0
	for i, verse := range req.Verses {
		if verse.Reference == "" || verse.Text == "" {
			log.Printf("Skipping verse %d: empty reference or text", i)
			continue
		}

		// Skip if text is just the reference (cross-reference without actual verse text)
		if strings.TrimSpace(verse.Text) == strings.TrimSpace(verse.Reference) {
			log.Printf("Skipping verse %d: text is same as reference (cross-reference only): %s", i, verse.Reference)
			continue
		}

		// Skip if text is too short (likely just a reference)
		if len(strings.TrimSpace(verse.Text)) < 10 {
			log.Printf("Skipping verse %d: text too short (likely reference only): %s", i, verse.Text)
			continue
		}
		question := models.Question{
			ThemeID:       theme.ID,
			Text:          verse.Text,
			CorrectAnswer: verse.Reference,
			WrongAnswers:  "[]", // Empty array for now
			Reference:     verse.Reference,
			Difficulty:    "medium",
		}
		if err := db.Create(&question).Error; err != nil {
			log.Printf("Error creating question %d for theme %d: %v", i, theme.ID, err)
			// Continue creating other questions even if one fails
		} else {
			successCount++
			log.Printf("Created question %d for theme %d: %s", i, theme.ID, verse.Reference)
		}
	}

	log.Printf("Theme %d created with %d/%d verses", theme.ID, successCount, len(req.Verses))

	return c.JSON(fiber.Map{
		"success":        true,
		"message":        "Theme created successfully",
		"theme":          theme,
		"verses_created": successCount,
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

// CreatePublicTheme creates a new public theme (no auth required)
func CreatePublicTheme(c *fiber.Ctx) error {
	var req struct {
		Name   string `json:"name"`
		Verses []struct {
			Reference string `json:"reference"`
			Text      string `json:"text"`
		} `json:"verses"`
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

	if len(req.Verses) < 5 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "At least 5 verses are required",
		})
	}

	if len(req.Verses) > 500 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Maximum 500 verses allowed",
		})
	}

	db := database.GetDB()

	var existing models.Theme
	if err := db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{
			"success": false,
			"error":   "Theme with this name already exists",
		})
	}

	theme := models.Theme{
		Name:           req.Name,
		Description:    "",
		Icon:           "📖",
		Color:          "#4caf50",
		IsActive:       true,
		IsPublic:       true,
		CreatedByGuest: true,
		CreatedBy:      nil,
	}

	if err := db.Create(&theme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create theme",
		})
	}

	// Generate questions from verses - alternating between two question types
	successCount := 0
	for i, verse := range req.Verses {
		if verse.Reference == "" || verse.Text == "" {
			continue
		}

		// Clean the verse text
		cleanText := strings.TrimPrefix(verse.Text, "- ")
		cleanText = strings.TrimSpace(cleanText)

		var question models.Question

		// Long verses (>200 chars) always show verse → pick reference
		if len(cleanText) > 200 {
			wrongRefs := generateWrongReferences(db, verse.Reference, req.Verses)
			wrongAnswersJSON, _ := json.Marshal(wrongRefs)

			question = models.Question{
				ThemeID:       theme.ID,
				ThemeName:     theme.Name,
				Text:          cleanText,
				CorrectAnswer: verse.Reference,
				WrongAnswers:  string(wrongAnswersJSON),
				Reference:     verse.Reference,
				Difficulty:    "medium",
			}
		} else if i%2 == 0 {
			// Short verses alternate: verse → reference
			wrongRefs := generateWrongReferences(db, verse.Reference, req.Verses)
			wrongAnswersJSON, _ := json.Marshal(wrongRefs)

			question = models.Question{
				ThemeID:       theme.ID,
				ThemeName:     theme.Name,
				Text:          cleanText,
				CorrectAnswer: verse.Reference,
				WrongAnswers:  string(wrongAnswersJSON),
				Reference:     verse.Reference,
				Difficulty:    "medium",
			}
		} else {
			// Short verses alternate: reference → verse
			wrongTexts := generateWrongVerses(db, cleanText, verse.Reference, req.Verses)
			wrongAnswersJSON, _ := json.Marshal(wrongTexts)

			question = models.Question{
				ThemeID:       theme.ID,
				ThemeName:     theme.Name,
				Text:          verse.Reference,
				CorrectAnswer: cleanText,
				WrongAnswers:  string(wrongAnswersJSON),
				Reference:     verse.Reference,
				Difficulty:    "medium",
			}
		}

		if err := db.Create(&question).Error; err == nil {
			successCount++
		}
	}

	return c.JSON(fiber.Map{
		"success":        true,
		"message":        "Theme created successfully",
		"theme":          theme,
		"verses_created": successCount,
		"total_verses":   len(req.Verses),
	})
}

// CreateThemeFromVerses creates a new theme from bulk verses (admin endpoint - protected)
func CreateThemeFromVerses(c *fiber.Ctx) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		Verses      []struct {
			Reference string `json:"reference"`
			Text      string `json:"text"`
		} `json:"verses"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validation
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Theme name is required",
		})
	}

	if len(req.Verses) < 5 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "At least 5 verses are required",
		})
	}

	if len(req.Verses) > 500 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Maximum 500 verses allowed",
		})
	}

	db := database.GetDB()

	// Check for duplicate theme name
	var existing models.Theme
	if err := db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{
			"success": false,
			"error":   "Theme with this name already exists",
		})
	}

	// Create theme - marked as not file-backed since it's user-created
	theme := models.Theme{
		Name:         req.Name,
		Description:  req.Description,
		Icon:         req.Icon,
		Color:        req.Color,
		IsActive:     true,
		IsDefault:    false,
		IsFileBacked: false, // User-created themes are not file-backed
	}

	if err := db.Create(&theme).Error; err != nil {
		log.Printf("Error creating theme: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create theme",
		})
	}

	log.Printf("🎨 Created theme %d: '%s' for admin user", theme.ID, req.Name)

	// Create questions for each verse
	successCount := 0
	failureCount := 0

	for i, verse := range req.Verses {
		if verse.Reference == "" || verse.Text == "" {
			log.Printf("⚠️  Skipping verse %d: empty reference or text", i)
			failureCount++
			continue
		}

		question := models.Question{
			ThemeID:       theme.ID,
			Text:          verse.Text,
			CorrectAnswer: verse.Reference,
			WrongAnswers:  "[]", // Empty for now - could be enhanced with similar verses
			Reference:     verse.Reference,
			Difficulty:    "medium",
		}

		if err := db.Create(&question).Error; err != nil {
			log.Printf("❌ Error creating question %d for theme %d: %v", i, theme.ID, err)
			failureCount++
		} else {
			successCount++
			if i < 3 || (i+1)%50 == 0 { // Log first 3 and every 50th
				log.Printf("✅ Created question %d: %s", i, verse.Reference)
			}
		}
	}

	log.Printf("✅ Theme %d finalized: %d/%d verses created successfully (%d failed)",
		theme.ID, successCount, len(req.Verses), failureCount)

	return c.Status(201).JSON(fiber.Map{
		"success":        true,
		"message":        "Theme created successfully",
		"theme":          theme,
		"verses_created": successCount,
		"verses_failed":  failureCount,
		"total_verses":   len(req.Verses),
	})
}

// generateWrongReferences generates wrong references from same theme verses
func generateWrongReferences(_ *gorm.DB, correctRef string, themeVerses []struct {
	Reference string `json:"reference"`
	Text      string `json:"text"`
}) []string {
	wrong := []string{}
	used := map[string]bool{correctRef: true}

	// Get all other verses from the same theme
	for _, v := range themeVerses {
		if len(wrong) >= 3 {
			break
		}
		if !used[v.Reference] && v.Reference != correctRef {
			wrong = append(wrong, v.Reference)
			used[v.Reference] = true
		}
	}

	// Fallback: generate random references if not enough verses in theme
	if len(wrong) < 3 {
		books := []string{"John", "Matthew", "Mark", "Luke", "Romans", "Genesis", "Psalms", "Proverbs", "Isaiah", "Jeremiah"}
		for len(wrong) < 3 {
			book := books[rand.Intn(len(books))]
			chapter := rand.Intn(20) + 1
			verse := rand.Intn(30) + 1
			ref := fmt.Sprintf("%s %d:%d", book, chapter, verse)
			if !used[ref] {
				wrong = append(wrong, ref)
				used[ref] = true
			}
		}
	}

	return wrong
}

// generateWrongVerses generates wrong verse texts from same theme verses
func generateWrongVerses(_ *gorm.DB, correctText string, _ string, themeVerses []struct {
	Reference string `json:"reference"`
	Text      string `json:"text"`
}) []string {
	wrong := []string{}
	used := map[string]bool{correctText: true}

	// Get all other verses from the same theme
	for _, v := range themeVerses {
		if len(wrong) >= 3 {
			break
		}
		cleanText := strings.TrimPrefix(v.Text, "- ")
		cleanText = strings.TrimSpace(cleanText)
		if !used[cleanText] && cleanText != correctText && cleanText != "" {
			wrong = append(wrong, cleanText)
			used[cleanText] = true
		}
	}

	// Fallback: generate generic text if not enough verses in theme
	for len(wrong) < 3 {
		wrong = append(wrong, fmt.Sprintf("Alternative verse text %d", len(wrong)+1))
	}

	return wrong
}

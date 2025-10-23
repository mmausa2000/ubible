// handlers/team_themes.go - Team Theme Library Handlers
package handlers

import (
	"ubible/database"
	"ubible/models"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ================== TEAM THEME CRUD ENDPOINTS ==================

// CreateTeamTheme creates a new team theme
// POST /api/teams/:id/themes
func CreateTeamTheme(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	// Verify user is team admin
	if !teamService.IsTeamAdmin(userID, uint(teamID)) {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Only team admins can create themes",
		})
	}

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		VerseFile   string   `json:"verse_file"`
		IsPublic    bool     `json:"is_public"`
		Tags        []string `json:"tags"`
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

	if req.VerseFile == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Verse file is required",
		})
	}

	// Create theme
	theme := &models.TeamTheme{
		TeamID:      uint(teamID),
		Name:        req.Name,
		Description: req.Description,
		VerseFile:   req.VerseFile,
		IsPublic:    req.IsPublic,
		Tags:        req.Tags,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		IsActive:    true,
	}

	db := database.GetDB()
	if err := db.Create(theme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create theme",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "Theme created successfully",
		"theme":   theme,
	})
}

// GetTeamThemes retrieves all themes for a team
// GET /api/teams/:id/themes
func GetTeamThemes(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	db := database.GetDB()
	var themes []models.TeamTheme
	
	query := db.Where("team_id = ? AND is_active = ?", uint(teamID), true)
	
	// Filter by tag if provided
	if tag := c.Query("tag"); tag != "" {
		query = query.Where("? = ANY(tags)", tag)
	}

	if err := query.
		Preload("Creator").
		Order("usage_count DESC, created_at DESC").
		Find(&themes).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve themes",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"themes":  themes,
		"count":   len(themes),
	})
}

// GetTeamTheme retrieves a specific team theme
// GET /api/themes/:id
func GetTeamTheme(c *fiber.Ctx) error {
	themeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid theme ID",
		})
	}

	db := database.GetDB()
	var theme models.TeamTheme
	if err := db.Where("id = ? AND is_active = ?", uint(themeID), true).
		Preload("Creator").
		First(&theme).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Theme not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"theme":   theme,
	})
}

// UpdateTeamTheme updates a team theme
// PUT /api/themes/:id
func UpdateTeamTheme(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	themeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid theme ID",
		})
	}

	db := database.GetDB()
	var theme models.TeamTheme
	if err := db.First(&theme, uint(themeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Theme not found",
		})
	}

	// Verify user is admin or creator
	if !teamService.IsTeamAdmin(userID, theme.TeamID) && theme.CreatedBy != userID {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Not authorized to update this theme",
		})
	}

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		IsPublic    bool     `json:"is_public"`
		Tags        []string `json:"tags"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Update fields
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	updates["is_public"] = req.IsPublic
	if len(req.Tags) > 0 {
		updates["tags"] = req.Tags
	}

	if err := db.Model(&theme).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update theme",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Theme updated successfully",
	})
}

// DeleteTeamTheme soft deletes a team theme
// DELETE /api/themes/:id
func DeleteTeamTheme(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	themeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid theme ID",
		})
	}

	db := database.GetDB()
	var theme models.TeamTheme
	if err := db.First(&theme, uint(themeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Theme not found",
		})
	}

	// Verify user is admin or creator
	if !teamService.IsTeamAdmin(userID, theme.TeamID) && theme.CreatedBy != userID {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Not authorized to delete this theme",
		})
	}

	// Soft delete
	if err := db.Model(&theme).Update("is_active", false).Error; err != nil {
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

// ================== THEME USAGE & STATISTICS ==================

// IncrementThemeUsage increments usage count when theme is used
// POST /api/themes/:id/use
func IncrementThemeUsage(c *fiber.Ctx) error {
	themeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid theme ID",
		})
	}

	db := database.GetDB()
	if err := db.Model(&models.TeamTheme{}).
		Where("id = ?", uint(themeID)).
		Updates(map[string]interface{}{
			"usage_count": db.Raw("usage_count + 1"),
			"last_used":   time.Now(),
		}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update usage count",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Theme usage recorded",
	})
}

// GetPopularTeamThemes retrieves most used themes for a team
// GET /api/teams/:id/themes/popular
func GetPopularTeamThemes(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	db := database.GetDB()
	var themes []models.TeamTheme
	if err := db.Where("team_id = ? AND is_active = ?", uint(teamID), true).
		Order("usage_count DESC, created_at DESC").
		Limit(limit).
		Preload("Creator").
		Find(&themes).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve popular themes",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"themes":  themes,
		"count":   len(themes),
	})
}

// GetRecentTeamThemes retrieves recently created themes
// GET /api/teams/:id/themes/recent
func GetRecentTeamThemes(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	db := database.GetDB()
	var themes []models.TeamTheme
	if err := db.Where("team_id = ? AND is_active = ?", uint(teamID), true).
		Order("created_at DESC").
		Limit(limit).
		Preload("Creator").
		Find(&themes).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve recent themes",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"themes":  themes,
		"count":   len(themes),
	})
}

// ================== PUBLIC THEME DISCOVERY ==================

// SearchPublicThemes searches for public themes across all teams
// GET /api/themes/public/search
func SearchPublicThemes(c *fiber.Ctx) error {
	query := c.Query("q", "")
	tag := c.Query("tag", "")
	limit := 20

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	db := database.GetDB()
	searchQuery := db.Where("is_public = ? AND is_active = ?", true, true)

	if query != "" {
		searchQuery = searchQuery.Where("name LIKE ? OR description LIKE ?", 
			"%"+query+"%", "%"+query+"%")
	}

	if tag != "" {
		searchQuery = searchQuery.Where("? = ANY(tags)", tag)
	}

	var themes []models.TeamTheme
	if err := searchQuery.
		Preload("Creator").
		Preload("Team").
		Order("usage_count DESC, created_at DESC").
		Limit(limit).
		Find(&themes).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to search themes",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"themes":  themes,
		"count":   len(themes),
	})
}

// GetAllTags retrieves all unique tags across themes
// GET /api/themes/tags
func GetAllTags(c *fiber.Ctx) error {
	teamID := c.Query("team_id", "")
	
	db := database.GetDB()
	query := db.Model(&models.TeamTheme{}).
		Select("DISTINCT unnest(tags) as tag").
		Where("is_active = ?", true)

	if teamID != "" {
		if tid, err := strconv.ParseUint(teamID, 10, 32); err == nil {
			query = query.Where("team_id = ?", uint(tid))
		}
	}

	var tags []string
	if err := query.Pluck("tag", &tags).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve tags",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"tags":    tags,
		"count":   len(tags),
	})
}

// CloneTheme clones a public theme to user's team
// POST /api/themes/:id/clone
func CloneTheme(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	themeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid theme ID",
		})
	}

	var req struct {
		TargetTeamID uint `json:"target_team_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Verify user is admin of target team
	if !teamService.IsTeamAdmin(userID, req.TargetTeamID) {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Not authorized to add themes to this team",
		})
	}

	db := database.GetDB()
	var originalTheme models.TeamTheme
	if err := db.First(&originalTheme, uint(themeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Theme not found",
		})
	}

	// Verify theme is public or user is team member
	if !originalTheme.IsPublic && !teamService.IsTeamMember(userID, originalTheme.TeamID) {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Theme is not public",
		})
	}

	// Create cloned theme
	clonedTheme := &models.TeamTheme{
		TeamID:      req.TargetTeamID,
		Name:        originalTheme.Name + " (Cloned)",
		Description: originalTheme.Description,
		VerseFile:   originalTheme.VerseFile,
		IsPublic:    false,
		Tags:        originalTheme.Tags,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		IsActive:    true,
	}

	if err := db.Create(clonedTheme).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to clone theme",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "Theme cloned successfully",
		"theme":   clonedTheme,
	})
}

// handlers/leaderboard.go
package handlers

import (
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/fiber/v2"
)

// GetLeaderboard returns the global leaderboard
func GetLeaderboard(c *fiber.Ctx) error {
	category := c.Query("category", "level")
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	if limit > 100 {
		limit = 100
	}

	db := database.GetDB()
	var users []models.User

	var orderBy string
	switch category {
	case "level":
		orderBy = "level DESC, xp DESC"
	case "wins":
		orderBy = "wins DESC"
	case "streak":
		orderBy = "best_streak DESC"
	case "accuracy":
		orderBy = "perfect_games DESC, wins DESC"
	default:
		orderBy = "level DESC, xp DESC"
	}

	if err := db.Where("is_guest = ?", false).
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch leaderboard",
		})
	}

	// Remove sensitive data
	for i := range users {
		users[i].Password = ""
		users[i].Email = nil
	}

	var total int64
	db.Model(&models.User{}).Where("is_guest = ?", false).Count(&total)

	return c.JSON(fiber.Map{
		"success":  true,
		"users":    users,
		"category": category,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetSeasonLeaderboard returns the seasonal leaderboard
func GetSeasonLeaderboard(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	if limit > 100 {
		limit = 100
	}

	db := database.GetDB()
	
	type LeaderboardEntry struct {
		UserID       uint    `json:"user_id"`
		Username     string  `json:"username"`
		Avatar       string  `json:"avatar"`
		Level        int     `json:"level"`
		TotalGames   int     `json:"total_games"`
		Wins         int     `json:"wins"`
		WinRate      float64 `json:"win_rate"`
		CurrentStreak int    `json:"current_streak"`
	}

	var entries []LeaderboardEntry

	db.Raw(`
		SELECT 
			id as user_id,
			username,
			avatar,
			level,
			total_games,
			wins,
			CASE WHEN total_games > 0 THEN (CAST(wins AS FLOAT) / total_games * 100) ELSE 0 END as win_rate,
			current_streak
		FROM users
		WHERE is_guest = false
		ORDER BY wins DESC, level DESC
		LIMIT ? OFFSET ?
	`, limit, offset).Scan(&entries)

	var total int64
	db.Model(&models.User{}).Where("is_guest = ?", false).Count(&total)

	return c.JSON(fiber.Map{
		"success": true,
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetUserRank returns a user's rank in the leaderboard
func GetUserRank(c *fiber.Ctx) error {
	userID := c.Params("id")
	category := c.Query("category", "level")

	db := database.GetDB()
	var user models.User

	if err := db.Where("id = ? OR username = ?", userID, userID).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var rank int64
	var query string

	switch category {
	case "level":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND (level > ? OR (level = ? AND xp > ?))"
		db.Raw(query, user.Level, user.Level, user.XP).Scan(&rank)
	case "wins":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND wins > ?"
		db.Raw(query, user.Wins).Scan(&rank)
	case "streak":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND best_streak > ?"
		db.Raw(query, user.BestStreak).Scan(&rank)
	case "accuracy":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND (perfect_games > ? OR (perfect_games = ? AND wins > ?))"
		db.Raw(query, user.PerfectGames, user.PerfectGames, user.Wins).Scan(&rank)
	default:
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND (level > ? OR (level = ? AND xp > ?))"
		db.Raw(query, user.Level, user.Level, user.XP).Scan(&rank)
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"user_id":  user.ID,
		"username": user.Username,
		"rank":     rank,
		"category": category,
	})
}

// GetLeaderboardAroundUser returns leaderboard entries around a specific user
func GetLeaderboardAroundUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	category := c.Query("category", "level")
	context := c.QueryInt("context", 5) // Number of users above and below

	if context > 20 {
		context = 20
	}

	db := database.GetDB()
	var user models.User

	if err := db.Where("id = ? OR username = ?", userID, userID).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var users []models.User
	var orderBy string

	switch category {
	case "level":
		orderBy = "level DESC, xp DESC"
	case "wins":
		orderBy = "wins DESC"
	case "streak":
		orderBy = "best_streak DESC"
	case "accuracy":
		orderBy = "perfect_games DESC, wins DESC"
	default:
		orderBy = "level DESC, xp DESC"
	}

	// Get users around the target user
	db.Raw(`
		WITH ranked_users AS (
			SELECT *, ROW_NUMBER() OVER (ORDER BY `+orderBy+`) as rank
			FROM users
			WHERE is_guest = false
		),
		target_rank AS (
			SELECT rank FROM ranked_users WHERE id = ?
		)
		SELECT * FROM ranked_users
		WHERE rank BETWEEN (SELECT rank FROM target_rank) - ? AND (SELECT rank FROM target_rank) + ?
		ORDER BY rank
	`, user.ID, context, context).Scan(&users)

	// Remove sensitive data
	for i := range users {
		users[i].Password = ""
		users[i].Email = nil
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"users":      users,
		"target_user": user.ID,
		"category":   category,
		"context":    context,
	})
}

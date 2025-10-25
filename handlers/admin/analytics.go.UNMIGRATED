// handlers/admin/analytics.go
package admin

import (
	"ubible/database"
	"ubible/models"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetAnalytics returns comprehensive system analytics
func GetAnalytics(c *fiber.Ctx) error {
	db := database.GetDB()
	
	// Get time range from query params (default: last 30 days)
	daysStr := c.Query("days", "30")
	var days int
	if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil {
		days = 30
	}
	
	startDate := time.Now().AddDate(0, 0, -days)

	// User Statistics
	var totalUsers int64
	db.Model(&models.User{}).Count(&totalUsers)

	var activeUsers int64
	db.Model(&models.User{}).
		Where("last_login >= ?", startDate).
		Count(&activeUsers)

	var newUsers int64
	db.Model(&models.User{}).
		Where("created_at >= ?", startDate).
		Count(&newUsers)

	var guestUsers int64
	db.Model(&models.User{}).
		Where("is_guest = ?", true).
		Count(&guestUsers)

	var premiumUsers int64
	db.Model(&models.User{}).
		Where("is_guest = ? AND level >= ?", false, 10).
		Count(&premiumUsers)

	// Game Statistics
	var totalGames int64
	db.Model(&models.Attempt{}).Count(&totalGames)

	var recentGames int64
	db.Model(&models.Attempt{}).
		Where("created_at >= ?", startDate).
		Count(&recentGames)

	var perfectGames int64
	db.Model(&models.Attempt{}).
		Where("is_perfect = ?", true).
		Count(&perfectGames)

	// Calculate average score
	var avgScore float64
	db.Model(&models.Attempt{}).
		Select("AVG(score)").
		Scan(&avgScore)

	// Calculate average accuracy
	var avgAccuracy float64
	db.Model(&models.Attempt{}).
		Select("AVG(CAST(correct_answers AS FLOAT) / CAST(total_questions AS FLOAT) * 100)").
		Scan(&avgAccuracy)

	// Theme Statistics
	type ThemeStats struct {
		ThemeID   uint
		ThemeName string
		PlayCount int64
		AvgScore  float64
	}

	var themeStats []ThemeStats
	db.Table("attempts").
		Select("attempts.theme_id, themes.name as theme_name, COUNT(*) as play_count, AVG(attempts.score) as avg_score").
		Joins("LEFT JOIN themes ON themes.id = attempts.theme_id").
		Where("attempts.created_at >= ?", startDate).
		Group("attempts.theme_id, themes.name").
		Order("play_count DESC").
		Limit(10).
		Scan(&themeStats)

	// Achievement Statistics
	var totalAchievements int64
	db.Model(&models.Achievement{}).Count(&totalAchievements)

	var unlockedAchievements int64
	db.Model(&models.UserAchievement{}).Count(&unlockedAchievements)

	var recentUnlocks int64
	db.Model(&models.UserAchievement{}).
		Where("unlocked_at >= ?", startDate).
		Count(&recentUnlocks)

	// Most unlocked achievements
	type AchievementUnlockStats struct {
		AchievementID   uint
		AchievementName string
		UnlockCount     int64
	}

	var popularAchievements []AchievementUnlockStats
	db.Table("user_achievements").
		Select("user_achievements.achievement_id, achievements.name as achievement_name, COUNT(*) as unlock_count").
		Joins("LEFT JOIN achievements ON achievements.id = user_achievements.achievement_id").
		Group("user_achievements.achievement_id, achievements.name").
		Order("unlock_count DESC").
		Limit(10).
		Scan(&popularAchievements)

	// Revenue Statistics (Faith Points)
	var totalFPEarned int64
	db.Model(&models.Attempt{}).
		Select("SUM(fp_earned)").
		Scan(&totalFPEarned)

	var totalFPSpent int64
	db.Model(&models.User{}).
		Select("SUM(power_up_5050 + power_up_time_freeze + power_up_hint + power_up_skip + power_up_double)").
		Scan(&totalFPSpent)

	// Level Distribution
	type LevelDistribution struct {
		LevelRange string
		UserCount  int64
	}

	var levelDist []LevelDistribution
	db.Raw(`
		SELECT 
			CASE 
				WHEN level BETWEEN 1 AND 10 THEN '1-10'
				WHEN level BETWEEN 11 AND 25 THEN '11-25'
				WHEN level BETWEEN 26 AND 50 THEN '26-50'
				WHEN level BETWEEN 51 AND 75 THEN '51-75'
				WHEN level BETWEEN 76 AND 100 THEN '76-100'
				ELSE '100+'
			END as level_range,
			COUNT(*) as user_count
		FROM users
		WHERE is_guest = false
		GROUP BY level_range
		ORDER BY level_range
	`).Scan(&levelDist)

	// Daily Activity (last 30 days)
	type DailyActivity struct {
		Date       string
		GameCount  int64
		UserCount  int64
		NewUsers   int64
	}

	var dailyActivity []DailyActivity
	db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as game_count,
			COUNT(DISTINCT user_id) as user_count
		FROM attempts
		WHERE created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date DESC
		LIMIT 30
	`, startDate).Scan(&dailyActivity)

	// Get new users per day
	var newUsersDaily []struct {
		Date      string
		UserCount int64
	}
	db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as user_count
		FROM users
		WHERE created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, startDate).Scan(&newUsersDaily)

	// Merge new users into daily activity
	newUsersMap := make(map[string]int64)
	for _, nu := range newUsersDaily {
		newUsersMap[nu.Date] = nu.UserCount
	}
	for i := range dailyActivity {
		if count, ok := newUsersMap[dailyActivity[i].Date]; ok {
			dailyActivity[i].NewUsers = count
		}
	}

	// Top Players
	type TopPlayer struct {
		UserID   uint
		Username string
		Level    int
		XP       int
		Wins     int
		Streak   int
	}

	var topPlayers []TopPlayer
	db.Table("users").
		Select("id as user_id, username, level, xp, wins, best_streak as streak").
		Where("is_guest = false").
		Order("level DESC, xp DESC").
		Limit(10).
		Scan(&topPlayers)

	// Retention Metrics
	var dayOneRetention float64
	db.Raw(`
		SELECT 
			COUNT(DISTINCT u2.id) * 100.0 / COUNT(DISTINCT u1.id) as retention
		FROM users u1
		LEFT JOIN users u2 ON u1.id = u2.id 
			AND u2.last_login >= DATE(u1.created_at, '+1 day')
			AND u2.last_login < DATE(u1.created_at, '+2 days')
		WHERE u1.created_at >= ?
	`, startDate).Scan(&dayOneRetention)

	var daySevenRetention float64
	db.Raw(`
		SELECT 
			COUNT(DISTINCT u2.id) * 100.0 / COUNT(DISTINCT u1.id) as retention
		FROM users u1
		LEFT JOIN users u2 ON u1.id = u2.id 
			AND u2.last_login >= DATE(u1.created_at, '+7 days')
		WHERE u1.created_at >= ?
	`, startDate.AddDate(0, 0, -7)).Scan(&daySevenRetention)

	// Power-up Usage
	type PowerUpUsage struct {
		PowerUpType string
		TotalOwned  int64
		AvgPerUser  float64
	}

	var powerUpStats []PowerUpUsage
	db.Raw(`
		SELECT 
			'50/50' as power_up_type,
			SUM(power_up_5050) as total_owned,
			AVG(power_up_5050) as avg_per_user
		FROM users WHERE is_guest = false
		UNION ALL
		SELECT 
			'Time Freeze' as power_up_type,
			SUM(power_up_time_freeze) as total_owned,
			AVG(power_up_time_freeze) as avg_per_user
		FROM users WHERE is_guest = false
		UNION ALL
		SELECT 
			'Hint' as power_up_type,
			SUM(power_up_hint) as total_owned,
			AVG(power_up_hint) as avg_per_user
		FROM users WHERE is_guest = false
		UNION ALL
		SELECT 
			'Skip' as power_up_type,
			SUM(power_up_skip) as total_owned,
			AVG(power_up_skip) as avg_per_user
		FROM users WHERE is_guest = false
		UNION ALL
		SELECT 
			'Double Points' as power_up_type,
			SUM(power_up_double) as total_owned,
			AVG(power_up_double) as avg_per_user
		FROM users WHERE is_guest = false
	`).Scan(&powerUpStats)

	// Compile response
	analytics := fiber.Map{
		"period": fiber.Map{
			"days":       days,
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   time.Now().Format("2006-01-02"),
		},
		"users": fiber.Map{
			"total":         totalUsers,
			"active":        activeUsers,
			"new":           newUsers,
			"guests":        guestUsers,
			"premium":       premiumUsers,
			"level_distribution": levelDist,
		},
		"games": fiber.Map{
			"total":         totalGames,
			"recent":        recentGames,
			"perfect":       perfectGames,
			"avg_score":     avgScore,
			"avg_accuracy":  avgAccuracy,
		},
		"themes": fiber.Map{
			"popular": themeStats,
		},
		"achievements": fiber.Map{
			"total":           totalAchievements,
			"unlocked":        unlockedAchievements,
			"recent_unlocks":  recentUnlocks,
			"popular":         popularAchievements,
		},
		"economy": fiber.Map{
			"total_fp_earned": totalFPEarned,
			"total_fp_spent":  totalFPSpent,
			"powerup_usage":   powerUpStats,
		},
		"activity": fiber.Map{
			"daily": dailyActivity,
		},
		"leaderboard": fiber.Map{
			"top_players": topPlayers,
		},
		"retention": fiber.Map{
			"day_1":  dayOneRetention,
			"day_7":  daySevenRetention,
		},
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"analytics": analytics,
		"generated_at": time.Now().Unix(),
	})
}

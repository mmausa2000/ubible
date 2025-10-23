// handlers/progression.go
package handlers

import (
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type RecordGameRequest struct {
	ThemeID        uint   `json:"theme_id"`
	CorrectAnswers int    `json:"correct_answers"`
	TotalQuestions int    `json:"total_questions"`
	Score          int    `json:"score"`
	TimeElapsed    int    `json:"time_elapsed"`
	FastAnswers    int    `json:"fast_answers"`
	IsPerfect      bool   `json:"is_perfect"`
	Won            bool   `json:"won"`
	IsMultiplayer  bool   `json:"is_multiplayer"`
	OpponentID     *uint  `json:"opponent_id"`
	Difficulty     string `json:"difficulty"`
}

type AwardXPRequest struct {
	Amount int    `json:"amount"`
	Reason string `json:"reason"`
}

func RecordGame(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	var req RecordGameRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	db := database.GetDB()
	var user models.User
	
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	xp := calculateXP(req)
	faithPoints := calculateFaithPoints(req)

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	user.TotalGames++
	if req.Won {
		user.Wins++
		user.CurrentStreak++
		if user.CurrentStreak > user.BestStreak {
			user.BestStreak = user.CurrentStreak
		}
	} else {
		user.Losses++
		user.CurrentStreak = 0
	}

	if req.IsPerfect {
		user.PerfectGames++
	}

	oldLevel := user.Level
	user.XP += xp
	user.FaithPoints += faithPoints

	for {
		xpNeeded := calculateXPForLevel(user.Level + 1)
		if user.XP >= xpNeeded {
			user.Level++
			user.XP -= xpNeeded
			levelReward := 50 + (user.Level * 10)
			user.FaithPoints += levelReward
		} else {
			break
		}
	}

	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update user"})
	}

	attempt := models.Attempt{
		UserID:         userID,
		ThemeID:        req.ThemeID,
		Score:          req.Score,
		CorrectAnswers: req.CorrectAnswers,
		TotalQuestions: req.TotalQuestions,
		TimeElapsed:    req.TimeElapsed,
		IsPerfect:      req.IsPerfect,
		Difficulty:     req.Difficulty,
		XPEarned:       xp,
		FPEarned:       faithPoints,
	}

	if err := tx.Create(&attempt).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Failed to record attempt"})
	}

	newAchievements := checkAchievements(&user, req, tx)

	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to commit transaction"})
	}

	leveledUp := user.Level > oldLevel
	response := fiber.Map{
		"success":            true,
		"xp_earned":          xp,
		"faith_points":       faithPoints,
		"new_level":          user.Level,
		"leveled_up":         leveledUp,
		"current_xp":         user.XP,
		"xp_to_next_level":   calculateXPForLevel(user.Level + 1),
		"total_faith_points": user.FaithPoints,
		"current_streak":     user.CurrentStreak,
		"best_streak":        user.BestStreak,
		"new_achievements":   newAchievements,
	}

	if leveledUp {
		levelReward := 50 + (user.Level * 10)
		response["level_reward"] = levelReward
	}

	return c.JSON(response)
}

func AwardXP(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	var req AwardXPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "XP amount must be positive"})
	}

	db := database.GetDB()
	var user models.User
	
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	oldLevel := user.Level
	user.XP += req.Amount

	levelsGained := 0
	for {
		xpNeeded := calculateXPForLevel(user.Level + 1)
		if user.XP >= xpNeeded {
			user.Level++
			user.XP -= xpNeeded
			levelsGained++
			levelReward := 50 + (user.Level * 10)
			user.FaithPoints += levelReward
		} else {
			break
		}
	}

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update user"})
	}

	return c.JSON(fiber.Map{
		"success":          true,
		"xp_awarded":       req.Amount,
		"new_level":        user.Level,
		"leveled_up":       user.Level > oldLevel,
		"levels_gained":    levelsGained,
		"current_xp":       user.XP,
		"xp_to_next_level": calculateXPForLevel(user.Level + 1),
		"reason":           req.Reason,
	})
}

func GetProgression(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	db := database.GetDB()
	var user models.User
	
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	xpToNext := calculateXPForLevel(user.Level + 1)
	progress := (float64(user.XP) / float64(xpToNext)) * 100

	return c.JSON(fiber.Map{
		"success":          true,
		"level":            user.Level,
		"xp":               user.XP,
		"xp_to_next_level": xpToNext,
		"progress_percent": progress,
		"faith_points":     user.FaithPoints,
		"total_games":      user.TotalGames,
		"wins":             user.Wins,
		"losses":           user.Losses,
		"current_streak":   user.CurrentStreak,
		"best_streak":      user.BestStreak,
		"perfect_games":    user.PerfectGames,
	})
}

func GetUserAchievements(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	db := database.GetDB()
	
	var unlocked []models.UserAchievement
	if err := db.Preload("Achievement").Where("user_id = ?", userID).Order("unlocked_at DESC").Find(&unlocked).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch achievements"})
	}

	var allAchievements []models.Achievement
	if err := db.Find(&allAchievements).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch all achievements"})
	}

	unlockedMap := make(map[uint]models.UserAchievement)
	for _, ua := range unlocked {
		unlockedMap[ua.AchievementID] = ua
	}

	achievements := make([]fiber.Map, 0, len(allAchievements))
	for _, achievement := range allAchievements {
		achData := fiber.Map{
			"id":          achievement.ID,
			"name":        achievement.Name,
			"description": achievement.Description,
			"category":    achievement.Category,
			"tier":        achievement.Tier,
			"icon":        achievement.Icon,
			"xp_reward":   achievement.XPReward,
			"fp_reward":   achievement.FPReward,
			"unlocked":    false,
		}

		if ua, ok := unlockedMap[achievement.ID]; ok {
			achData["unlocked"] = true
			achData["unlocked_at"] = ua.UnlockedAt
		}

		achievements = append(achievements, achData)
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"achievements": achievements,
		"total":        len(allAchievements),
		"unlocked":     len(unlocked),
	})
}

func calculateXP(req RecordGameRequest) int {
	xp := req.CorrectAnswers * 10
	xp += req.FastAnswers * 5
	
	if req.IsPerfect {
		xp += 50
	}
	
	if req.Won {
		xp += 20
	}
	
	if req.IsMultiplayer {
		xp += 30
	}
	
	switch req.Difficulty {
	case "hard":
		xp = int(float64(xp) * 1.5)
	case "expert":
		xp = int(float64(xp) * 2.0)
	}
	
	return xp
}

func calculateFaithPoints(req RecordGameRequest) int {
	fp := req.CorrectAnswers * 2
	
	if req.IsPerfect {
		fp += 10
	}
	
	if req.Won {
		fp += 5
	}
	
	return fp
}

func calculateXPForLevel(level int) int {
	return int(100 * math.Pow(float64(level), 1.5))
}

func checkAchievements(user *models.User, req RecordGameRequest, tx *gorm.DB) []models.Achievement {
	newAchievements := []models.Achievement{}

	var allAchievements []models.Achievement
	tx.Find(&allAchievements)

	var unlockedIDs []uint
	tx.Model(&models.UserAchievement{}).Where("user_id = ?", user.ID).Pluck("achievement_id", &unlockedIDs)

	unlockedMap := make(map[uint]bool)
	for _, id := range unlockedIDs {
		unlockedMap[id] = true
	}

	for _, achievement := range allAchievements {
		if unlockedMap[achievement.ID] {
			continue
		}

		unlocked := false

		switch achievement.Category {
		case "Speed":
			unlocked = checkSpeedAchievement(achievement, user, req)
		case "Accuracy":
			unlocked = checkAccuracyAchievement(achievement, user, req)
		case "Streak":
			unlocked = checkStreakAchievement(achievement, user)
		case "Social":
			unlocked = checkSocialAchievement(achievement, user, tx)
		case "Theme":
			unlocked = checkThemeAchievement(achievement, user, req, tx)
		case "Special":
			unlocked = checkSpecialAchievement(achievement, user, req)
		}

		if unlocked {
			userAchievement := models.UserAchievement{
				UserID:        user.ID,
				AchievementID: achievement.ID,
				UnlockedAt:    time.Now(),
			}
			tx.Create(&userAchievement)

			user.XP += achievement.XPReward
			user.FaithPoints += achievement.FPReward

			if achievement.PowerUpReward != "" {
				awardPowerUp(user, achievement.PowerUpReward, achievement.PowerUpQuantity)
			}

			newAchievements = append(newAchievements, achievement)
		}
	}

	if len(newAchievements) > 0 {
		tx.Save(user)
	}

	return newAchievements
}

func checkSpeedAchievement(achievement models.Achievement, user *models.User, req RecordGameRequest) bool {
	switch achievement.Name {
	case "Lightning Fast":
		return req.FastAnswers >= 5
	case "Speed Demon":
		return req.FastAnswers >= 10
	case "Flash":
		return req.TimeElapsed < 60 && req.TotalQuestions >= 10
	}
	return false
}

func checkAccuracyAchievement(achievement models.Achievement, user *models.User, req RecordGameRequest) bool {
	accuracy := float64(req.CorrectAnswers) / float64(req.TotalQuestions) * 100

	switch achievement.Name {
	case "First Perfect":
		return req.IsPerfect
	case "Perfect Scholar":
		return user.PerfectGames >= 10
	case "Master Scholar":
		return user.PerfectGames >= 50
	case "Accuracy Expert":
		return accuracy >= 90 && user.TotalGames >= 20
	}
	return false
}

func checkStreakAchievement(achievement models.Achievement, user *models.User) bool {
	switch achievement.Name {
	case "Hot Streak":
		return user.CurrentStreak >= 3
	case "On Fire":
		return user.CurrentStreak >= 5
	case "Unstoppable":
		return user.CurrentStreak >= 10
	case "Legendary Streak":
		return user.BestStreak >= 20
	}
	return false
}

func checkSocialAchievement(achievement models.Achievement, user *models.User, tx *gorm.DB) bool {
	var friendCount int64
	tx.Model(&models.Friend{}).Where("user_id = ? OR friend_id = ?", user.ID, user.ID).Count(&friendCount)

	switch achievement.Name {
	case "Social Butterfly":
		return friendCount >= 5
	case "Popular":
		return friendCount >= 20
	case "First Victory":
		return user.Wins >= 1
	case "Conqueror":
		return user.Wins >= 100
	}
	return false
}

func checkThemeAchievement(achievement models.Achievement, user *models.User, req RecordGameRequest, tx *gorm.DB) bool {
	var themeAttempts int64
	tx.Model(&models.Attempt{}).Where("user_id = ? AND theme_id = ?", user.ID, req.ThemeID).Count(&themeAttempts)

	switch achievement.Name {
	case "Theme Explorer":
		var uniqueThemes int64
		tx.Model(&models.Attempt{}).Where("user_id = ?", user.ID).Distinct("theme_id").Count(&uniqueThemes)
		return uniqueThemes >= 5
	case "Theme Master":
		return themeAttempts >= 50 && req.IsPerfect
	}
	return false
}

func checkSpecialAchievement(achievement models.Achievement, user *models.User, req RecordGameRequest) bool {
	switch achievement.Name {
	case "First Steps":
		return user.TotalGames >= 1
	case "Dedicated":
		return user.TotalGames >= 50
	case "Veteran":
		return user.TotalGames >= 100
	case "Level 10":
		return user.Level >= 10
	case "Level 50":
		return user.Level >= 50
	case "Centurion":
		return user.Level >= 100
	}
	return false
}

func awardPowerUp(user *models.User, powerUpType string, quantity int) {
	switch powerUpType {
	case "5050":
		user.PowerUp5050 += quantity
	case "timefreeze":
		user.PowerUpTimeFreeze += quantity
	case "hint":
		user.PowerUpHint += quantity
	case "skip":
		user.PowerUpSkip += quantity
	case "double":
		user.PowerUpDouble += quantity
	}
}

func UsePowerUp(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	var req struct {
		PowerUpType string `json:"powerup_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	db := database.GetDB()
	var user models.User
	
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	hasPowerUp := false
	switch req.PowerUpType {
	case "5050":
		hasPowerUp = user.PowerUp5050 > 0
		if hasPowerUp {
			user.PowerUp5050--
		}
	case "timefreeze":
		hasPowerUp = user.PowerUpTimeFreeze > 0
		if hasPowerUp {
			user.PowerUpTimeFreeze--
		}
	case "hint":
		hasPowerUp = user.PowerUpHint > 0
		if hasPowerUp {
			user.PowerUpHint--
		}
	case "skip":
		hasPowerUp = user.PowerUpSkip > 0
		if hasPowerUp {
			user.PowerUpSkip--
		}
	case "double":
		hasPowerUp = user.PowerUpDouble > 0
		if hasPowerUp {
			user.PowerUpDouble--
		}
	default:
		return c.Status(400).JSON(fiber.Map{"error": "Invalid power-up type"})
	}

	if !hasPowerUp {
		return c.Status(400).JSON(fiber.Map{"error": "You don't have this power-up"})
	}

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to use power-up"})
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"powerup_type": req.PowerUpType,
		"remaining":    getPowerUpCount(&user, req.PowerUpType),
	})
}

func PurchasePowerUp(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	var req struct {
		PowerUpType string `json:"powerup_type"`
		Quantity    int    `json:"quantity"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Quantity <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Quantity must be positive"})
	}

	db := database.GetDB()
	var user models.User
	
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	unitCost := getPowerUpCost(req.PowerUpType)
	totalCost := unitCost * req.Quantity

	if user.FaithPoints < totalCost {
		return c.Status(400).JSON(fiber.Map{
			"error":     "Insufficient Faith Points",
			"required":  totalCost,
			"available": user.FaithPoints,
		})
	}

	user.FaithPoints -= totalCost
	
	switch req.PowerUpType {
	case "5050":
		user.PowerUp5050 += req.Quantity
	case "timefreeze":
		user.PowerUpTimeFreeze += req.Quantity
	case "hint":
		user.PowerUpHint += req.Quantity
	case "skip":
		user.PowerUpSkip += req.Quantity
	case "double":
		user.PowerUpDouble += req.Quantity
	default:
		return c.Status(400).JSON(fiber.Map{"error": "Invalid power-up type"})
	}

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to purchase power-up"})
	}

	return c.JSON(fiber.Map{
		"success":                true,
		"powerup_type":           req.PowerUpType,
		"quantity":               req.Quantity,
		"cost":                   totalCost,
		"remaining_faith_points": user.FaithPoints,
		"powerup_count":          getPowerUpCount(&user, req.PowerUpType),
	})
}

func GetPowerUpInventory(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	db := database.GetDB()
	var user models.User
	
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	inventory := fiber.Map{
		"5050": fiber.Map{
			"count":       user.PowerUp5050,
			"cost":        getPowerUpCost("5050"),
			"name":        "50/50",
			"description": "Eliminate 2 wrong answers",
		},
		"timefreeze": fiber.Map{
			"count":       user.PowerUpTimeFreeze,
			"cost":        getPowerUpCost("timefreeze"),
			"name":        "Time Freeze",
			"description": "Pause timer for 10 seconds",
		},
		"hint": fiber.Map{
			"count":       user.PowerUpHint,
			"cost":        getPowerUpCost("hint"),
			"name":        "Hint",
			"description": "Show first letter of answer",
		},
		"skip": fiber.Map{
			"count":       user.PowerUpSkip,
			"cost":        getPowerUpCost("skip"),
			"name":        "Skip",
			"description": "Move to next question",
		},
		"double": fiber.Map{
			"count":       user.PowerUpDouble,
			"cost":        getPowerUpCost("double"),
			"name":        "Double Points",
			"description": "2x score for next question",
		},
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"inventory":    inventory,
		"faith_points": user.FaithPoints,
	})
}

func getPowerUpCost(powerUpType string) int {
	costs := map[string]int{
		"5050":       50,
		"timefreeze": 75,
		"hint":       40,
		"skip":       30,
		"double":     100,
	}
	
	if cost, ok := costs[powerUpType]; ok {
		return cost
	}
	return 50
}

func getPowerUpCount(user *models.User, powerUpType string) int {
	switch powerUpType {
	case "5050":
		return user.PowerUp5050
	case "timefreeze":
		return user.PowerUpTimeFreeze
	case "hint":
		return user.PowerUpHint
	case "skip":
		return user.PowerUpSkip
	case "double":
		return user.PowerUpDouble
	default:
		return 0
	}
}

func AwardFaithPoints(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func GetAchievements(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "achievements": []interface{}{}})
}

func CheckAchievements(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func GetUserProgress(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func GetLevelRewards(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "rewards": []interface{}{}})
}

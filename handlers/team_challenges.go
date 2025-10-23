// handlers/team_challenges.go - Team Challenge System Handlers
package handlers

import (
	"ubible/database"
	"ubible/models"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ================== CHALLENGE CRUD ENDPOINTS ==================

// CreateChallenge creates a new team challenge
// POST /api/teams/:id/challenges
func CreateChallenge(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	// Verify user is admin or owner
	if !teamService.IsTeamAdmin(userID, uint(teamID)) {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Only team admins can create challenges",
		})
	}

	var req struct {
		Name            string    `json:"name"`
		Description     string    `json:"description"`
		ThemeID         uint      `json:"theme_id"`
		QuestionCount   int       `json:"question_count"`
		TimeLimit       int       `json:"time_limit"`
		StartDate       time.Time `json:"start_date"`
		EndDate         time.Time `json:"end_date"`
		MinParticipants int       `json:"min_participants"`
		MaxParticipants int       `json:"max_participants"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge name is required",
		})
	}

	if req.QuestionCount <= 0 {
		req.QuestionCount = 10
	}

	if req.TimeLimit <= 0 {
		req.TimeLimit = 30
	}

	// Create challenge
	challenge := &models.Challenge{
		TeamID:          uint(teamID),
		Name:            req.Name,
		Description:     req.Description,
		ThemeID:         req.ThemeID,
		QuestionCount:   req.QuestionCount,
		TimeLimit:       req.TimeLimit,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		MinParticipants: req.MinParticipants,
		MaxParticipants: req.MaxParticipants,
		Status:          models.ChallengeStatusPending,
		CreatedBy:       userID,
		CreatedAt:       time.Now(),
	}

	db := database.GetDB()
	if err := db.Create(challenge).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create challenge",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"success":   true,
		"message":   "Challenge created successfully",
		"challenge": challenge,
	})
}

// GetTeamChallenges retrieves all challenges for a team
// GET /api/teams/:id/challenges
func GetTeamChallenges(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	status := c.Query("status", "") // Filter by status

	db := database.GetDB()
	query := db.Where("team_id = ?", uint(teamID))

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var challenges []models.Challenge
	if err := query.Preload("Theme").
		Preload("Participants").
		Order("created_at DESC").
		Find(&challenges).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve challenges",
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"challenges": challenges,
		"count":      len(challenges),
	})
}

// GetChallenge retrieves a specific challenge
// GET /api/challenges/:id
func GetChallenge(c *fiber.Ctx) error {
	challengeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid challenge ID",
		})
	}

	db := database.GetDB()
	var challenge models.Challenge
	if err := db.Where("id = ?", uint(challengeID)).
		Preload("Theme").
		Preload("Participants").
		Preload("Participants.User").
		First(&challenge).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge not found",
		})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"challenge": challenge,
	})
}

// UpdateChallenge updates a challenge
// PUT /api/challenges/:id
func UpdateChallenge(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	challengeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid challenge ID",
		})
	}

	db := database.GetDB()
	var challenge models.Challenge
	if err := db.First(&challenge, uint(challengeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge not found",
		})
	}

	// Verify user is admin or creator
	if !teamService.IsTeamAdmin(userID, challenge.TeamID) && challenge.CreatedBy != userID {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Not authorized to update this challenge",
		})
	}

	// Can't update active or completed challenges
	if challenge.Status != models.ChallengeStatusPending {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Can only update pending challenges",
		})
	}

	var req struct {
		Name          string    `json:"name"`
		Description   string    `json:"description"`
		QuestionCount int       `json:"question_count"`
		TimeLimit     int       `json:"time_limit"`
		StartDate     time.Time `json:"start_date"`
		EndDate       time.Time `json:"end_date"`
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
	if req.QuestionCount > 0 {
		updates["question_count"] = req.QuestionCount
	}
	if req.TimeLimit > 0 {
		updates["time_limit"] = req.TimeLimit
	}
	if !req.StartDate.IsZero() {
		updates["start_date"] = req.StartDate
	}
	if !req.EndDate.IsZero() {
		updates["end_date"] = req.EndDate
	}

	if err := db.Model(&challenge).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update challenge",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Challenge updated successfully",
	})
}

// DeleteChallenge deletes a challenge
// DELETE /api/challenges/:id
func DeleteChallenge(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	challengeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid challenge ID",
		})
	}

	db := database.GetDB()
	var challenge models.Challenge
	if err := db.First(&challenge, uint(challengeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge not found",
		})
	}

	// Verify user is admin or creator
	if !teamService.IsTeamAdmin(userID, challenge.TeamID) && challenge.CreatedBy != userID {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Not authorized to delete this challenge",
		})
	}

	// Can't delete active challenges
	if challenge.Status == models.ChallengeStatusActive {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Cannot delete active challenges",
		})
	}

	if err := db.Delete(&challenge).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete challenge",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Challenge deleted successfully",
	})
}

// ================== CHALLENGE PARTICIPATION ENDPOINTS ==================

// JoinChallenge allows a user to join a challenge
// POST /api/challenges/:id/join
func JoinChallenge(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	challengeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid challenge ID",
		})
	}

	db := database.GetDB()
	var challenge models.Challenge
	if err := db.Preload("Participants").First(&challenge, uint(challengeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge not found",
		})
	}

	// Verify user is team member
	if !teamService.IsTeamMember(userID, challenge.TeamID) {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Must be a team member to join challenge",
		})
	}

	// Check if already joined
	for _, p := range challenge.Participants {
		if p.UserID == userID {
			return c.Status(400).JSON(fiber.Map{
				"success": false,
				"error":   "Already joined this challenge",
			})
		}
	}

	// Check max participants
	if challenge.MaxParticipants > 0 && len(challenge.Participants) >= challenge.MaxParticipants {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge is full",
		})
	}

	// Check if challenge is open
	if challenge.Status != models.ChallengeStatusPending && challenge.Status != models.ChallengeStatusActive {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge is not accepting participants",
		})
	}

	// Add participant
	participant := &models.ChallengeParticipant{
		ChallengeID: uint(challengeID),
		UserID:      userID,
		JoinedAt:    time.Now(),
		Status:      "joined",
	}

	if err := db.Create(participant).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to join challenge",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Successfully joined challenge",
	})
}

// LeaveChallenge allows a user to leave a challenge
// POST /api/challenges/:id/leave
func LeaveChallenge(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	challengeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid challenge ID",
		})
	}

	db := database.GetDB()
	var challenge models.Challenge
	if err := db.First(&challenge, uint(challengeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge not found",
		})
	}

	// Can't leave active or completed challenges
	if challenge.Status != models.ChallengeStatusPending {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Cannot leave challenge that has started",
		})
	}

	// Remove participant
	if err := db.Where("challenge_id = ? AND user_id = ?", uint(challengeID), userID).
		Delete(&models.ChallengeParticipant{}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to leave challenge",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Successfully left challenge",
	})
}

// StartChallenge starts a pending challenge
// POST /api/challenges/:id/start
func StartChallenge(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	challengeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid challenge ID",
		})
	}

	db := database.GetDB()
	var challenge models.Challenge
	if err := db.Preload("Participants").First(&challenge, uint(challengeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge not found",
		})
	}

	// Verify user is admin or creator
	if !teamService.IsTeamAdmin(userID, challenge.TeamID) && challenge.CreatedBy != userID {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   "Not authorized to start this challenge",
		})
	}

	// Check status
	if challenge.Status != models.ChallengeStatusPending {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge is not in pending status",
		})
	}

	// Check min participants
	if challenge.MinParticipants > 0 && len(challenge.Participants) < challenge.MinParticipants {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Not enough participants to start challenge",
		})
	}

	// Update status
	updates := map[string]interface{}{
		"status":     models.ChallengeStatusActive,
		"started_at": time.Now(),
	}

	if err := db.Model(&challenge).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to start challenge",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Challenge started successfully",
	})
}

// SubmitChallengeScore submits a user's score for a challenge
// POST /api/challenges/:id/submit
func SubmitChallengeScore(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	challengeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid challenge ID",
		})
	}

	var req struct {
		Score     int `json:"score"`
		TimeSpent int `json:"time_spent"`
		Correct   int `json:"correct"`
		Incorrect int `json:"incorrect"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	db := database.GetDB()
	var challenge models.Challenge
	if err := db.First(&challenge, uint(challengeID)).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge not found",
		})
	}

	// Check if challenge is active
	if challenge.Status != models.ChallengeStatusActive {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Challenge is not active",
		})
	}

	// Update participant score
	updates := map[string]interface{}{
		"score":        req.Score,
		"time_spent":   req.TimeSpent,
		"correct":      req.Correct,
		"incorrect":    req.Incorrect,
		"completed_at": time.Now(),
		"status":       "completed",
	}

	if err := db.Model(&models.ChallengeParticipant{}).
		Where("challenge_id = ? AND user_id = ?", uint(challengeID), userID).
		Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to submit score",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Score submitted successfully",
	})
}

// GetChallengeLeaderboard retrieves challenge leaderboard
// GET /api/challenges/:id/leaderboard
func GetChallengeLeaderboard(c *fiber.Ctx) error {
	challengeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid challenge ID",
		})
	}

	db := database.GetDB()
	var participants []models.ChallengeParticipant
	if err := db.Where("challenge_id = ?", uint(challengeID)).
		Preload("User").
		Order("score DESC, time_spent ASC").
		Find(&participants).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve leaderboard",
		})
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"leaderboard": participants,
		"count":       len(participants),
	})
}

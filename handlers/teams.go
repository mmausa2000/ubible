// handlers/teams.go - Team Portal HTTP Handlers
package handlers

import (
	"log"
	"strconv"
	"ubible/database"
	"ubible/models"
	"ubible/services"

	"github.com/gofiber/fiber/v2"
)

var teamService *services.TeamService

// InitTeamHandlers initializes the team service
func InitTeamHandlers() {
	db := database.GetDB()
	if db == nil {
		panic("Database not initialized before InitTeamHandlers")
	}
	teamService = services.NewTeamService(db)
}

// ================== TEAM CRUD ENDPOINTS ==================

// CreateTeam creates a new team
// POST /api/teams
func CreateTeam(c *fiber.Ctx) error {
	log.Println("📝 CreateTeam endpoint called")

	// Check if teamService is initialized
	if teamService == nil {
		log.Println("❌ Team service is nil!")
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Team service not initialized",
		})
	}
	log.Println("✅ Team service initialized")

	// Get authenticated user ID from context (middleware sets "userId")
	userIDRaw := c.Locals("userId")
	if userIDRaw == nil {
		log.Println("❌ No userId in context - authentication failed")
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized - no user ID",
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
		log.Printf("❌ Invalid userId type: %T (value: %v)\n", userIDRaw, userIDRaw)
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid user ID type",
		})
	}
	log.Printf("✅ User ID from context: %d\n", userID)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Printf("❌ Body parse error: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}
	log.Printf("✅ Request body parsed: name='%s', desc='%s', public=%v\n", req.Name, req.Description, req.IsPublic)

	// Validate name
	if req.Name == "" {
		log.Println("❌ Team name is empty")
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Team name is required",
		})
	}

	// Create team
	log.Printf("🔨 Creating team: name='%s', userID=%d\n", req.Name, userID)
	team, err := teamService.CreateTeam(req.Name, req.Description, req.IsPublic, userID)
	if err != nil {
		log.Printf("❌ Team creation failed: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	log.Printf("✅ Team created successfully! ID=%d, Code=%s\n", team.ID, team.TeamCode)
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "Team created successfully",
		"team":    team,
	})
}

// GetTeam retrieves a team by ID
// GET /api/teams/:id
func GetTeam(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	team, err := teamService.GetTeamByID(uint(teamID))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Team not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"team":    team,
	})
}

// GetUserTeams retrieves all teams for the authenticated user
// GET /api/teams/my-teams
func GetUserTeams(c *fiber.Ctx) error {
	userIDRaw := c.Locals("userId")
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}

	teams, err := teamService.GetUserTeams(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve teams",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"teams":   teams,
		"count":   len(teams),
	})
}

// UpdateTeam updates team information
// PUT /api/teams/:id
func UpdateTeam(c *fiber.Ctx) error {
	userIDRaw := c.Locals("userId")
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	err = teamService.UpdateTeam(uint(teamID), req.Name, req.Description, req.IsPublic, userID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Team updated successfully",
	})
}

// DeleteTeam deletes a team
// DELETE /api/teams/:id
func DeleteTeam(c *fiber.Ctx) error {
	userIDRaw := c.Locals("userId")
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	err = teamService.DeleteTeam(uint(teamID), userID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Team deleted successfully",
	})
}

// ================== TEAM MEMBERSHIP ENDPOINTS ==================

// JoinTeam allows a user to join a team via code
// POST /api/teams/join
func JoinTeam(c *fiber.Ctx) error {
	userIDRaw := c.Locals("userId")
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}

	var req struct {
		TeamCode string `json:"team_code"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.TeamCode == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Team code is required",
		})
	}

	err := teamService.JoinTeam(userID, req.TeamCode)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Get the team details to return
	team, _ := teamService.GetTeamByCode(req.TeamCode)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Successfully joined team",
		"team":    team,
	})
}

// LeaveTeam allows a user to leave a team
// POST /api/teams/:id/leave
func LeaveTeam(c *fiber.Ctx) error {
	userIDRaw := c.Locals("userId")
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	err = teamService.LeaveTeam(userID, uint(teamID))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Successfully left team",
	})
}

// GetTeamMembers retrieves all members of a team
// GET /api/teams/:id/members
func GetTeamMembers(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	members, err := teamService.GetTeamMembers(uint(teamID))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve members",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"members": members,
		"count":   len(members),
	})
}

// RemoveMember removes a member from a team
// DELETE /api/teams/:id/members/:memberId
func RemoveMember(c *fiber.Ctx) error {
	adminIDRaw := c.Locals("userId")
	var adminID uint
	switch v := adminIDRaw.(type) {
	case float64:
		adminID = uint(v)
	case uint:
		adminID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}

	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	memberID, err := strconv.ParseUint(c.Params("memberId"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid member ID",
		})
	}

	err = teamService.RemoveMember(uint(teamID), adminID, uint(memberID))
	if err != nil {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Member removed successfully",
	})
}

// PromoteMember promotes a member to admin
// POST /api/teams/:id/members/:memberId/promote
func PromoteMember(c *fiber.Ctx) error {
	ownerIDRaw := c.Locals("userId")
	var ownerID uint
	switch v := ownerIDRaw.(type) {
	case float64:
		ownerID = uint(v)
	case uint:
		ownerID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}

	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	memberID, err := strconv.ParseUint(c.Params("memberId"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid member ID",
		})
	}

	err = teamService.PromoteMember(uint(teamID), ownerID, uint(memberID))
	if err != nil {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Member promoted to admin",
	})
}

// DemoteMember demotes an admin to member
// POST /api/teams/:id/members/:memberId/demote
func DemoteMember(c *fiber.Ctx) error {
	ownerIDRaw := c.Locals("userId")
	var ownerID uint
	switch v := ownerIDRaw.(type) {
	case float64:
		ownerID = uint(v)
	case uint:
		ownerID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}

	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	memberID, err := strconv.ParseUint(c.Params("memberId"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid member ID",
		})
	}

	err = teamService.DemoteMember(uint(teamID), ownerID, uint(memberID))
	if err != nil {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Member demoted to regular member",
	})
}

// TransferOwnership transfers team ownership
// POST /api/teams/:id/transfer
func TransferOwnership(c *fiber.Ctx) error {
	currentOwnerIDRaw := c.Locals("userId")
	var currentOwnerID uint
	switch v := currentOwnerIDRaw.(type) {
	case float64:
		currentOwnerID = uint(v)
	case uint:
		currentOwnerID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}

	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	var req struct {
		NewOwnerID uint `json:"new_owner_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	err = teamService.TransferOwnership(uint(teamID), currentOwnerID, req.NewOwnerID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Ownership transferred successfully",
	})
}

// ================== TEAM STATISTICS ENDPOINTS ==================

// GetTeamLeaderboard retrieves team leaderboard
// GET /api/teams/:id/leaderboard
func GetTeamLeaderboard(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	// Get limit from query params (default 50)
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	members, err := teamService.GetTeamLeaderboard(uint(teamID), limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve leaderboard",
		})
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"leaderboard": members,
		"count":       len(members),
	})
}

// GetTeamStats retrieves team statistics
// GET /api/teams/:id/stats
func GetTeamStats(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	stats, err := teamService.GetTeamStats(uint(teamID))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"stats":   stats,
	})
}

// UpdateMemberStats updates member statistics after a game
// POST /api/teams/:id/members/:memberId/stats
func UpdateMemberStats(c *fiber.Ctx) error {
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	memberID, err := strconv.ParseUint(c.Params("memberId"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid member ID",
		})
	}

	var req struct {
		Score int `json:"score"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	err = teamService.UpdateMemberStats(uint(teamID), uint(memberID), req.Score)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Stats updated successfully",
	})
}

// ================== TEAM DISCOVERY ENDPOINTS ==================

// SearchTeams searches for public teams
// GET /api/teams/search
func SearchTeams(c *fiber.Ctx) error {
	query := c.Query("q", "")
	limit := 20

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	teams, err := teamService.SearchPublicTeams(query, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to search teams",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"teams":   teams,
		"count":   len(teams),
	})
}

// GetPopularTeams retrieves popular teams
// GET /api/teams/popular
func GetPopularTeams(c *fiber.Ctx) error {
	limit := 10

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	teams, err := teamService.GetPopularTeams(limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve popular teams",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"teams":   teams,
		"count":   len(teams),
	})
}

// GetPublicTeams retrieves public teams (no auth required)
// GET /api/teams/public
func GetPublicTeams(c *fiber.Ctx) error {
	limit := 20

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	teams, err := teamService.GetPublicTeams(limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve public teams",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"teams":   teams,
		"count":   len(teams),
	})
}

// CheckMembership checks if user is a member of a team
// GET /api/teams/:id/check-membership
func CheckMembership(c *fiber.Ctx) error {
	userIDRaw := c.Locals("userId")
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "Invalid user ID"})
	}
	teamID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid team ID",
		})
	}

	isMember := teamService.IsTeamMember(userID, uint(teamID))
	isAdmin := teamService.IsTeamAdmin(userID, uint(teamID))

	var role models.TeamRole
	if isMember {
		role, _ = teamService.GetMemberRole(userID, uint(teamID))
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"is_member": isMember,
		"is_admin":  isAdmin,
		"role":      role,
	})
}

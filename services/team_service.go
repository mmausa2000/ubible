// services/team_service.go - Complete Team Portal Business Logic
package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
	"ubible/models"

	"gorm.io/gorm"
)

type TeamService struct {
	db *gorm.DB
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{db: db}
}

// ================== TEAM CRUD OPERATIONS ==================

// CreateTeam creates a new team with the user as owner
func (s *TeamService) CreateTeam(name, description string, isPublic bool, creatorID uint) (*models.Team, error) {
	if name == "" {
		return nil, errors.New("team name is required")
	}

	// Generate unique team code
	teamCode := s.generateUniqueTeamCode()

	team := &models.Team{
		Name:        name,
		Description: description,
		TeamCode:    teamCode,
		IsPublic:    isPublic,
		CreatorID:   creatorID,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	// Create team and add creator as owner in a transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}

		// Add creator as team owner
		member := &models.TeamMember{
			TeamID:        team.ID,
			UserID:        creatorID,
			Role:          models.TeamRoleOwner,
			JoinedAt:      time.Now(),
			IsActive:      true,
			TotalScore:    0,
			QuizzesPlayed: 0,
		}

		return tx.Create(member).Error
	})

	if err != nil {
		return nil, err
	}

	return team, nil
}

// GetTeamByID retrieves a team by ID with members preloaded
func (s *TeamService) GetTeamByID(teamID uint) (*models.Team, error) {
	var team models.Team
	err := s.db.Where("id = ? AND is_active = ?", teamID, true).
		Preload("Members").
		Preload("Members.User").
		First(&team).Error

	if err != nil {
		return nil, err
	}

	return &team, nil
}

// GetTeamByCode retrieves a team by its join code
func (s *TeamService) GetTeamByCode(code string) (*models.Team, error) {
	var team models.Team
	err := s.db.Where("team_code = ? AND is_active = ?", code, true).
		Preload("Members").
		First(&team).Error

	if err != nil {
		return nil, errors.New("team not found or inactive")
	}

	return &team, nil
}

// GetUserTeams retrieves all teams a user is a member of
func (s *TeamService) GetUserTeams(userID uint) ([]models.Team, error) {
	var teams []models.Team

	err := s.db.Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.user_id = ? AND team_members.is_active = ? AND teams.is_active = ?",
			userID, true, true).
		Preload("Members", "is_active = ?", true).
		Find(&teams).Error

	return teams, err
}

// UpdateTeam updates team information (owner/admin only)
func (s *TeamService) UpdateTeam(teamID uint, name, description string, isPublic bool, updaterID uint) error {
	// Verify updater is owner or admin
	if !s.IsTeamAdmin(updaterID, teamID) {
		return errors.New("only team owner or admin can update team")
	}

	updates := map[string]interface{}{
		"name":        name,
		"description": description,
		"is_public":   isPublic,
		"updated_at":  time.Now(),
	}

	return s.db.Model(&models.Team{}).Where("id = ?", teamID).Updates(updates).Error
}

// DeleteTeam soft deletes a team (owner only)
func (s *TeamService) DeleteTeam(teamID, ownerID uint) error {
	// Verify ownership
	var member models.TeamMember
	if err := s.db.Where("team_id = ? AND user_id = ?", teamID, ownerID).First(&member).Error; err != nil {
		return errors.New("team not found")
	}

	if member.Role != models.TeamRoleOwner {
		return errors.New("only team owner can delete team")
	}

	// Soft delete team and all memberships
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Deactivate all members
		if err := tx.Model(&models.TeamMember{}).Where("team_id = ?", teamID).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// Deactivate team
		return tx.Model(&models.Team{}).Where("id = ?", teamID).
			Update("is_active", false).Error
	})
}

// ================== TEAM MEMBERSHIP OPERATIONS ==================

// JoinTeam adds a user to a team via invite code
func (s *TeamService) JoinTeam(userID uint, teamCode string) error {
	// Get team by code
	team, err := s.GetTeamByCode(teamCode)
	if err != nil {
		return err
	}

	// Check if already a member
	if s.IsTeamMember(userID, team.ID) {
		return errors.New("already a member of this team")
	}

	// Add as member
	member := &models.TeamMember{
		TeamID:        team.ID,
		UserID:        userID,
		Role:          models.TeamRoleMember,
		JoinedAt:      time.Now(),
		IsActive:      true,
		TotalScore:    0,
		QuizzesPlayed: 0,
	}

	return s.db.Create(member).Error
}

// LeaveTeam removes a user from a team
func (s *TeamService) LeaveTeam(userID, teamID uint) error {
	var member models.TeamMember
	if err := s.db.Where("team_id = ? AND user_id = ?", teamID, userID).First(&member).Error; err != nil {
		return errors.New("not a member of this team")
	}

	// Owner cannot leave without transferring ownership
	if member.Role == models.TeamRoleOwner {
		return errors.New("team owner must transfer ownership before leaving")
	}

	return s.db.Model(&member).Update("is_active", false).Error
}

// RemoveMember removes a member from team (admin/owner only)
func (s *TeamService) RemoveMember(teamID, adminID, memberID uint) error {
	// Verify admin permissions
	if !s.IsTeamAdmin(adminID, teamID) {
		return errors.New("only team admin or owner can remove members")
	}

	// Cannot remove owner
	var targetMember models.TeamMember
	if err := s.db.Where("team_id = ? AND user_id = ?", teamID, memberID).First(&targetMember).Error; err != nil {
		return errors.New("member not found")
	}

	if targetMember.Role == models.TeamRoleOwner {
		return errors.New("cannot remove team owner")
	}

	return s.db.Model(&targetMember).Update("is_active", false).Error
}

// PromoteMember promotes a member to admin (owner only)
func (s *TeamService) PromoteMember(teamID, ownerID, memberID uint) error {
	// Verify ownership
	var owner models.TeamMember
	if err := s.db.Where("team_id = ? AND user_id = ?", teamID, ownerID).First(&owner).Error; err != nil {
		return errors.New("not authorized")
	}

	if owner.Role != models.TeamRoleOwner {
		return errors.New("only team owner can promote members")
	}

	// Update member role
	return s.db.Model(&models.TeamMember{}).
		Where("team_id = ? AND user_id = ?", teamID, memberID).
		Update("role", models.TeamRoleAdmin).Error
}

// DemoteMember demotes an admin to regular member (owner only)
func (s *TeamService) DemoteMember(teamID, ownerID, memberID uint) error {
	// Verify ownership
	var owner models.TeamMember
	if err := s.db.Where("team_id = ? AND user_id = ?", teamID, ownerID).First(&owner).Error; err != nil {
		return errors.New("not authorized")
	}

	if owner.Role != models.TeamRoleOwner {
		return errors.New("only team owner can demote members")
	}

	// Cannot demote owner
	var targetMember models.TeamMember
	if err := s.db.Where("team_id = ? AND user_id = ?", teamID, memberID).First(&targetMember).Error; err != nil {
		return errors.New("member not found")
	}

	if targetMember.Role == models.TeamRoleOwner {
		return errors.New("cannot demote team owner")
	}

	// Update member role
	return s.db.Model(&targetMember).Update("role", models.TeamRoleMember).Error
}

// TransferOwnership transfers team ownership to another member
func (s *TeamService) TransferOwnership(teamID, currentOwnerID, newOwnerID uint) error {
	// Verify current ownership
	var currentOwner models.TeamMember
	if err := s.db.Where("team_id = ? AND user_id = ?", teamID, currentOwnerID).First(&currentOwner).Error; err != nil {
		return errors.New("not authorized")
	}

	if currentOwner.Role != models.TeamRoleOwner {
		return errors.New("only team owner can transfer ownership")
	}

	// Verify new owner is a member
	var newOwnerMember models.TeamMember
	if err := s.db.Where("team_id = ? AND user_id = ?", teamID, newOwnerID).First(&newOwnerMember).Error; err != nil {
		return errors.New("new owner must be a team member")
	}

	// Update roles in transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Demote current owner to admin
		if err := tx.Model(&models.TeamMember{}).
			Where("team_id = ? AND user_id = ?", teamID, currentOwnerID).
			Update("role", models.TeamRoleAdmin).Error; err != nil {
			return err
		}

		// Promote new owner
		if err := tx.Model(&models.TeamMember{}).
			Where("team_id = ? AND user_id = ?", teamID, newOwnerID).
			Update("role", models.TeamRoleOwner).Error; err != nil {
			return err
		}

		// Update team creator
		return tx.Model(&models.Team{}).
			Where("id = ?", teamID).
			Update("creator_id", newOwnerID).Error
	})
}

// ================== TEAM STATISTICS & LEADERBOARD ==================

// GetTeamMembers returns all active members of a team with stats
func (s *TeamService) GetTeamMembers(teamID uint) ([]models.TeamMember, error) {
	var members []models.TeamMember

	err := s.db.Where("team_id = ? AND is_active = ?", teamID, true).
		Preload("User").
		Order("role ASC, total_score DESC").
		Find(&members).Error

	return members, err
}

// GetTeamLeaderboard returns team members sorted by score
func (s *TeamService) GetTeamLeaderboard(teamID uint, limit int) ([]models.TeamMember, error) {
	var members []models.TeamMember

	query := s.db.Where("team_id = ? AND is_active = ?", teamID, true).
		Preload("User").
		Order("total_score DESC, quizzes_played DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&members).Error
	return members, err
}

// UpdateMemberStats updates a member's game statistics
func (s *TeamService) UpdateMemberStats(teamID, userID uint, scoreToAdd int) error {
	return s.db.Model(&models.TeamMember{}).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Updates(map[string]interface{}{
			"total_score":    gorm.Expr("total_score + ?", scoreToAdd),
			"quizzes_played": gorm.Expr("quizzes_played + 1"),
			"last_active":    time.Now(),
		}).Error
}

// GetTeamStats returns aggregated team statistics
func (s *TeamService) GetTeamStats(teamID uint) (map[string]interface{}, error) {
	var stats struct {
		TotalMembers  int64
		ActiveMembers int64
		TotalScore    int
		TotalQuizzes  int
		AvgScore      float64
	}

	// Get total members count
	s.db.Model(&models.TeamMember{}).
		Where("team_id = ?", teamID).
		Count(&stats.TotalMembers)

	// Get active members and their stats
	s.db.Model(&models.TeamMember{}).
		Where("team_id = ? AND is_active = ?", teamID, true).
		Count(&stats.ActiveMembers)

	// Get aggregate stats
	s.db.Model(&models.TeamMember{}).
		Where("team_id = ? AND is_active = ?", teamID, true).
		Select("SUM(total_score) as total_score, SUM(quizzes_played) as total_quizzes").
		Scan(&stats)

	if stats.TotalQuizzes > 0 {
		stats.AvgScore = float64(stats.TotalScore) / float64(stats.TotalQuizzes)
	}

	return map[string]interface{}{
		"total_members":  stats.TotalMembers,
		"active_members": stats.ActiveMembers,
		"total_score":    stats.TotalScore,
		"total_quizzes":  stats.TotalQuizzes,
		"avg_score":      stats.AvgScore,
	}, nil
}

// ================== TEAM SEARCH & DISCOVERY ==================

// SearchPublicTeams searches for public teams by name
func (s *TeamService) SearchPublicTeams(query string, limit int) ([]models.Team, error) {
	var teams []models.Team

	searchQuery := s.db.Where("is_public = ? AND is_active = ?", true, true)

	if query != "" {
		searchQuery = searchQuery.Where("name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%")
	}

	err := searchQuery.
		Preload("Members", "is_active = ?", true).
		Limit(limit).
		Order("created_at DESC").
		Find(&teams).Error

	return teams, err
}

// GetPopularTeams returns teams with most members
func (s *TeamService) GetPopularTeams(limit int) ([]models.Team, error) {
	var teams []models.Team

	err := s.db.
		Select("teams.*, COUNT(team_members.id) as member_count").
		Joins("LEFT JOIN team_members ON team_members.team_id = teams.id AND team_members.is_active = true").
		Where("teams.is_public = ? AND teams.is_active = ?", true, true).
		Group("teams.id").
		Order("member_count DESC").
		Limit(limit).
		Preload("Members", "is_active = ?", true).
		Find(&teams).Error

	return teams, err
}

// GetPublicTeams returns all public teams
func (s *TeamService) GetPublicTeams(limit int) ([]models.Team, error) {
	var teams []models.Team

	err := s.db.
		Where("is_public = ? AND is_active = ?", true, true).
		Preload("Members", "is_active = ?", true).
		Order("created_at DESC").
		Limit(limit).
		Find(&teams).Error

	return teams, err
}

// ================== HELPER FUNCTIONS ==================

// IsTeamMember checks if a user is an active member of a team
func (s *TeamService) IsTeamMember(userID, teamID uint) bool {
	var count int64
	s.db.Model(&models.TeamMember{}).
		Where("team_id = ? AND user_id = ? AND is_active = ?", teamID, userID, true).
		Count(&count)
	return count > 0
}

// IsTeamAdmin checks if a user is owner or admin of a team
func (s *TeamService) IsTeamAdmin(userID, teamID uint) bool {
	var member models.TeamMember
	err := s.db.Where("team_id = ? AND user_id = ? AND is_active = ?", teamID, userID, true).
		First(&member).Error

	if err != nil {
		return false
	}

	return member.Role == models.TeamRoleOwner || member.Role == models.TeamRoleAdmin
}

// GetMemberRole returns the role of a user in a team
func (s *TeamService) GetMemberRole(userID, teamID uint) (models.TeamRole, error) {
	var member models.TeamMember
	err := s.db.Where("team_id = ? AND user_id = ? AND is_active = ?", teamID, userID, true).
		First(&member).Error

	if err != nil {
		return "", errors.New("not a member of this team")
	}

	return member.Role, nil
}

// generateUniqueTeamCode generates a unique 6-character alphanumeric code
func (s *TeamService) generateUniqueTeamCode() string {
	for {
		bytes := make([]byte, 3)
		rand.Read(bytes)
		code := hex.EncodeToString(bytes)[:6]

		// Check if code already exists
		var count int64
		s.db.Model(&models.Team{}).Where("team_code = ?", code).Count(&count)

		if count == 0 {
			return code
		}
	}
}

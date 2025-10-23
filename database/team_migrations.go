// database/team_migrations.go - Team Portal Database Migrations
package database

import (
	"ubible/models"
	"log"

	"gorm.io/gorm"
)

// RunTeamMigrations creates all team portal tables
func RunTeamMigrations(db *gorm.DB) error {
	log.Println("Running Team Portal migrations...")

	// Migrate team models
	if err := db.AutoMigrate(
		&models.Team{},
		&models.TeamMember{},
		&models.TeamTheme{},
		&models.Challenge{},
		&models.ChallengeParticipant{},
	); err != nil {
		return err
	}

	// Create indexes for better performance
	if err := createTeamIndexes(db); err != nil {
		return err
	}

	log.Println("✅ Team Portal migrations completed successfully")
	return nil
}

// createTeamIndexes creates database indexes for team tables
func createTeamIndexes(db *gorm.DB) error {
	log.Println("Creating Team Portal indexes...")

	// Team indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_teams_creator ON teams(creator_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_teams_code ON teams(team_code)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_teams_public ON teams(is_public)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_teams_active ON teams(is_active)")

	// Team member indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_members_team ON team_members(team_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_members_active ON team_members(is_active)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_members_role ON team_members(role)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_members_score ON team_members(total_score DESC)")

	// Team theme indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_themes_team ON team_themes(team_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_themes_creator ON team_themes(created_by)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_themes_public ON team_themes(is_public)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_themes_active ON team_themes(is_active)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_team_themes_usage ON team_themes(usage_count DESC)")

	// Challenge indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_challenges_team ON challenges(team_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_challenges_creator ON challenges(created_by)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_challenges_theme ON challenges(theme_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_challenges_status ON challenges(status)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_challenges_dates ON challenges(start_date, end_date)")

	// Challenge participant indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_challenge_participants_challenge ON challenge_participants(challenge_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_challenge_participants_user ON challenge_participants(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_challenge_participants_score ON challenge_participants(score DESC)")

	log.Println("✅ Team Portal indexes created successfully")
	return nil
}

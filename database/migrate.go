// database/migrate.go - Database Migration Runner
package database

import (
	"ubible/models"
	"log"
)

// RunMigrations runs all database migrations
func RunMigrations() {
	db := GetDB()
	log.Println("🔄 Running database migrations...")

	// Core application models
	if err := db.AutoMigrate(
		&models.User{},
		&models.Theme{},
		&models.Question{},
		&models.Attempt{},
		&models.Achievement{},
		&models.UserAchievement{},
		&models.Friend{},
		&models.FriendRequest{},
		&models.PowerUp{},
	); err != nil {
		log.Fatalf("❌ Failed to run core migrations: %v", err)
	}

	log.Println("✅ Core migrations completed")

	// Run Team Portal migrations
	if err := RunTeamMigrations(db); err != nil {
		log.Fatalf("❌ Failed to run team migrations: %v", err)
	}

	// Create indexes for core tables
	createCoreIndexes()

	log.Println("✅ All migrations completed successfully")
}

// createCoreIndexes creates indexes for core tables
func createCoreIndexes() {
	db := GetDB()
	log.Println("Creating core indexes...")

	// User indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_level ON users(level DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_guest ON users(is_guest)")

	// Theme indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_themes_active ON themes(is_active)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_themes_default ON themes(is_default)")

	// Question indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_questions_theme ON questions(theme_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_questions_difficulty ON questions(difficulty)")

	// Attempt indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_attempts_user ON attempts(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_attempts_theme ON attempts(theme_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_attempts_created ON attempts(created_at DESC)")

	// Achievement indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_achievements_type ON achievements(type)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_achievements_user ON user_achievements(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_achievements_achievement ON user_achievements(achievement_id)")

	// Friend indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_friends_user ON friends(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_friend_requests_to ON friend_requests(to_user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_friend_requests_from ON friend_requests(from_user_id)")

	log.Println("✅ Core indexes created successfully")
}

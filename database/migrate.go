// database/migrate.go - Database Migration Runner
package database

import (
	"log"
	"ubible/models"
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

	// Multiplayer tracking models - all relationships removed from gorm tags to avoid circular dependencies
	if err := db.AutoMigrate(
		&models.MultiplayerGame{},
		&models.MultiplayerGamePlayer{},
		&models.MultiplayerGameEvent{},
	); err != nil {
		log.Fatalf("❌ Failed to run multiplayer tracking migrations: %v", err)
	}

	log.Println("✅ Multiplayer tracking migrations completed")

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
	db.Exec("CREATE INDEX IF NOT EXISTS idx_achievements_category ON achievements(category)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_achievements_user ON user_achievements(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_achievements_achievement ON user_achievements(achievement_id)")

	// Friend indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_friends_user ON friends(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_friend_requests_to ON friend_requests(to_user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_friend_requests_from ON friend_requests(from_user_id)")

	log.Println("✅ Core indexes created successfully")

	// Multiplayer game indexes
	createMultiplayerIndexes()
}

// createMultiplayerIndexes creates indexes for multiplayer tracking tables
func createMultiplayerIndexes() {
	db := GetDB()
	log.Println("Creating multiplayer tracking indexes...")

	// MultiplayerGame indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_games_game_id ON multiplayer_games(game_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_games_room_code ON multiplayer_games(room_code)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_games_status ON multiplayer_games(status)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_games_created ON multiplayer_games(created_at DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_games_started ON multiplayer_games(started_at DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_games_completed ON multiplayer_games(completed_at DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_games_host ON multiplayer_games(host_player_id)")

	// MultiplayerGamePlayer indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_players_game ON multiplayer_game_players(game_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_players_player_id ON multiplayer_game_players(player_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_players_user ON multiplayer_game_players(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_players_guest ON multiplayer_game_players(is_guest)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_players_joined ON multiplayer_game_players(joined_at DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_players_composite ON multiplayer_game_players(game_id, player_id)")

	// MultiplayerGameEvent indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_events_game ON multiplayer_game_events(game_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_events_type ON multiplayer_game_events(event_type)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_events_player ON multiplayer_game_events(player_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_events_timestamp ON multiplayer_game_events(timestamp DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_events_sequence ON multiplayer_game_events(game_id, sequence_num)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_mp_events_composite ON multiplayer_game_events(game_id, event_type, timestamp)")

	log.Println("✅ Multiplayer tracking indexes created successfully")
}

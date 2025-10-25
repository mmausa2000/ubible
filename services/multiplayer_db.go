// services/multiplayer_db.go - Multiplayer Database Service
package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"ubible/database"
	"ubible/models"
)

// MultiplayerDBService handles database operations for multiplayer games
type MultiplayerDBService struct{}

// NewMultiplayerDBService creates a new multiplayer database service
func NewMultiplayerDBService() *MultiplayerDBService {
	return &MultiplayerDBService{}
}

// CreateGame creates a new multiplayer game record
func (s *MultiplayerDBService) CreateGame(gameID, roomCode, gameURL, hostPlayerID string, maxPlayers, questionCount, timeLimit int, selectedThemes []int) (*models.MultiplayerGame, error) {
	db := database.GetDB()

	themesJSON, err := json.Marshal(selectedThemes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal themes: %w", err)
	}

	game := &models.MultiplayerGame{
		GameID:         gameID,
		RoomCode:       roomCode,
		GameURL:        gameURL,
		HostPlayerID:   hostPlayerID,
		MaxPlayers:     maxPlayers,
		QuestionCount:  questionCount,
		TimeLimit:      timeLimit,
		SelectedThemes: string(themesJSON),
		Status:         "waiting",
		CurrentQuestion: 0,
	}

	if err := db.Create(game).Error; err != nil {
		return nil, fmt.Errorf("failed to create game: %w", err)
	}

	log.Printf("📊 DB: Created game record: ID=%s, Room=%s", gameID, roomCode)
	return game, nil
}

// AddPlayer adds a player to a game
func (s *MultiplayerDBService) AddPlayer(gameID string, playerID, username string, userID *uint, isGuest, isHost, isPlaying bool) (*models.MultiplayerGamePlayer, error) {
	db := database.GetDB()

	// Find the game
	var game models.MultiplayerGame
	if err := db.Where("game_id = ?", gameID).First(&game).Error; err != nil {
		return nil, fmt.Errorf("game not found: %w", err)
	}

	player := &models.MultiplayerGamePlayer{
		GameID:     game.ID,
		PlayerID:   playerID,
		UserID:     userID,
		Username:   username,
		IsGuest:    isGuest,
		IsHost:     isHost,
		IsPlaying:  isPlaying,
		IsReady:    false,
		JoinedAt:   time.Now(),
	}

	if err := db.Create(player).Error; err != nil {
		return nil, fmt.Errorf("failed to add player: %w", err)
	}

	log.Printf("📊 DB: Player %s joined game %s", playerID, gameID)
	return player, nil
}

// UpdatePlayerReady updates a player's ready status
func (s *MultiplayerDBService) UpdatePlayerReady(gameID, playerID string, isReady bool) error {
	db := database.GetDB()

	result := db.Model(&models.MultiplayerGamePlayer{}).
		Where("game_id IN (SELECT id FROM multiplayer_games WHERE game_id = ?) AND player_id = ?", gameID, playerID).
		Update("is_ready", isReady)

	if result.Error != nil {
		return fmt.Errorf("failed to update player ready: %w", result.Error)
	}

	log.Printf("📊 DB: Player %s ready status: %v", playerID, isReady)
	return nil
}

// StartGame marks a game as started
func (s *MultiplayerDBService) StartGame(gameID string) error {
	db := database.GetDB()

	now := time.Now()
	result := db.Model(&models.MultiplayerGame{}).
		Where("game_id = ?", gameID).
		Updates(map[string]interface{}{
			"status":     "playing",
			"started_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to start game: %w", result.Error)
	}

	log.Printf("📊 DB: Game %s started at %s", gameID, now.Format(time.RFC3339))
	return nil
}

// UpdatePlayerScore updates a player's score and stats
func (s *MultiplayerDBService) UpdatePlayerScore(gameID, playerID string, score, correctAnswers, wrongAnswers int) error {
	db := database.GetDB()

	result := db.Model(&models.MultiplayerGamePlayer{}).
		Where("game_id IN (SELECT id FROM multiplayer_games WHERE game_id = ?) AND player_id = ?", gameID, playerID).
		Updates(map[string]interface{}{
			"final_score":        score,
			"correct_answers":    correctAnswers,
			"wrong_answers":      wrongAnswers,
			"questions_answered": correctAnswers + wrongAnswers,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update player score: %w", result.Error)
	}

	return nil
}

// UpdateCurrentQuestion updates the game's current question index
func (s *MultiplayerDBService) UpdateCurrentQuestion(gameID string, questionIndex int) error {
	db := database.GetDB()

	result := db.Model(&models.MultiplayerGame{}).
		Where("game_id = ?", gameID).
		Update("current_question", questionIndex)

	if result.Error != nil {
		return fmt.Errorf("failed to update current question: %w", result.Error)
	}

	return nil
}

// CompleteGame marks a game as completed and calculates placements
func (s *MultiplayerDBService) CompleteGame(gameID string) error {
	db := database.GetDB()

	now := time.Now()

	// Update game status
	if err := db.Model(&models.MultiplayerGame{}).
		Where("game_id = ?", gameID).
		Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to complete game: %w", err)
	}

	// Calculate placements
	var game models.MultiplayerGame
	if err := db.Where("game_id = ?", gameID).First(&game).Error; err != nil {
		return fmt.Errorf("game not found: %w", err)
	}

	var players []models.MultiplayerGamePlayer
	if err := db.Where("game_id = ?", game.ID).Order("final_score DESC, correct_answers DESC").Find(&players).Error; err != nil {
		return fmt.Errorf("failed to get players: %w", err)
	}

	// Assign placements
	for i, player := range players {
		placement := i + 1
		db.Model(&models.MultiplayerGamePlayer{}).
			Where("id = ?", player.ID).
			Update("placement", placement)
	}

	log.Printf("📊 DB: Game %s completed with %d players", gameID, len(players))
	return nil
}

// RecordPlayerDisconnect records when a player disconnects
func (s *MultiplayerDBService) RecordPlayerDisconnect(gameID, playerID string) error {
	db := database.GetDB()

	now := time.Now()
	result := db.Model(&models.MultiplayerGamePlayer{}).
		Where("game_id IN (SELECT id FROM multiplayer_games WHERE game_id = ?) AND player_id = ?", gameID, playerID).
		Update("disconnected_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to record disconnect: %w", result.Error)
	}

	log.Printf("📊 DB: Player %s disconnected from game %s", playerID, gameID)
	return nil
}

// RecordPlayerReconnect records when a player reconnects
func (s *MultiplayerDBService) RecordPlayerReconnect(gameID, playerID string) error {
	db := database.GetDB()

	now := time.Now()
	result := db.Model(&models.MultiplayerGamePlayer{}).
		Where("game_id IN (SELECT id FROM multiplayer_games WHERE game_id = ?) AND player_id = ?", gameID, playerID).
		Update("reconnected_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to record reconnect: %w", result.Error)
	}

	log.Printf("📊 DB: Player %s reconnected to game %s", playerID, gameID)
	return nil
}

// RecordPlayerLeft records when a player leaves
func (s *MultiplayerDBService) RecordPlayerLeft(gameID, playerID string) error {
	db := database.GetDB()

	now := time.Now()
	result := db.Model(&models.MultiplayerGamePlayer{}).
		Where("game_id IN (SELECT id FROM multiplayer_games WHERE game_id = ?) AND player_id = ?", gameID, playerID).
		Update("left_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to record player left: %w", result.Error)
	}

	log.Printf("📊 DB: Player %s left game %s", playerID, gameID)
	return nil
}

// LogEvent logs a game event
func (s *MultiplayerDBService) LogEvent(gameID, eventType, playerID string, questionIndex *int, eventData map[string]interface{}, sequenceNum int64) error {
	db := database.GetDB()

	// Find the game
	var game models.MultiplayerGame
	if err := db.Where("game_id = ?", gameID).First(&game).Error; err != nil {
		return fmt.Errorf("game not found: %w", err)
	}

	dataJSON, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	event := &models.MultiplayerGameEvent{
		GameID:        game.ID,
		EventType:     eventType,
		PlayerID:      playerID,
		QuestionIndex: questionIndex,
		EventData:     string(dataJSON),
		Timestamp:     time.Now(),
		SequenceNum:   sequenceNum,
	}

	if err := db.Create(event).Error; err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}

	return nil
}

// GetGameByID retrieves a game by game_id
func (s *MultiplayerDBService) GetGameByID(gameID string) (*models.MultiplayerGame, error) {
	db := database.GetDB()

	var game models.MultiplayerGame
	if err := db.Preload("Players").Preload("Events").Where("game_id = ?", gameID).First(&game).Error; err != nil {
		return nil, fmt.Errorf("game not found: %w", err)
	}

	return &game, nil
}

// GetRecentGames retrieves recent games with pagination
func (s *MultiplayerDBService) GetRecentGames(limit, offset int) ([]models.MultiplayerGame, error) {
	db := database.GetDB()

	var games []models.MultiplayerGame
	if err := db.Preload("Players").Order("created_at DESC").Limit(limit).Offset(offset).Find(&games).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent games: %w", err)
	}

	return games, nil
}

// GetPlayerHistory retrieves a player's game history
func (s *MultiplayerDBService) GetPlayerHistory(playerID string, limit int) ([]models.MultiplayerGamePlayer, error) {
	db := database.GetDB()

	var players []models.MultiplayerGamePlayer
	if err := db.Preload("Game").Where("player_id = ?", playerID).Order("joined_at DESC").Limit(limit).Find(&players).Error; err != nil {
		return nil, fmt.Errorf("failed to get player history: %w", err)
	}

	return players, nil
}

// GetGameEvents retrieves events for a game
func (s *MultiplayerDBService) GetGameEvents(gameID string) ([]models.MultiplayerGameEvent, error) {
	db := database.GetDB()

	var game models.MultiplayerGame
	if err := db.Where("game_id = ?", gameID).First(&game).Error; err != nil {
		return nil, fmt.Errorf("game not found: %w", err)
	}

	var events []models.MultiplayerGameEvent
	if err := db.Where("game_id = ?", game.ID).Order("sequence_num ASC, timestamp ASC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	return events, nil
}

// GetActiveGames retrieves all currently active (waiting or playing) games
func (s *MultiplayerDBService) GetActiveGames() ([]models.MultiplayerGame, error) {
	db := database.GetDB()

	var games []models.MultiplayerGame
	if err := db.Preload("Players").Where("status IN ?", []string{"waiting", "playing"}).Order("created_at DESC").Find(&games).Error; err != nil {
		return nil, fmt.Errorf("failed to get active games: %w", err)
	}

	return games, nil
}

// Global instance
var MultiplayerDB = NewMultiplayerDBService()

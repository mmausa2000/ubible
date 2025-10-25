// models/multiplayer.go - Multiplayer Game Tracking Models
package models

import (
	"time"
)

// MultiplayerGame represents a multiplayer quiz game session
type MultiplayerGame struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	GameID         string    `json:"game_id" gorm:"uniqueIndex;not null;size:100"` // UUID from multiplayer handler
	RoomCode       string    `json:"room_code" gorm:"index;not null;size:20"`
	GameURL        string    `json:"game_url" gorm:"size:200"`
	HostPlayerID   string    `json:"host_player_id" gorm:"index;size:100"`
	MaxPlayers     int       `json:"max_players" gorm:"default:10"`
	QuestionCount  int       `json:"question_count" gorm:"default:10"`
	TimeLimit      int       `json:"time_limit" gorm:"default:10"` // seconds per question
	SelectedThemes string    `json:"selected_themes" gorm:"type:text"` // JSON array of theme IDs

	// Game state
	Status         string    `json:"status" gorm:"default:'waiting';size:20;index"` // waiting, playing, completed, abandoned
	CurrentQuestion int      `json:"current_question" gorm:"default:0"`

	// Timestamps
	CreatedAt      time.Time `json:"created_at" gorm:"index"`
	StartedAt      *time.Time `json:"started_at" gorm:"index"`
	CompletedAt    *time.Time `json:"completed_at" gorm:"index"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relationships (loaded via Preload, not enforced at DB level on parent)
	Players        []MultiplayerGamePlayer `json:"players,omitempty" gorm:"-"`
	Events         []MultiplayerGameEvent  `json:"events,omitempty" gorm:"-"`
}

// MultiplayerGamePlayer represents a player's participation in a multiplayer game
type MultiplayerGamePlayer struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	GameID            uint      `json:"game_id" gorm:"not null;index"`
	Game              *MultiplayerGame `json:"game,omitempty" gorm:"-"`

	// Player info
	PlayerID          string    `json:"player_id" gorm:"not null;index;size:100"` // In-game player ID
	UserID            *uint     `json:"user_id" gorm:"index"` // Database user ID (nil for guests)
	User              *User     `json:"user,omitempty" gorm:"-"`
	Username          string    `json:"username" gorm:"size:100"`
	IsGuest           bool      `json:"is_guest" gorm:"default:false;index"`
	IsHost            bool      `json:"is_host" gorm:"default:false"`

	// Game participation
	IsPlaying         bool      `json:"is_playing" gorm:"default:true"`
	IsReady           bool      `json:"is_ready" gorm:"default:false"`

	// Performance stats
	FinalScore        int       `json:"final_score" gorm:"default:0"`
	CorrectAnswers    int       `json:"correct_answers" gorm:"default:0"`
	WrongAnswers      int       `json:"wrong_answers" gorm:"default:0"`
	QuestionsAnswered int       `json:"questions_answered" gorm:"default:0"`
	Placement         int       `json:"placement" gorm:"default:0"` // 1st, 2nd, 3rd, etc.

	// Timestamps
	JoinedAt          time.Time  `json:"joined_at" gorm:"index"`
	LeftAt            *time.Time `json:"left_at"`
	DisconnectedAt    *time.Time `json:"disconnected_at"`
	ReconnectedAt     *time.Time `json:"reconnected_at"`

	// Rewards (if applicable)
	XPEarned          int       `json:"xp_earned" gorm:"default:0"`
	FPEarned          int       `json:"fp_earned" gorm:"default:0"`

	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// MultiplayerGameEvent represents an event that occurred during a multiplayer game
type MultiplayerGameEvent struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	GameID      uint      `json:"game_id" gorm:"not null;index"`
	Game        *MultiplayerGame `json:"game,omitempty" gorm:"-"`

	// Event details
	EventType   string    `json:"event_type" gorm:"not null;size:50;index"` // player_joined, player_left, answer_submitted, question_advanced, game_started, game_completed, player_disconnected, player_reconnected
	PlayerID    string    `json:"player_id" gorm:"index;size:100"` // Empty for game-level events

	// Event data (JSON)
	EventData   string    `json:"event_data" gorm:"type:text"` // JSON with event-specific data

	// Question context (if applicable)
	QuestionIndex *int    `json:"question_index"` // Which question was active when event occurred

	// Metadata
	Timestamp   time.Time `json:"timestamp" gorm:"index;not null"`
	SequenceNum int64     `json:"sequence_num"` // For ordering events

	CreatedAt   time.Time `json:"created_at"`
}

// TableName methods for custom table names
func (MultiplayerGame) TableName() string {
	return "multiplayer_games"
}

func (MultiplayerGamePlayer) TableName() string {
	return "multiplayer_game_players"
}

func (MultiplayerGameEvent) TableName() string {
	return "multiplayer_game_events"
}

// Helper methods

// IsActive checks if game is currently active
func (g *MultiplayerGame) IsActive() bool {
	return g.Status == "waiting" || g.Status == "playing"
}

// Duration returns how long the game lasted
func (g *MultiplayerGame) Duration() time.Duration {
	if g.StartedAt == nil || g.CompletedAt == nil {
		return 0
	}
	return g.CompletedAt.Sub(*g.StartedAt)
}

// ActivePlayerCount returns number of active (not left) players
func (g *MultiplayerGame) ActivePlayerCount() int {
	count := 0
	for _, p := range g.Players {
		if p.LeftAt == nil {
			count++
		}
	}
	return count
}

// WasDisconnected checks if player disconnected during game
func (p *MultiplayerGamePlayer) WasDisconnected() bool {
	return p.DisconnectedAt != nil
}

// DisconnectDuration returns how long player was disconnected
func (p *MultiplayerGamePlayer) DisconnectDuration() time.Duration {
	if p.DisconnectedAt == nil || p.ReconnectedAt == nil {
		return 0
	}
	return p.ReconnectedAt.Sub(*p.DisconnectedAt)
}

// AccuracyRate returns percentage of correct answers
func (p *MultiplayerGamePlayer) AccuracyRate() float64 {
	if p.QuestionsAnswered == 0 {
		return 0
	}
	return float64(p.CorrectAnswers) / float64(p.QuestionsAnswered) * 100
}

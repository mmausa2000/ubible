// models/game_state.go - Game state persistence for Chess.com-style experience
package models

import (
	"encoding/json"
	"time"
)

// ActiveGameState stores the complete state of an active game session
// This enables Chess.com-style persistence across page refreshes
type ActiveGameState struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	GameID    string    `json:"game_id" gorm:"uniqueIndex;not null;size:100"` // UUID from multiplayer.go
	RoomCode  string    `json:"room_code" gorm:"index;size:10"`                // Room code if multiplayer
	GameToken string    `json:"game_token" gorm:"not null;size:100"`          // Security token

	// Game Type
	IsMultiplayer bool   `json:"is_multiplayer" gorm:"default:false"`
	IsSinglePlayer bool  `json:"is_single_player" gorm:"default:false"`

	// Current Game State
	CurrentQuestionIndex int       `json:"current_question_index" gorm:"default:0"`   // Which question (0-based)
	TotalQuestions       int       `json:"total_questions" gorm:"default:10"`
	TimeLimit            int       `json:"time_limit" gorm:"default:10"`              // Seconds per question
	TimeRemaining        int       `json:"time_remaining" gorm:"default:10"`          // Seconds left on current question
	QuestionStartedAt    time.Time `json:"question_started_at"`                       // When current question started

	// Score & Progress
	CurrentScore     int  `json:"current_score" gorm:"default:0"`
	CorrectAnswers   int  `json:"correct_answers" gorm:"default:0"`
	WrongAnswers     int  `json:"wrong_answers" gorm:"default:0"`
	StreakCount      int  `json:"streak_count" gorm:"default:0"`

	// Questions Data (stored as JSON)
	QuestionsJSON    string `json:"questions_json" gorm:"type:text"`              // Array of question IDs/data
	UserAnswersJSON  string `json:"user_answers_json" gorm:"type:text"`           // Array of user answers

	// Theme Configuration
	SelectedThemesJSON string `json:"selected_themes_json" gorm:"type:text"`       // Array of theme IDs

	// Player Authorization (stored as JSON)
	AuthorizedPlayersJSON string `json:"authorized_players_json" gorm:"type:text"`  // Map of playerID -> bool

	// Player Sessions (for reconnection)
	PlayerSessionsJSON string `json:"player_sessions_json" gorm:"type:text"`       // Map of playerID -> session data

	// Game Status
	Status    string    `json:"status" gorm:"default:'active';size:20"`        // active, paused, completed, abandoned
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ExpiresAt time.Time `json:"expires_at"`                                     // When to clean up (24 hours)

	// Guest Support
	GuestSessionID string `json:"guest_session_id" gorm:"size:100"`             // For guest players

	// Multiplayer Specific
	HostPlayerID string `json:"host_player_id" gorm:"size:100"`
	Players      string `json:"players" gorm:"type:text"`                       // JSON array of player data
}

// TableName specifies the table name for ActiveGameState
func (ActiveGameState) TableName() string {
	return "active_game_states"
}

// GameStateSnapshot represents a moment-in-time snapshot for restoration
type GameStateSnapshot struct {
	GameID              string                 `json:"game_id"`
	CurrentQuestionIdx  int                    `json:"current_question_index"`
	TotalQuestions      int                    `json:"total_questions"`
	TimeRemaining       int                    `json:"time_remaining"`
	CurrentScore        int                    `json:"current_score"`
	CorrectAnswers      int                    `json:"correct_answers"`
	WrongAnswers        int                    `json:"wrong_answers"`
	StreakCount         int                    `json:"streak_count"`
	Questions           []QuestionData         `json:"questions"`
	UserAnswers         []string               `json:"user_answers"`
	SelectedThemes      []int                  `json:"selected_themes"`
	Status              string                 `json:"status"`
	IsMultiplayer       bool                   `json:"is_multiplayer"`
	AuthorizedPlayers   map[string]bool        `json:"authorized_players"`
}

// QuestionData represents a single question in the game state
type QuestionData struct {
	ID            int      `json:"id"`
	Text          string   `json:"text"`
	CorrectAnswer string   `json:"correct_answer"`
	Options       []string `json:"options"`
	Reference     string   `json:"reference"`
}

// Helper methods to marshal/unmarshal JSON fields

func (ags *ActiveGameState) GetQuestionsData() ([]QuestionData, error) {
	var questions []QuestionData
	if ags.QuestionsJSON == "" {
		return questions, nil
	}
	err := json.Unmarshal([]byte(ags.QuestionsJSON), &questions)
	return questions, err
}

func (ags *ActiveGameState) SetQuestionsData(questions []QuestionData) error {
	data, err := json.Marshal(questions)
	if err != nil {
		return err
	}
	ags.QuestionsJSON = string(data)
	return nil
}

func (ags *ActiveGameState) GetUserAnswers() ([]string, error) {
	var answers []string
	if ags.UserAnswersJSON == "" {
		return answers, nil
	}
	err := json.Unmarshal([]byte(ags.UserAnswersJSON), &answers)
	return answers, err
}

func (ags *ActiveGameState) SetUserAnswers(answers []string) error {
	data, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	ags.UserAnswersJSON = string(data)
	return nil
}

func (ags *ActiveGameState) GetSelectedThemes() ([]int, error) {
	var themes []int
	if ags.SelectedThemesJSON == "" {
		return themes, nil
	}
	err := json.Unmarshal([]byte(ags.SelectedThemesJSON), &themes)
	return themes, err
}

func (ags *ActiveGameState) SetSelectedThemes(themes []int) error {
	data, err := json.Marshal(themes)
	if err != nil {
		return err
	}
	ags.SelectedThemesJSON = string(data)
	return nil
}

func (ags *ActiveGameState) GetAuthorizedPlayers() (map[string]bool, error) {
	players := make(map[string]bool)
	if ags.AuthorizedPlayersJSON == "" {
		return players, nil
	}
	err := json.Unmarshal([]byte(ags.AuthorizedPlayersJSON), &players)
	return players, err
}

func (ags *ActiveGameState) SetAuthorizedPlayers(players map[string]bool) error {
	data, err := json.Marshal(players)
	if err != nil {
		return err
	}
	ags.AuthorizedPlayersJSON = string(data)
	return nil
}

// CreateSnapshot generates a complete snapshot for client restoration
func (ags *ActiveGameState) CreateSnapshot() (*GameStateSnapshot, error) {
	questions, err := ags.GetQuestionsData()
	if err != nil {
		return nil, err
	}

	answers, err := ags.GetUserAnswers()
	if err != nil {
		return nil, err
	}

	themes, err := ags.GetSelectedThemes()
	if err != nil {
		return nil, err
	}

	authorized, err := ags.GetAuthorizedPlayers()
	if err != nil {
		return nil, err
	}

	return &GameStateSnapshot{
		GameID:             ags.GameID,
		CurrentQuestionIdx: ags.CurrentQuestionIndex,
		TotalQuestions:     ags.TotalQuestions,
		TimeRemaining:      ags.TimeRemaining,
		CurrentScore:       ags.CurrentScore,
		CorrectAnswers:     ags.CorrectAnswers,
		WrongAnswers:       ags.WrongAnswers,
		StreakCount:        ags.StreakCount,
		Questions:          questions,
		UserAnswers:        answers,
		SelectedThemes:     themes,
		Status:             ags.Status,
		IsMultiplayer:      ags.IsMultiplayer,
		AuthorizedPlayers:  authorized,
	}, nil
}

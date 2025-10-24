// models/models.go - Core Models (Challenge removed - defined in challenge.go)
package models

import (
	"time"
)

// Theme represents a quiz theme
type Theme struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	Name           string     `json:"name" gorm:"not null;size:100"`
	Description    string     `json:"description" gorm:"type:text"`
	Icon           string     `json:"icon" gorm:"size:50"`
	Color          string     `json:"color" gorm:"size:20"`
	IsActive       bool       `json:"is_active" gorm:"default:true"`
	IsDefault      bool       `json:"is_default" gorm:"default:false"`
	IsFileBacked   bool       `json:"is_file_backed" gorm:"default:false"`
	IsPublic       bool       `json:"is_public" gorm:"default:true"`
	CreatedByGuest bool       `json:"created_by_guest" gorm:"default:false"`
	CreatedBy      *uint      `json:"created_by" gorm:"index"`
	Creator        *User      `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	UnlockCost     int        `json:"unlock_cost" gorm:"default:0"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Questions      []Question `json:"questions,omitempty" gorm:"foreignKey:ThemeID"`
}

// Question represents a U Bible quiz question
type Question struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ThemeID       uint      `json:"theme_id" gorm:"not null;index"`
	Theme         *Theme    `json:"theme,omitempty" gorm:"foreignKey:ThemeID"`
	ThemeName     string    `json:"theme_name" gorm:"size:100;index"`
	Text          string    `json:"text" gorm:"not null;type:text"`
	CorrectAnswer string    `json:"correct_answer" gorm:"not null;size:500"`
	WrongAnswers  string    `json:"wrong_answers" gorm:"not null;type:text"`
	Reference     string    `json:"reference" gorm:"size:100"`
	Difficulty    string    `json:"difficulty" gorm:"default:'medium';size:20"`
	CreatedAt     time.Time `json:"created_at"`
}

// Attempt represents a user's quiz attempt
type Attempt struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	User           *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	ThemeID        uint      `json:"theme_id" gorm:"index"`
	Theme          *Theme    `json:"theme,omitempty" gorm:"foreignKey:ThemeID"`
	Score          int       `json:"score" gorm:"default:0"`
	CorrectAnswers int       `json:"correct_answers" gorm:"default:0"`
	TotalQuestions int       `json:"total_questions" gorm:"default:0"`
	TimeElapsed    int       `json:"time_elapsed" gorm:"default:0"` // in seconds
	IsPerfect      bool      `json:"is_perfect" gorm:"default:false"`
	Difficulty     string    `json:"difficulty" gorm:"size:20"`
	XPEarned       int       `json:"xp_earned" gorm:"default:0"`
	FPEarned       int       `json:"fp_earned" gorm:"default:0"`
	CreatedAt      time.Time `json:"created_at"`
}

// Friend represents a friendship between users
type Friend struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	User      *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	FriendID  uint      `json:"friend_id" gorm:"not null;index"`
	Friend    *User     `json:"friend,omitempty" gorm:"foreignKey:FriendID"`
	CreatedAt time.Time `json:"created_at"`
}

// FriendRequest represents a pending friend request
type FriendRequest struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	FromUserID uint      `json:"from_user_id" gorm:"not null;index"`
	FromUser   *User     `json:"from_user,omitempty" gorm:"foreignKey:FromUserID"`
	ToUserID   uint      `json:"to_user_id" gorm:"not null;index"`
	ToUser     *User     `json:"to_user,omitempty" gorm:"foreignKey:ToUserID"`
	Status     string    `json:"status" gorm:"default:'pending';size:20"` // pending, accepted, rejected
	CreatedAt  time.Time `json:"created_at"`
}

// PowerUp represents a power-up item
type PowerUp struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"not null;index"`
	User        *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Type        string    `json:"type" gorm:"not null;size:50"` // hint, skip, time_boost, etc.
	Quantity    int       `json:"quantity" gorm:"default:0"`
	PurchasedAt time.Time `json:"purchased_at"`
}

// TableName methods for custom table names (optional)
func (Theme) TableName() string {
	return "themes"
}

func (Question) TableName() string {
	return "questions"
}

func (Attempt) TableName() string {
	return "attempts"
}

func (Friend) TableName() string {
	return "friends"
}

func (FriendRequest) TableName() string {
	return "friend_requests"
}

func (PowerUp) TableName() string {
	return "power_ups"
}

// models/challenge.go - Challenge System Data Models
package models

import (
	"time"
)

// Challenge status constants
type ChallengeStatus string

const (
	ChallengeStatusPending   ChallengeStatus = "pending"
	ChallengeStatusActive    ChallengeStatus = "active"
	ChallengeStatusCompleted ChallengeStatus = "completed"
	ChallengeStatusCancelled ChallengeStatus = "cancelled"
)

// Challenge represents a team challenge/competition
type Challenge struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	TeamID          uint            `json:"team_id" gorm:"not null;index"`
	Team            *Team           `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	Name            string          `json:"name" gorm:"not null;size:100"`
	Description     string          `json:"description" gorm:"type:text"`
	ThemeID         uint            `json:"theme_id" gorm:"index"`
	Theme           *TeamTheme      `json:"theme,omitempty" gorm:"foreignKey:ThemeID"`
	NumQuestions    int             `json:"num_questions" gorm:"column:question_count;not null;default:10"` // Alias as num_questions
	TimeLimit       int             `json:"time_limit" gorm:"not null;default:30"`
	StartDate       time.Time       `json:"start_date"`
	EndDate         time.Time       `json:"end_date"`
	MinParticipants int             `json:"min_participants" gorm:"default:2"`
	MaxParticipants int             `json:"max_participants" gorm:"default:0"`
	Status          ChallengeStatus `json:"status" gorm:"not null;default:'pending';index"`
	CreatedBy       uint            `json:"created_by_user_id" gorm:"column:created_by;not null"` // JSON shows as created_by_user_id
	CreatedAt       time.Time       `json:"created_at" gorm:"not null"`
	UpdatedAt       time.Time       `json:"updated_at"`
	StartedAt       *time.Time      `json:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at"`
	Participants    []ChallengeParticipant `json:"participants,omitempty" gorm:"foreignKey:ChallengeID"`
}

// ChallengeParticipant represents a user's participation in a challenge
type ChallengeParticipant struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	ChallengeID uint       `json:"challenge_id" gorm:"not null;index"`
	Challenge   *Challenge `json:"challenge,omitempty" gorm:"foreignKey:ChallengeID"`
	UserID      uint       `json:"user_id" gorm:"not null;index"`
	User        *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	JoinedAt    time.Time  `json:"joined_at" gorm:"not null"`
	Score       int        `json:"score" gorm:"default:0"`
	TimeSpent   int        `json:"time_spent" gorm:"default:0"`
	Correct     int        `json:"correct" gorm:"default:0"`
	Incorrect   int        `json:"incorrect" gorm:"default:0"`
	CompletedAt *time.Time `json:"completed_at"`
	Status      string     `json:"status" gorm:"default:'joined'"`
}

func (Challenge) TableName() string {
	return "challenges"
}

func (ChallengeParticipant) TableName() string {
	return "challenge_participants"
}

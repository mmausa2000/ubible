// models/user.go
package models

import (
	"time"
)

type User struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Username    string     `gorm:"uniqueIndex;not null" json:"username"`
	Email       *string    `gorm:"uniqueIndex" json:"email,omitempty"`
	Password    string     `gorm:"not null" json:"-"`
	DisplayName string     `json:"display_name"`
	Avatar      string     `json:"avatar"`
	Bio         string     `json:"bio"`
	IsGuest     bool       `gorm:"default:false" json:"is_guest"`
	IsAdmin     bool       `gorm:"default:false" json:"is_admin"`
	IsBanned    bool       `gorm:"default:false" json:"is_banned"`
	EmailPublic bool       `gorm:"default:false" json:"email_public"`
	
	// Progression
	Level       int        `gorm:"default:1" json:"level"`
	XP          int        `gorm:"default:0" json:"xp"`
	FaithPoints int        `gorm:"default:0" json:"faith_points"`
	
	// Stats
	TotalGames   int       `gorm:"default:0" json:"total_games"`
	Wins         int       `gorm:"default:0" json:"wins"`
	Losses       int       `gorm:"default:0" json:"losses"`
	PerfectGames int       `gorm:"default:0" json:"perfect_games"`
	CurrentStreak int      `gorm:"default:0" json:"current_streak"`
	BestStreak   int       `gorm:"default:0" json:"best_streak"`
	
	// Power-ups
	PowerUp5050       int `gorm:"default:3" json:"powerup_5050"`
	PowerUpTimeFreeze int `gorm:"default:3" json:"powerup_time_freeze"`
	PowerUpHint       int `gorm:"default:3" json:"powerup_hint"`
	PowerUpSkip       int `gorm:"default:1" json:"powerup_skip"`
	PowerUpDouble     int `gorm:"default:2" json:"powerup_double"`
	
	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastLogin time.Time  `json:"last_login"`
	
	// Relationships
	Achievements []UserAchievement `gorm:"foreignKey:UserID" json:"achievements,omitempty"`
	Attempts     []Attempt         `gorm:"foreignKey:UserID" json:"attempts,omitempty"`
}

type UserAchievement struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"not null;index" json:"user_id"`
	AchievementID uint      `gorm:"not null;index" json:"achievement_id"`
	UnlockedAt    time.Time `json:"unlocked_at"`
	
	// Relationships
	User        User        `gorm:"foreignKey:UserID" json:"-"`
	Achievement Achievement `gorm:"foreignKey:AchievementID" json:"achievement,omitempty"`
}

// models/achievement.go
package models

import "time"

type Achievement struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null;uniqueIndex" json:"name"`
	Description string `gorm:"not null" json:"description"`
	Category    string `gorm:"not null;index" json:"category"` // Speed, Accuracy, Streak, Social, Theme, Special
	Tier        string `gorm:"not null" json:"tier"`           // Beginner, Intermediate, Advanced, Elite
	Icon        string `json:"icon"`
	
	// Rewards
	XPReward    int    `gorm:"default:0" json:"xp_reward"`
	FPReward    int    `gorm:"default:0" json:"fp_reward"`
	PowerUpReward   string `json:"powerup_reward,omitempty"`
	PowerUpQuantity int    `json:"powerup_quantity,omitempty"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

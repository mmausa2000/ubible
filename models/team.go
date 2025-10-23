// models/team.go
package models

import "time"

type Team struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"not null;size:100"`
	Description string       `json:"description" gorm:"type:text"`
	TeamCode    string       `json:"team_code" gorm:"unique;size:10"`
	IsPublic    bool         `json:"is_public" gorm:"default:true"`
	IsActive    bool         `json:"is_active" gorm:"default:true;index"`
	CreatorID   uint         `json:"creator_id" gorm:"not null"`
	Creator     *User        `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Members     []TeamMember `json:"members,omitempty" gorm:"foreignKey:TeamID"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (Team) TableName() string {
	return "teams"
}

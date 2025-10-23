// models/team_member.go
package models

import "time"

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
)

type TeamMember struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	TeamID        uint      `json:"team_id" gorm:"not null;index"`
	Team          *Team     `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	UserID        uint      `json:"user_id" gorm:"not null;index"`
	User          *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role          TeamRole  `json:"role" gorm:"not null;default:'member'"`
	JoinedAt      time.Time `json:"joined_at" gorm:"not null"`
	IsActive      bool      `json:"is_active" gorm:"default:true;index"`
	TotalScore    int       `json:"total_score" gorm:"default:0"`
	QuizzesPlayed int       `json:"quizzes_played" gorm:"default:0"`
	LastActive    time.Time `json:"last_active"`
}

func (TeamMember) TableName() string {
	return "team_members"
}

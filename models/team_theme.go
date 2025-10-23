// models/team_theme.go - Team Theme Data Model
package models

import (
	"time"

	"github.com/lib/pq"
)

// TeamTheme represents a shared verse collection within a team
type TeamTheme struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	TeamID      uint           `json:"team_id" gorm:"not null;index"`
	Team        *Team          `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	Name        string         `json:"name" gorm:"not null;size:100"`
	Description string         `json:"description" gorm:"type:text"`
	VerseFile   string         `json:"verse_file" gorm:"not null;size:255"` // Path to verse file or content
	IsPublic    bool           `json:"is_public" gorm:"default:false;index"`
	Tags        pq.StringArray `json:"tags" gorm:"type:text[]"` // Categories/tags for filtering
	CreatedBy   uint           `json:"created_by" gorm:"not null"`
	Creator     *User          `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	CreatedAt   time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time      `json:"updated_at"`
	UsageCount  int            `json:"usage_count" gorm:"default:0"` // How many times used
	LastUsed    *time.Time     `json:"last_used"`
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
}

// TableName specifies the table name for TeamTheme
func (TeamTheme) TableName() string {
	return "team_themes"
}

// HasTag checks if theme has a specific tag
func (t *TeamTheme) HasTag(tag string) bool {
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag if it doesn't exist
func (t *TeamTheme) AddTag(tag string) {
	if !t.HasTag(tag) {
		t.Tags = append(t.Tags, tag)
	}
}

// RemoveTag removes a tag if it exists
func (t *TeamTheme) RemoveTag(tag string) {
	for i, existingTag := range t.Tags {
		if existingTag == tag {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			return
		}
	}
}

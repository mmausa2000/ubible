// models/team_theme.go - Team Theme Data Model
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// StringArray is a custom type for PostgreSQL text[] arrays
type StringArray []string

// Scan implements sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan StringArray: value is not []byte")
	}

	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "{}", nil
	}
	return json.Marshal(s)
}

// TeamTheme represents a shared verse collection within a team
type TeamTheme struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	TeamID      uint        `json:"team_id" gorm:"not null;index"`
	Team        *Team       `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	Name        string      `json:"name" gorm:"not null;size:100"`
	Description string      `json:"description" gorm:"type:text"`
	VerseFile   string      `json:"verse_file" gorm:"not null;size:255"` // Path to verse file or content
	IsPublic    bool        `json:"is_public" gorm:"default:false;index"`
	Tags        StringArray `json:"tags" gorm:"type:text[]"` // Categories/tags for filtering
	CreatedBy   uint        `json:"created_by_user_id" gorm:"column:created_by;not null"` // JSON shows as created_by_user_id
	Creator     *User       `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	CreatedAt   time.Time   `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time   `json:"updated_at"`
	UsageCount  int         `json:"usage_count" gorm:"default:0"` // How many times used
	LastUsed    *time.Time  `json:"last_used"`
	IsActive    bool        `json:"is_active" gorm:"default:true;index"`
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

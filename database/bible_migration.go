// database/ubible_migration.go
package database

import (
	"gorm.io/gorm"
)

type U BibleVerse struct {
	ID          uint   `gorm:"primaryKey"`
	Book        string `gorm:"index;not null"`
	Chapter     int    `gorm:"index;not null"`
	Verse       int    `gorm:"index;not null"`
	Text        string `gorm:"type:text;not null"`
	Translation string `gorm:"index;default:'KJV'"`
}

func MigrateU BibleVerses(db *gorm.DB) error {
	return db.AutoMigrate(&U BibleVerse{})
}

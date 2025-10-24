package services

import (
	"log"
	"ubible/database"
	"ubible/models"
)

// CleanupService handles background cleanup tasks
type CleanupService struct{}

var cleanupService *CleanupService

// InitCleanupService initializes the singleton cleanup service.
func InitCleanupService() {
	cleanupService = &CleanupService{}
}

// GetCleanupService returns the initialized cleanup service.
func GetCleanupService() *CleanupService {
	return cleanupService
}

// Start starts the cleanup worker(s). No-op stub for now.
func (s *CleanupService) Start() {
	// TODO: implement background cleanup logic
}

// Stop stops the cleanup worker(s). No-op stub for now.
func (s *CleanupService) Stop() {
	// TODO: implement cleanup shutdown logic
}

// CleanupDeletedThemes removes themes with no questions (but preserves file-backed themes)
func (s *CleanupService) CleanupDeletedThemes() error {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	// Find all user-created themes (IsFileBacked = false) with no questions
	var emptyThemes []models.Theme
	if err := db.Where("is_file_backed = ? AND id NOT IN (SELECT DISTINCT theme_id FROM questions)", false).
		Find(&emptyThemes).Error; err != nil {
		log.Printf("Error finding empty themes: %v", err)
		return err
	}

	if len(emptyThemes) == 0 {
		log.Println("No empty user-created themes to cleanup")
		return nil
	}

	// Delete empty themes
	themeIDs := make([]uint, len(emptyThemes))
	for i, theme := range emptyThemes {
		themeIDs[i] = theme.ID
	}

	if err := db.Delete(&emptyThemes).Error; err != nil {
		log.Printf("Error deleting empty themes: %v", err)
		return err
	}

	log.Printf("✅ Cleaned up %d empty user-created themes", len(emptyThemes))
	return nil
}

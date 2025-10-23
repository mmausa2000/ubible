// database/db.go - Database Connection (Fixed)
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// InitDB initializes the database connection
func InitDB() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/ubible_quiz.db"
	}

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("✅ Database connected successfully")

	// Run migrations (no parameter needed)
	RunMigrations()
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	if db == nil {
		log.Fatal("Database not initialized. Call InitDB() first.")
	}
	return db
}

// CloseDB closes the database connection
func CloseDB() error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %v", err)
	}

	log.Println("Database connection closed")
	return nil
}

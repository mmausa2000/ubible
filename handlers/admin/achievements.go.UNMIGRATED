package admin

import (
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/fiber/v2"
)

// GetAchievements returns all achievements
func GetAchievements(c *fiber.Ctx) error {
	db := database.GetDB()
	
	var achievements []models.Achievement
	if err := db.Find(&achievements).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch achievements",
		})
	}
	
	return c.JSON(achievements)
}

// CreateAchievement creates a new achievement
func CreateAchievement(c *fiber.Ctx) error {
	db := database.GetDB()
	
	var achievement models.Achievement
	if err := c.BodyParser(&achievement); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	if err := db.Create(&achievement).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create achievement",
		})
	}
	
	return c.Status(201).JSON(achievement)
}

// UpdateAchievement updates an existing achievement
func UpdateAchievement(c *fiber.Ctx) error {
	db := database.GetDB()
	id := c.Params("id")
	
	var achievement models.Achievement
	if err := db.First(&achievement, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Achievement not found",
		})
	}
	
	if err := c.BodyParser(&achievement); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	if err := db.Save(&achievement).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update achievement",
		})
	}
	
	return c.JSON(achievement)
}

// DeleteAchievement deletes an achievement
func DeleteAchievement(c *fiber.Ctx) error {
	db := database.GetDB()
	id := c.Params("id")
	
	if err := db.Delete(&models.Achievement{}, id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete achievement",
		})
	}
	
	return c.JSON(fiber.Map{
		"message": "Achievement deleted successfully",
	})
}

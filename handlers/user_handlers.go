package handlers

import (
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"github.com/gofiber/fiber/v2"
)

func GetCurrentUser(c *fiber.Ctx) error {
	userID, _ := middleware.GetUserID(c)
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}
	return c.JSON(fiber.Map{"success": true, "user": user})
}

func UpdateCurrentUser(c *fiber.Ctx) error {
	userID, _ := middleware.GetUserID(c)
	var req map[string]interface{}
	c.BodyParser(&req)
	db := database.GetDB()
	db.Model(&models.User{}).Where("id = ?", userID).Updates(req)
	return c.JSON(fiber.Map{"success": true})
}

func GetUserStats(c *fiber.Ctx) error {
	userID, _ := middleware.GetUserID(c)
	db := database.GetDB()
	var user models.User
	db.First(&user, userID)
	return c.JSON(fiber.Map{"success": true, "stats": user})
}

func SearchUsers(c *fiber.Ctx) error {
	query := c.Query("q")
	db := database.GetDB()
	var users []models.User
	db.Where("username LIKE ?", "%"+query+"%").Limit(20).Find(&users)
	return c.JSON(fiber.Map{"success": true, "users": users})
}

func GetUserProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}
	return c.JSON(fiber.Map{"success": true, "user": user})
}

func GetGameHistory(c *fiber.Ctx) error {
	userID, _ := middleware.GetUserID(c)
	db := database.GetDB()
	var attempts []models.Attempt
	db.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&attempts)
	return c.JSON(fiber.Map{"success": true, "history": attempts})
}

func GetFriends(c *fiber.Ctx) error {
	userID, _ := middleware.GetUserID(c)
	db := database.GetDB()
	var friends []models.Friend
	db.Preload("Friend").Where("user_id = ?", userID).Find(&friends)
	return c.JSON(fiber.Map{"success": true, "friends": friends})
}

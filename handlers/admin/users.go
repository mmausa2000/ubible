package admin

import (
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// GetUsers returns all users with pagination
func GetUsers(c *fiber.Ctx) error {
	db := database.GetDB()

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	search := c.Query("search", "")

	offset := (page - 1) * limit

	var users []models.User
	var total int64

	query := db.Model(&models.User{})

	// Apply search filter if provided
	if search != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	query.Count(&total)

	// Get paginated users
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}

	return c.JSON(fiber.Map{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUser returns a single user by ID
func GetUser(c *fiber.Ctx) error {
	db := database.GetDB()
	id := c.Params("id")

	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user)
}

// UpdateUser updates a user's information
func UpdateUser(c *fiber.Ctx) error {
	db := database.GetDB()
	id := c.Params("id")

	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Parse update data
	var updateData struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		Level       int    `json:"level"`
		XP          int    `json:"xp"`
		FaithPoints int    `json:"faith_points"`
		IsAdmin     bool   `json:"is_admin"`
		IsBanned    bool   `json:"is_banned"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update fields
	if updateData.Username != "" {
		user.Username = updateData.Username
	}
	if updateData.Email != "" {
		email := updateData.Email
		user.Email = &email
	}
	if updateData.Level > 0 {
		user.Level = updateData.Level
	}
	if updateData.XP >= 0 {
		user.XP = updateData.XP
	}
	if updateData.FaithPoints >= 0 {
		user.FaithPoints = updateData.FaithPoints
	}
	user.IsAdmin = updateData.IsAdmin
	user.IsBanned = updateData.IsBanned

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	return c.JSON(user)
}

// DeleteUser deletes a user
func DeleteUser(c *fiber.Ctx) error {
	db := database.GetDB()
	id := c.Params("id")

	// Check if user exists
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Prevent deleting admin users
	if user.IsAdmin {
		return c.Status(403).JSON(fiber.Map{
			"error": "Cannot delete admin users",
		})
	}

	if err := db.Delete(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// BanUser bans or unbans a user
func BanUser(c *fiber.Ctx) error {
	db := database.GetDB()
	id := c.Params("id")

	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var banData struct {
		IsBanned bool `json:"is_banned"`
	}

	if err := c.BodyParser(&banData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user.IsBanned = banData.IsBanned

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update ban status",
		})
	}

	return c.JSON(user)
}

// ResetUserPassword resets a user's password (admin function)
func ResetUserPassword(c *fiber.Ctx) error {
	db := database.GetDB()
	id := c.Params("id")

	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var passwordData struct {
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&passwordData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(passwordData.NewPassword) < 6 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Password must be at least 6 characters",
		})
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordData.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	user.Password = string(hashedPassword)

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to reset password",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

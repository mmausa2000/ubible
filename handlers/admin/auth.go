package admin

import (
	"ubible/database"
	"ubible/models"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expires_at"`
}

// Login authenticates an admin user
func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Username and password are required",
		})
	}

	// Find admin user
	db := database.GetDB()
	var user models.User
	if err := db.Where("username = ? AND is_admin = ?", req.Username, true).First(&user).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Update last login
	user.LastLogin = time.Now()
	db.Save(&user)

	// Generate JWT token
	token, expiresAt, err := generateAdminToken(user.ID, user.Username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.JSON(LoginResponse{
		Token:     token,
		Username:  user.Username,
		ExpiresAt: expiresAt,
	})
}

// VerifyToken verifies an admin JWT token
func VerifyToken(c *fiber.Ctx) error {
	// Token is already validated by middleware
	// Just return success with user info
	return c.JSON(fiber.Map{
		"valid":    true,
		"user_id":  c.Locals("userId"),
		"username": c.Locals("username"),
		"is_admin": c.Locals("isAdmin"),
	})
}

// Logout handles admin logout (client-side token removal)
func Logout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// generateAdminToken creates a JWT token for admin users
func generateAdminToken(userID uint, username string) (string, int64, error) {
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"is_admin": true,
		"exp":      expiresAt,
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt, nil
}

// middleware/auth.go
package middleware

import (
	"os"
	"strings"
	"time"
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Missing authorization header"})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid authorization header format"})
	}

	tokenString := parts[1]
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "ubible-secret-change-in-production"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(401, "Invalid signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	exp, ok := claims["exp"].(float64)
	if !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
		return c.Status(401).JSON(fiber.Map{"error": "Token expired"})
	}

	c.Locals("userId", claims["user_id"])
	c.Locals("username", claims["username"])
	c.Locals("isGuest", claims["is_guest"])

	// Update user's last activity
	updateUserActivity(claims["user_id"])

	return c.Next()
}

func AdminAuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Missing authorization header"})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid authorization header format"})
	}

	tokenString := parts[1]
	jwtSecret := os.Getenv("JWT_SECRET")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(401, "Invalid signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	isAdmin, ok := claims["is_admin"].(bool)
	if !ok || !isAdmin {
		return c.Status(403).JSON(fiber.Map{"error": "Access denied. Admin privileges required."})
	}

	c.Locals("userId", claims["user_id"])
	c.Locals("username", claims["username"])
	c.Locals("isAdmin", true)

	return c.Next()
}

func GetUserID(c *fiber.Ctx) (uint, error) {
	userID := c.Locals("userId")
	if userID == nil {
		return 0, fiber.NewError(401, "User not authenticated")
	}

	if id, ok := userID.(float64); ok {
		return uint(id), nil
	}

	if id, ok := userID.(uint); ok {
		return id, nil
	}

	return 0, fiber.NewError(401, "Invalid user ID format")
}

func GetUsername(c *fiber.Ctx) (string, error) {
	username := c.Locals("username")
	if username == nil {
		return "", fiber.NewError(401, "User not authenticated")
	}

	if name, ok := username.(string); ok {
		return name, nil
	}

	return "", fiber.NewError(401, "Invalid username format")
}

func IsGuest(c *fiber.Ctx) bool {
	isGuest := c.Locals("isGuest")
	if isGuest == nil {
		return false
	}

	if guest, ok := isGuest.(bool); ok {
		return guest
	}

	return false
}

// WebSocketAuthMiddleware validates JWT for WebSocket connections
// Supports both Authorization header and cookies for flexibility
func WebSocketAuthMiddleware(c *fiber.Ctx) error {
	var tokenString string

	// Try Authorization header first (Bearer token)
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString = parts[1]
		}
	}

	// Fall back to cookie if no header
	if tokenString == "" {
		tokenString = c.Cookies("token")
	}

	// If no token found, allow connection but mark as guest
	if tokenString == "" {
		c.Locals("userId", nil)
		c.Locals("username", "Guest")
		c.Locals("isGuest", true)
		return c.Next()
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "ubible-secret-change-in-production"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(401, "Invalid signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		// Invalid token - treat as guest
		c.Locals("userId", nil)
		c.Locals("username", "Guest")
		c.Locals("isGuest", true)
		return c.Next()
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		// Invalid claims - treat as guest
		c.Locals("userId", nil)
		c.Locals("username", "Guest")
		c.Locals("isGuest", true)
		return c.Next()
	}

	// Check expiration
	exp, ok := claims["exp"].(float64)
	if !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
		// Expired token - treat as guest
		c.Locals("userId", nil)
		c.Locals("username", "Guest")
		c.Locals("isGuest", true)
		return c.Next()
	}

	// Valid token - extract user info
	c.Locals("userId", claims["user_id"])
	c.Locals("username", claims["username"])
	c.Locals("isGuest", claims["is_guest"])

	// Update user's last activity
	updateUserActivity(claims["user_id"])

	return c.Next()
}

// updateUserActivity updates the user's last activity timestamp
func updateUserActivity(userID interface{}) {
	if userID == nil {
		return
	}

	db := database.GetDB()
	if db == nil {
		return
	}

	// Convert userID to uint
	var id uint
	switch v := userID.(type) {
	case float64:
		id = uint(v)
	case uint:
		id = v
	default:
		return
	}

	// Update last activity timestamp
	now := time.Now()
	db.Model(&models.User{}).Where("id = ?", id).Update("last_activity", now)
}

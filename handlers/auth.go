// ~/Documents/CODING/ubible/handlers/auth.go
// Complete file with fixed GuestLogin

package handlers

import (
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GuestLoginRequest struct {
	GuestName string `json:"guest_name,omitempty"`
}

type UpgradeGuestRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool     `json:"success"`
	Token   string   `json:"token,omitempty"`
	User    UserInfo `json:"user,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type UserInfo struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IsGuest   bool      `json:"is_guest"`
	Level     int       `json:"level"`
	XP        int       `json:"xp"`
	CreatedAt time.Time `json:"created_at"`
}

// GuestLogin creates a new guest session
func GuestLogin(c *fiber.Ctx) error {
	var req GuestLoginRequest
	
	// ✅ FIX: Don't fail on empty body - Fiber will leave req empty if body is {}
	_ = c.BodyParser(&req)

	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
	}

	// Generate guest name if not provided
	guestName := req.GuestName
	if guestName == "" {
		guestName = fmt.Sprintf("Guest_%s", uuid.New().String()[:8])
	}

	// Generate unique guest email
	guestEmail := fmt.Sprintf("guest_%s@quiz.local", uuid.New().String()[:8])

	// Create guest user
	user := models.User{
		Username:  guestName,
		Email:     &guestEmail,
		Password:  "",
		IsGuest:   true,
		Level:     1,
		XP:        0,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&user).Error; err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to create guest account",
		})
	}

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	return c.JSON(AuthResponse{
		Success: true,
		Token:   token,
		User: UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     email,
			IsGuest:   user.IsGuest,
			Level:     user.Level,
			XP:        user.XP,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Login authenticates a registered user
func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Username and password required",
		})
	}

	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
	}

	var user models.User
	if err := db.Where("username = ? AND is_guest = ?", req.Username, false).First(&user).Error; err != nil {
		return c.Status(401).JSON(AuthResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(AuthResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
	}

	// Update last login
	db.Model(&user).Update("last_login", time.Now())

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	return c.JSON(AuthResponse{
		Success: true,
		Token:   token,
		User: UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     email,
			IsGuest:   user.IsGuest,
			Level:     user.Level,
			XP:        user.XP,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Register creates a new user account
func Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Username and password required",
		})
	}

	if len(req.Password) < 6 {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Password must be at least 6 characters",
		})
	}

	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
	}

	// Check if username already exists
	var existingUser models.User
	if err := db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Username already taken",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to hash password",
		})
	}

	// Create new user
	user := models.User{
		Username:  req.Username,
		Email:     &req.Email,
		Password:  string(hashedPassword),
		IsGuest:   false,
		Level:     1,
		XP:        0,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&user).Error; err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to create account",
		})
	}

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
	}

	return c.JSON(AuthResponse{
		Success: true,
		Token:   token,
		User: UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     req.Email,
			IsGuest:   user.IsGuest,
			Level:     user.Level,
			XP:        user.XP,
			CreatedAt: user.CreatedAt,
		},
	})
}

// UpgradeGuest converts a guest account to a registered account
func UpgradeGuest(c *fiber.Ctx) error {
	// Get user ID from JWT
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(401).JSON(AuthResponse{
			Success: false,
			Error:   "Unauthorized",
		})
	}

	var req UpgradeGuestRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Username and password required",
		})
	}

	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
	}

	// Get guest user
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(AuthResponse{
			Success: false,
			Error:   "User not found",
		})
	}

	// Verify it's a guest account
	if !user.IsGuest {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Account is already registered",
		})
	}

	// Check if username already exists
	var existingUser models.User
	if err := db.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser).Error; err == nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Error:   "Username already taken",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to hash password",
		})
	}

	// Update user
	if err := db.Model(&user).Updates(map[string]interface{}{
		"username":   req.Username,
		"email":      req.Email,
		"password":   string(hashedPassword),
		"is_guest":   false,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to upgrade account",
		})
	}

	// Reload user
	db.First(&user, userID)

	// Generate new JWT token
	token, err := generateToken(user)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	return c.JSON(AuthResponse{
		Success: true,
		Token:   token,
		User: UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     email,
			IsGuest:   user.IsGuest,
			Level:     user.Level,
			XP:        user.XP,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Helper functions

func generateToken(user models.User) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "ubible-secret-change-in-production"
	}

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"is_guest": user.IsGuest,
		"exp":      time.Now().Add(time.Hour * 720).Unix(), // 30 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}
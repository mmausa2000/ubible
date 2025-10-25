// ~/Documents/CODING/ubible/handlers/auth.go
package handlers

import (
	"fmt"
	"net/http"
	"os"
	"time"
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"ubible/utils"

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
func GuestLogin(w http.ResponseWriter, r *http.Request) {
	var req GuestLoginRequest
	// Don't fail on empty body
	_ = utils.ParseJSON(r, &req)

	db := database.GetDB()
	if db == nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
		return
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
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to create guest account",
		})
		return
	}

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	utils.JSON(w, http.StatusOK, AuthResponse{
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
func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Username == "" || req.Password == "" {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Username and password required",
		})
		return
	}

	db := database.GetDB()
	if db == nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
		return
	}

	var user models.User
	if err := db.Where("LOWER(username) = LOWER(?) AND is_guest = ?", req.Username, false).First(&user).Error; err != nil {
		utils.JSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.JSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Update last login
	db.Model(&user).Update("last_login", time.Now())

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	utils.JSON(w, http.StatusOK, AuthResponse{
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
func Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Username and password required",
		})
		return
	}

	if len(req.Password) < 6 {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Password must be at least 6 characters",
		})
		return
	}

	db := database.GetDB()
	if db == nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
		return
	}

	// Check if username already exists
	var existingUser models.User
	if err := db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Username already taken",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to hash password",
		})
		return
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
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to create account",
		})
		return
	}

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	utils.JSON(w, http.StatusOK, AuthResponse{
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
func UpgradeGuest(w http.ResponseWriter, r *http.Request) {
	// Get user ID from JWT
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	var req UpgradeGuestRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" || req.Email == "" {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Username, email, and password are required",
		})
		return
	}

	if len(req.Password) < 6 {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Password must be at least 6 characters",
		})
		return
	}

	db := database.GetDB()
	if db == nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Database not available",
		})
		return
	}

	// Get guest user
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.JSON(w, http.StatusNotFound, AuthResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	// Verify it's a guest account
	if !user.IsGuest {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Account is already registered",
		})
		return
	}

	// Check if username already exists
	var existingUser models.User
	if err := db.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser).Error; err == nil {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Username already taken",
		})
		return
	}

	// Check if email already exists
	var emailUser models.User
	if err := db.Where("email = ? AND id != ?", req.Email, userID).First(&emailUser).Error; err == nil {
		utils.JSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Email already in use",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to hash password",
		})
		return
	}

	// Update user
	user.Username = req.Username
	user.Email = &req.Email
	user.Password = string(hashedPassword)
	user.IsGuest = false

	if err := db.Save(&user).Error; err != nil {
		fmt.Printf("Database error during upgrade: %v\n", err)
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to upgrade account. Please try again or contact support.",
		})
		return
	}

	// Reload user
	db.First(&user, userID)

	// Generate new JWT token
	token, err := generateToken(user)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	utils.JSON(w, http.StatusOK, AuthResponse{
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

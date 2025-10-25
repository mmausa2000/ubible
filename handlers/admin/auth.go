package admin

import (
	"net/http"
	"os"
	"time"
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"ubible/utils"

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
func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		utils.JSONError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Find admin user
	db := database.GetDB()
	var user models.User
	if err := db.Where("username = ? AND is_admin = ?", req.Username, true).First(&user).Error; err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Update last login
	user.LastLogin = time.Now()
	db.Save(&user)

	// Generate JWT token
	token, expiresAt, err := generateAdminToken(user.ID, user.Username)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	utils.JSON(w, http.StatusOK, LoginResponse{
		Token:     token,
		Username:  user.Username,
		ExpiresAt: expiresAt,
	})
}

// VerifyToken verifies an admin JWT token
func VerifyToken(w http.ResponseWriter, r *http.Request) {
	// Token is already validated by middleware
	// Just return success with user info
	userID, _ := middleware.GetUserID(r)
	username, _ := middleware.GetUsername(r)
	isAdmin := r.Context().Value(middleware.IsAdminKey)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"valid":    true,
		"user_id":  userID,
		"username": username,
		"is_admin": isAdmin,
	})
}

// Logout handles admin logout (client-side token removal)
func Logout(w http.ResponseWriter, r *http.Request) {
	utils.JSON(w, http.StatusOK, map[string]interface{}{
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

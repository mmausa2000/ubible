package handlers

import (
	"net/http"
	"strconv"
	"ubible/database"
	"ubible/models"
	"ubible/utils"
)

// GetUsers returns all users with pagination
func GetUsers(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()

	// Get pagination parameters
	pageStr := utils.Query(r, "page", "1")
	limitStr := utils.Query(r, "limit", "20")
	search := utils.Query(r, "search", "")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

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
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUser returns a single user by ID
func GetUser(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	id := r.PathValue("id")

	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.JSONError(w, http.StatusNotFound, "User not found")
		return
	}

	utils.JSON(w, http.StatusOK, user)
}

// UpdateUser updates a user's information
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	id := r.PathValue("id")

	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.JSONError(w, http.StatusNotFound, "User not found")
		return
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

	if err := utils.ParseJSON(r, &updateData); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
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
		utils.JSONError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	utils.JSON(w, http.StatusOK, user)
}

// DeleteUser deletes a user
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	id := r.PathValue("id")

	// Check if user exists
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.JSONError(w, http.StatusNotFound, "User not found")
		return
	}

	// Prevent deleting admin users
	if user.IsAdmin {
		utils.JSONError(w, http.StatusForbidden, "Cannot delete admin users")
		return
	}

	if err := db.Delete(&user).Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "User deleted successfully",
	})
}

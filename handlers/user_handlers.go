package handlers

import (
	"net/http"
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"ubible/utils"
)

func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.JSONError(w, http.StatusNotFound, "User not found")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "user": user})
}

func UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req map[string]interface{}
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	db := database.GetDB()
	db.Model(&models.User{}).Where("id = ?", userID).Updates(req)

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func GetUserStats(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	db := database.GetDB()
	var user models.User
	db.First(&user, userID)

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "stats": user})
}

func SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := utils.Query(r, "q", "")
	db := database.GetDB()
	var users []models.User
	db.Where("username LIKE ?", "%"+query+"%").Limit(20).Find(&users)

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "users": users})
}

func GetUserProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.JSONError(w, http.StatusNotFound, "User not found")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "user": user})
}

func GetGameHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	db := database.GetDB()
	var attempts []models.Attempt
	db.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&attempts)

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "history": attempts})
}

func GetFriends(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	db := database.GetDB()
	var friends []models.Friend
	db.Preload("Friend").Where("user_id = ?", userID).Find(&friends)

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "friends": friends})
}

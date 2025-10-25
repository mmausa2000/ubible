// handlers/stubs.go
package handlers

import (
	"net/http"
	"time"
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"ubible/utils"
)

// ============ FRIEND HANDLERS ============

func SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		FriendID uint `json:"friend_id"`
	}
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	db := database.GetDB()
	friendRequest := models.FriendRequest{
		FromUserID: userID,
		ToUserID:   req.FriendID,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	if err := db.Create(&friendRequest).Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to send request")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Friend request sent"})
}

func AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		RequestID uint `json:"request_id"`
	}
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	db := database.GetDB()

	// Update friend request
	if err := db.Model(&models.FriendRequest{}).Where("id = ? AND to_user_id = ?", req.RequestID, userID).Update("status", "accepted").Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to accept request")
		return
	}

	var friendRequest models.FriendRequest
	db.First(&friendRequest, req.RequestID)

	// Create friend relationship
	friend1 := models.Friend{
		UserID:    userID,
		FriendID:  friendRequest.FromUserID,
		CreatedAt: time.Now(),
	}
	friend2 := models.Friend{
		UserID:    friendRequest.FromUserID,
		FriendID:  userID,
		CreatedAt: time.Now(),
	}

	db.Create(&friend1)
	db.Create(&friend2)

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Friend request accepted"})
}

func RejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		RequestID uint `json:"request_id"`
	}
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	db := database.GetDB()
	if err := db.Model(&models.FriendRequest{}).Where("id = ? AND to_user_id = ?", req.RequestID, userID).Update("status", "rejected").Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to reject request")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Friend request rejected"})
}

// ============ WEBSOCKET HANDLER ============

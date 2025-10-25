package admin

import (
	"net/http"
	"ubible/database"
	"ubible/models"
	"ubible/utils"
)

// GetAchievements returns all achievements
func GetAchievements(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()

	var achievements []models.Achievement
	if err := db.Find(&achievements).Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch achievements")
		return
	}

	utils.JSON(w, http.StatusOK, achievements)
}

// CreateAchievement creates a new achievement
func CreateAchievement(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()

	var achievement models.Achievement
	if err := utils.ParseJSON(r, &achievement); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := db.Create(&achievement).Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to create achievement")
		return
	}

	utils.JSON(w, http.StatusCreated, achievement)
}

// UpdateAchievement updates an existing achievement
func UpdateAchievement(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	id := r.PathValue("id")

	var achievement models.Achievement
	if err := db.First(&achievement, id).Error; err != nil {
		utils.JSONError(w, http.StatusNotFound, "Achievement not found")
		return
	}

	if err := utils.ParseJSON(r, &achievement); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := db.Save(&achievement).Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to update achievement")
		return
	}

	utils.JSON(w, http.StatusOK, achievement)
}

// DeleteAchievement deletes an achievement
func DeleteAchievement(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	id := r.PathValue("id")

	if err := db.Delete(&models.Achievement{}, id).Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to delete achievement")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Achievement deleted successfully",
	})
}

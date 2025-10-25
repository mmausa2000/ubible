// handlers/leaderboard.go (migrated to net/http)
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"ubible/database"
	"ubible/models"
)

// writeJSON is a small helper for this file.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// GetLeaderboardHTTP returns the global leaderboard
// GET /api/leaderboard?category=level&limit=100&offset=0
func GetLeaderboardHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	category := q.Get("category")
	if category == "" {
		category = "level"
	}
	limit := clampInt(parseIntDefault(q.Get("limit"), 100), 1, 100)
	offset := maxInt(parseIntDefault(q.Get("offset"), 0), 0)

	db := database.GetDB()
	var users []models.User

	var orderBy string
	switch category {
	case "xp":
		orderBy = "xp DESC, wins DESC, total_games ASC"
	case "level":
		orderBy = "level DESC, xp DESC"
	case "wins":
		orderBy = "wins DESC"
	case "streak":
		orderBy = "best_streak DESC"
	case "accuracy":
		orderBy = "perfect_games DESC, wins DESC"
	default:
		orderBy = "xp DESC, wins DESC, total_games ASC"
	}

	if err := db.Where("is_guest = ?", false).
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "Failed to fetch leaderboard",
		})
		return
	}

	// Remove sensitive data
	for i := range users {
		users[i].Password = ""
		users[i].Email = nil
	}

	var total int64
	db.Model(&models.User{}).Where("is_guest = ?", false).Count(&total)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"users":    users,
		"category": category,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetSeasonLeaderboardHTTP returns the seasonal leaderboard
// GET /api/leaderboard/season?limit=100&offset=0
func GetSeasonLeaderboardHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := clampInt(parseIntDefault(q.Get("limit"), 100), 1, 100)
	offset := maxInt(parseIntDefault(q.Get("offset"), 0), 0)

	db := database.GetDB()

	type LeaderboardEntry struct {
		UserID        uint    `json:"user_id"`
		Username      string  `json:"username"`
		Avatar        string  `json:"avatar"`
		Level         int     `json:"level"`
		TotalGames    int     `json:"total_games"`
		Wins          int     `json:"wins"`
		WinRate       float64 `json:"win_rate"`
		CurrentStreak int     `json:"current_streak"`
	}

	var entries []LeaderboardEntry

	db.Raw(`
		SELECT 
			id as user_id,
			username,
			avatar,
			level,
			total_games,
			wins,
			CASE WHEN total_games > 0 THEN (CAST(wins AS FLOAT) / total_games * 100) ELSE 0 END as win_rate,
			current_streak
		FROM users
		WHERE is_guest = false
		ORDER BY wins DESC, level DESC
		LIMIT ? OFFSET ?
	`, limit, offset).Scan(&entries)

	var total int64
	db.Model(&models.User{}).Where("is_guest = ?", false).Count(&total)

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetUserRankHTTP returns a user's rank in the leaderboard
// GET /api/leaderboard/user/{id}?category=level
func GetUserRankHTTP(w http.ResponseWriter, r *http.Request) {
	// naive param extraction: trim prefix
	path := strings.TrimPrefix(r.URL.Path, "/api/leaderboard/user/")
	userID := path
	if userID == "" || userID == "/api/leaderboard/user" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing user id"})
		return
	}
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "level"
	}

	db := database.GetDB()
	var user models.User

	if err := db.Where("id = ? OR username = ?", userID, userID).First(&user).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "User not found"})
		return
	}

	var rank int64
	var query string

	switch category {
	case "xp":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND (xp > ? OR (xp = ? AND wins > ?) OR (xp = ? AND wins = ? AND total_games < ?))"
		db.Raw(query, user.XP, user.XP, user.Wins, user.XP, user.Wins, user.TotalGames).Scan(&rank)
	case "level":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND (level > ? OR (level = ? AND xp > ?))"
		db.Raw(query, user.Level, user.Level, user.XP).Scan(&rank)
	case "wins":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND wins > ?"
		db.Raw(query, user.Wins).Scan(&rank)
	case "streak":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND best_streak > ?"
		db.Raw(query, user.BestStreak).Scan(&rank)
	case "accuracy":
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND (perfect_games > ? OR (perfect_games = ? AND wins > ?))"
		db.Raw(query, user.PerfectGames, user.PerfectGames, user.Wins).Scan(&rank)
	default:
		query = "SELECT COUNT(*) + 1 FROM users WHERE is_guest = false AND (xp > ? OR (xp = ? AND wins > ?) OR (xp = ? AND wins = ? AND total_games < ?))"
		db.Raw(query, user.XP, user.XP, user.Wins, user.XP, user.Wins, user.TotalGames).Scan(&rank)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"user_id":  user.ID,
		"username": user.Username,
		"rank":     rank,
		"category": category,
	})
}

// GetLeaderboardAroundUserHTTP returns entries around a specific user
// GET /api/leaderboard/around/{id}?category=level&context=5
func GetLeaderboardAroundUserHTTP(w http.ResponseWriter, r *http.Request) {
	// naive param extraction
	path := strings.TrimPrefix(r.URL.Path, "/api/leaderboard/around/")
	userID := path
	if userID == "" || userID == "/api/leaderboard/around" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing user id"})
		return
	}
	q := r.URL.Query()
	category := q.Get("category")
	if category == "" {
		category = "level"
	}
	contextN := clampInt(parseIntDefault(q.Get("context"), 5), 1, 20)

	db := database.GetDB()
	var user models.User

	if err := db.Where("id = ? OR username = ?", userID, userID).First(&user).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "User not found"})
		return
	}

	var users []models.User
	var orderBy string

	switch category {
	case "xp":
		orderBy = "xp DESC, wins DESC, total_games ASC"
	case "level":
		orderBy = "level DESC, xp DESC"
	case "wins":
		orderBy = "wins DESC"
	case "streak":
		orderBy = "best_streak DESC"
	case "accuracy":
		orderBy = "perfect_games DESC, wins DESC"
	default:
		orderBy = "xp DESC, wins DESC, total_games ASC"
	}

	db.Raw(`
		WITH ranked_users AS (
			SELECT *, ROW_NUMBER() OVER (ORDER BY `+orderBy+`) as rank
			FROM users
			WHERE is_guest = false
		),
		target_rank AS (
			SELECT rank FROM ranked_users WHERE id = ?
		)
		SELECT * FROM ranked_users
		WHERE rank BETWEEN (SELECT rank FROM target_rank) - ? AND (SELECT rank FROM target_rank) + ?
		ORDER BY rank
	`, user.ID, contextN, contextN).Scan(&users)

	for i := range users {
		users[i].Password = ""
		users[i].Email = nil
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"users":       users,
		"target_user": user.ID,
		"category":    category,
		"context":     contextN,
	})
}

// helpers
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

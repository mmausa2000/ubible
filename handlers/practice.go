// handlers/practice.go
package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"ubible/database"
	"ubible/middleware"
	"ubible/models"
	"ubible/utils"
)

type PracticeCard struct {
	ID        uint   `json:"id"`
	ThemeID   uint   `json:"theme_id"`
	ThemeName string `json:"theme_name"`
	Reference string `json:"reference"`
	VerseText string `json:"verse_text"`
}

// 1) Reference question: Which reference is this verse from? "…verse…"
var refQRe = regexp.MustCompile(`^Which reference is this verse from\?\s*"(.*)"\s*$`)

//  2. Completion (dash or colon; unicode or ascii ellipsis)
//     Examples: `Book c:v — "first…"` OR `Book c:v: "first..."`
var compDashColonRe = regexp.MustCompile(
	`^([1-3]?\s*[A-Za-z]+(?:\s+[A-Za-z]+)*\s+\d+:\d+)\s+[-—–:]\s*"(.*?)(?:\.{3}|…)"\s*$`,
)

// 3) Completion with prefix: Complete this verse (Book c:v): "first..."
var compPrefixRe = regexp.MustCompile(
	`(?i)^Complete this verse(?:\s*\([^)]+\))?:\s*"(.*?)(?:\.{3}|…)"\s*$`,
)

// Strip any leading `Book c:v — "` or `Book c:v: "` prefixes if we fall back
var refPrefixStripRe = regexp.MustCompile(
	`^([1-3]?\s*[A-Za-z]+(?:\s+[A-Za-z]+)*\s+\d+:\d+)\s+[-—–:]\s*"`,
)

func reconstructVerse(q models.Question) (string, bool) {
	txt := strings.TrimSpace(q.Text)

	// Reference question carries the full verse in quotes
	if m := refQRe.FindStringSubmatch(txt); len(m) > 1 {
		return strings.TrimSpace(m[1]), true
	}

	// Completion (dash/colon variant): glue first part + correct answer
	if m := compDashColonRe.FindStringSubmatch(txt); len(m) > 2 && strings.TrimSpace(q.CorrectAnswer) != "" {
		first := strings.TrimSpace(m[2])
		second := strings.TrimSpace(q.CorrectAnswer)
		return strings.TrimSpace(first + " " + second), true
	}

	// Completion (prefix variant): glue first part + correct answer
	if m := compPrefixRe.FindStringSubmatch(txt); len(m) > 1 && strings.TrimSpace(q.CorrectAnswer) != "" {
		first := strings.TrimSpace(m[1])
		second := strings.TrimSpace(q.CorrectAnswer)
		return strings.TrimSpace(first + " " + second), true
	}

	// Fallback: strip any leading reference+dash/colon if present
	if refPrefixStripRe.MatchString(txt) {
		trimmed := refPrefixStripRe.ReplaceAllString(txt, "")
		trimmed = strings.TrimSuffix(trimmed, `"`)
		return strings.TrimSpace(trimmed), true
	}

	// If the correct_answer is the same as the reference, this is a "reference identification" question
	// Skip these as they don't contain verse text
	if strings.TrimSpace(q.CorrectAnswer) == strings.TrimSpace(q.Reference) {
		return "", false
	}

	// NEW: Extract verse portion before question marks
	// Example: "In the beginning, God created the heavens and the earth. What was...?"
	// Extract: "In the beginning, God created the heavens and the earth."
	if strings.Contains(txt, "?") {
		// Look for common question starters and cut before them
		questionStarters := []string{
			" What ", " Who ", " Where ", " When ", " Why ", " How ",
			" Which ", " Did ", " Does ", " Do ", " Is ", " Are ",
			" Was ", " Were ", " Can ", " Could ", " Should ", " Would ",
		}

		// Find the FIRST occurrence of any question starter
		cutIndex := -1
		for _, starter := range questionStarters {
			if i := strings.Index(txt, starter); i > 0 {
				if cutIndex == -1 || i < cutIndex {
					cutIndex = i
				}
			}
		}

		if cutIndex > 0 {
			verse := strings.TrimSpace(txt[:cutIndex])
			// Ensure it's substantial enough and not just a reference
			if len(verse) > 20 {
				return verse, true
			}
		}

		// If no question starter found, return empty to skip this question
		return "", false
	}

	// Last resort: if text doesn't contain a question mark and looks like a verse, use it
	if len(txt) > 20 && len(txt) < 500 {
		return txt, true
	}

	return "", false
}

func GetPracticeCards(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	if db == nil {
		utils.JSONError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	themeID := utils.Query(r, "theme_id", "")
	limit, _ := strconv.Atoi(utils.Query(r, "limit", "200"))
	offset, _ := strconv.Atoi(utils.Query(r, "offset", "0"))

	var questions []models.Question
	query := db.Model(&models.Question{}).Preload("Theme")

	// If specific theme requested, use it
	if themeID != "" {
		query = query.Where("theme_id = ?", themeID)
	} else {
		// Otherwise, filter by user's selected themes if user is authenticated
		userID, err := middleware.GetUserID(r)
		if err == nil && userID > 0 {
			var user models.User
			if err := db.First(&user, userID).Error; err == nil && user.SelectedThemes != "" {
				var selectedThemes []int
				if err := json.Unmarshal([]byte(user.SelectedThemes), &selectedThemes); err == nil && len(selectedThemes) > 0 {
					query = query.Where("theme_id IN ?", selectedThemes)
				}
			}
		}
	}

	if err := query.Limit(limit).Offset(offset).Find(&questions).Error; err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch verses")
		return
	}

	cards := make([]PracticeCard, 0, len(questions))
	seen := map[string]struct{}{} // dedupe by (reference, verse_text)

	for _, q := range questions {
		themeName := q.ThemeName // Use denormalized field
		if themeName == "" && q.Theme.ID != 0 {
			themeName = q.Theme.Name // Fallback to relationship if needed
		}
		verseText, _ := reconstructVerse(q)
		if verseText == "" {
			continue
		}
		key := strings.TrimSpace(q.Reference) + "||" + verseText
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		cards = append(cards, PracticeCard{
			ID:        q.ID,
			ThemeID:   q.ThemeID,
			ThemeName: themeName,
			Reference: q.Reference,
			VerseText: verseText,
		})
	}

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"cards":   cards,
		"count":   len(cards),
	})
}

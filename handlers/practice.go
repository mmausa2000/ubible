// handlers/practice.go
package handlers

import (
	"regexp"
	"strings"
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/fiber/v2"
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

	return txt, false
}

func GetPracticeCards(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Database not available"})
	}

	themeID := c.Query("theme_id", "")
	limit := c.QueryInt("limit", 200)
	offset := c.QueryInt("offset", 0)

	var questions []models.Question
	query := db.Model(&models.Question{}).Preload("Theme")
	if themeID != "" {
		query = query.Where("theme_id = ?", themeID)
	}
	if err := query.Limit(limit).Offset(offset).Find(&questions).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Failed to fetch verses"})
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

	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Set("Content-Type", "application/json; charset=utf-8")
	return c.JSON(fiber.Map{
		"success": true,
		"cards":   cards,
		"count":   len(cards),
	})
}

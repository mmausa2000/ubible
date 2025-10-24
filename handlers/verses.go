// handlers/verses.go
package handlers

import (
	"ubible/database"
	"ubible/models"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type VerseResponse struct {
	ID            uint     `json:"id"`
	ThemeID       uint     `json:"theme_id"`
	ThemeName     string   `json:"theme_name"`
	Text          string   `json:"text"`
	CorrectAnswer string   `json:"correct_answer"`
	Options       []string `json:"options"`
	Reference     string   `json:"reference"`
	Difficulty    string   `json:"difficulty"`
}

// Normalize "Book 1:1: ..." => "Book 1:1 — ..." (supports multi-word books)
var refColonRe = regexp.MustCompile(`^([1-3]?\s*[A-Za-z]+(?:\s+[A-Za-z]+)*\s+\d+:\d+)\s*:\s+`)

// Completion detectors (both … and ...; both em dash and colon; and the "Complete this verse" format)
var completionDashOrColonRe = regexp.MustCompile(
	`^([1-3]?\s*[A-Za-z]+(?:\s+[A-Za-z]+)*\s+\d+:\d+)\s+[-—–:]\s*".*(?:\.{3}|…)"\s*$`,
)
var completionPrefixRe = regexp.MustCompile(
	`(?i)^Complete this verse(?:\s*\([^)]+\))?:\s*".*(?:\.{3}|…)"\s*$`,
)

func sanitizeDisplayText(s string) string {
	if s == "" {
		return s
	}
	ss := strings.TrimSpace(s)
	ss = refColonRe.ReplaceAllString(ss, "$1 — ")
	return ss
}

func shuffleOptions(options []string) []string {
	out := make([]string, len(options))
	copy(out, options)
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func dedupStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		vv := strings.TrimSpace(v)
		if vv == "" {
			continue
		}
		if _, ok := seen[vv]; ok {
			continue
		}
		seen[vv] = struct{}{}
		out = append(out, vv)
	}
	return out
}

func sampleStrings(source []string, n int, exclude map[string]struct{}) []string {
	candidates := make([]string, 0, len(source))
	for _, v := range source {
		vv := strings.TrimSpace(v)
		if vv == "" {
			continue
		}
		if _, skip := exclude[vv]; skip {
			continue
		}
		candidates = append(candidates, vv)
	}
	if len(candidates) == 0 || n <= 0 {
		return []string{}
	}
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	if n > len(candidates) {
		n = len(candidates)
	}
	return candidates[:n]
}

// Build exactly 4 options (correct + 3 unique distractors). Fill from DB if needed.
func buildFourOptions(db *gorm.DB, correct string, wrongAnswers []string) []string {
	exclude := map[string]struct{}{strings.TrimSpace(correct): {}}

	// Clean wrong answers: trim, dedup, remove equal to correct
	cleanWrong := make([]string, 0, len(wrongAnswers))
	for _, w := range wrongAnswers {
		wt := strings.TrimSpace(w)
		if wt == "" || wt == correct {
			continue
		}
		cleanWrong = append(cleanWrong, wt)
	}
	cleanWrong = dedupStrings(cleanWrong)

	// Pick up to 3 from provided wrong answers
	var chosen []string
	if len(cleanWrong) >= 3 {
		chosen = sampleStrings(cleanWrong, 3, exclude)
	} else {
		chosen = cleanWrong
	}

	// If still need more distractors, pull from DB distinct correct_answer values
	need := 3 - len(chosen)
	if need > 0 && db != nil {
		var pool []string
		if err := db.Model(&models.Question{}).
			Where("correct_answer <> ?", correct).
			Distinct("correct_answer").
			Pluck("correct_answer", &pool).Error; err == nil && len(pool) > 0 {
			additional := sampleStrings(pool, need, exclude)
			chosen = append(chosen, additional...)
		}
	}

	// Edge: if still not enough, pad placeholders
	for len(chosen) < 3 {
		placeholder := fmt.Sprintf("Option %d", len(chosen)+1)
		if _, exists := exclude[placeholder]; !exists {
			chosen = append(chosen, placeholder)
		} else {
			break
		}
	}

	options := append([]string{correct}, chosen...)
	return shuffleOptions(options)
}

func isCompletionQuestion(q models.Question) bool {
	t := strings.TrimSpace(q.Text)
	// Fast-path: many completion items are saved as "hard"
	if strings.EqualFold(strings.TrimSpace(q.Difficulty), "hard") {
		if completionDashOrColonRe.MatchString(t) || completionPrefixRe.MatchString(t) || strings.Contains(t, "...") || strings.Contains(t, "…") {
			return true
		}
	}
	// Pattern-based detection for non-hard labels
	if completionDashOrColonRe.MatchString(t) || completionPrefixRe.MatchString(t) {
		return true
	}
	return false
}

func GetVerses(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		log.Println("Database not initialized in GetVerses")
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Database not available"})
	}

	themeID := c.Query("theme_id", "")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	// Control which direction to return (reference|completion|all). Default: reference.
	questionType := strings.ToLower(strings.TrimSpace(c.Query("question_type", "reference")))

	if limit < 1 || limit > 200 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Limit must be between 1 and 200"})
	}
	if offset < 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Offset must be non-negative"})
	}

	var questions []models.Question
	query := db.Model(&models.Question{}).Preload("Theme")
	if themeID != "" {
		query = query.Where("theme_id = ?", themeID)
	}
	if err := query.Limit(limit).Offset(offset).Find(&questions).Error; err != nil {
		log.Printf("Error fetching verses: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Failed to fetch verses"})
	}

	verses := make([]VerseResponse, 0, len(questions))
	for _, q := range questions {
		comp := isCompletionQuestion(q)

		// Filter by requested type
		switch questionType {
		case "reference":
			if comp {
				continue
			}
		case "completion":
			if !comp {
				continue
			}
		case "all":
			// include both
		default:
			// unknown value: default to reference-only
			if comp {
				continue
			}
		}

		var wrongAnswers []string
		if q.WrongAnswers != "" {
			if err := json.Unmarshal([]byte(q.WrongAnswers), &wrongAnswers); err != nil {
				log.Printf("Error unmarshaling wrong answers for question %d: %v", q.ID, err)
				wrongAnswers = []string{}
			}
		}
		options := buildFourOptions(db, q.CorrectAnswer, wrongAnswers)

		themeName := ""
		if q.Theme.ID != 0 {
			themeName = q.Theme.Name
		}

		verses = append(verses, VerseResponse{
			ID:            q.ID,
			ThemeID:       q.ThemeID,
			ThemeName:     themeName,
			Text:          sanitizeDisplayText(q.Text),
			CorrectAnswer: q.CorrectAnswer,
			Options:       options,
			Reference:     q.Reference,
			Difficulty:    q.Difficulty,
		})
	}

	c.Set("Content-Type", "application/json; charset=utf-8")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	return c.JSON(fiber.Map{"success": true, "verses": verses, "count": len(verses)})
}

func GetVerse(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		log.Println("Database not initialized in GetVerse")
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Database not available"})
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Verse ID is required"})
	}

	var question models.Question
	err := db.Preload("Theme").First(&question, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("Verse with ID %s not found", id)})
		}
		log.Printf("Error fetching verse %s: %v", id, err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Failed to fetch verse"})
	}

	var wrongAnswers []string
	if question.WrongAnswers != "" {
		if err := json.Unmarshal([]byte(question.WrongAnswers), &wrongAnswers); err != nil {
			log.Printf("Error unmarshaling wrong answers: %v", err)
			wrongAnswers = []string{}
		}
	}
	options := buildFourOptions(db, question.CorrectAnswer, wrongAnswers)

	themeName := ""
	if question.Theme.ID != 0 {
		themeName = question.Theme.Name
	}

	verse := VerseResponse{
		ID:            question.ID,
		ThemeID:       question.ThemeID,
		ThemeName:     themeName,
		Text:          sanitizeDisplayText(question.Text),
		CorrectAnswer: question.CorrectAnswer,
		Options:       options,
		Reference:     question.Reference,
		Difficulty:    question.Difficulty,
	}

	c.Set("Content-Type", "application/json; charset=utf-8")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	return c.JSON(fiber.Map{"success": true, "verse": verse, "count": 1})
}

func GetQuizQuestions(c *fiber.Ctx) error {
	db := database.GetDB()
	if db == nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Database not available"})
	}

	themeID := c.Query("theme_id")
	count := c.QueryInt("count", 20)
	difficulty := c.Query("difficulty", "")
	// Same question_type filter for quiz
	questionType := strings.ToLower(strings.TrimSpace(c.Query("question_type", "reference")))

	if count < 1 || count > 100 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Count must be between 1 and 100"})
	}

	query := db.Model(&models.Question{}).Preload("Theme")
	if themeID != "" {
		query = query.Where("theme_id = ?", themeID)
	}
	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}

	// Don't limit initially - we need to know total available
	var questions []models.Question
	if err := query.Find(&questions).Error; err != nil {
		log.Printf("Error fetching quiz questions: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Failed to fetch questions"})
	}

	verses := make([]VerseResponse, 0, len(questions))
	for _, q := range questions {
		comp := isCompletionQuestion(q)

		switch questionType {
		case "reference":
			if comp {
				continue
			}
		case "completion":
			if !comp {
				continue
			}
		case "all":
			// include both
		default:
			if comp {
				continue
			}
		}

		var wrongAnswers []string
		if q.WrongAnswers != "" {
			if err := json.Unmarshal([]byte(q.WrongAnswers), &wrongAnswers); err != nil {
				log.Printf("Error unmarshaling wrong answers: %v", err)
				wrongAnswers = []string{}
			}
		}
		options := buildFourOptions(db, q.CorrectAnswer, wrongAnswers)

		themeName := ""
		if q.Theme.ID != 0 {
			themeName = q.Theme.Name
		}

		verses = append(verses, VerseResponse{
			ID:            q.ID,
			ThemeID:       q.ThemeID,
			ThemeName:     themeName,
			Text:          sanitizeDisplayText(q.Text),
			CorrectAnswer: q.CorrectAnswer,
			Options:       options,
			Reference:     q.Reference,
			Difficulty:    q.Difficulty,
		})
	}

	if len(verses) == 0 {
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "No questions available for the selected criteria"})
	}

	// Smart repetition: if user requests more questions than available, repeat to fill gap
	finalVerses := verses
	if count > len(verses) {
		available := len(verses)
		needed := count - available
		log.Printf("📚 Quiz: Requested %d questions, only %d available. Repeating %d questions.", count, available, needed)

		// Create repeat pool and shuffle it
		repeatPool := make([]VerseResponse, len(verses))
		copy(repeatPool, verses)
		rand.Shuffle(len(repeatPool), func(i, j int) {
			repeatPool[i], repeatPool[j] = repeatPool[j], repeatPool[i]
		})

		// Add repeated questions to fill the gap
		for i := 0; i < needed && i < len(repeatPool); i++ {
			finalVerses = append(finalVerses, repeatPool[i])
		}

		// If still need more (very small theme), loop through again
		for len(finalVerses) < count && len(repeatPool) > 0 {
			idx := len(finalVerses) % len(repeatPool)
			finalVerses = append(finalVerses, repeatPool[idx])
		}

		// Shuffle to mix original and repeated questions
		rand.Shuffle(len(finalVerses), func(i, j int) {
			finalVerses[i], finalVerses[j] = finalVerses[j], finalVerses[i]
		})

		log.Printf("✅ Quiz: Returning %d questions (%d original + %d repeated)", len(finalVerses), available, len(finalVerses)-available)
	} else {
		// Shuffle and limit to requested count
		rand.Shuffle(len(finalVerses), func(i, j int) {
			finalVerses[i], finalVerses[j] = finalVerses[j], finalVerses[i]
		})
		finalVerses = finalVerses[:count]
	}

	c.Set("Content-Type", "application/json; charset=utf-8")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	return c.JSON(fiber.Map{"success": true, "verses": finalVerses, "count": len(finalVerses)})
}
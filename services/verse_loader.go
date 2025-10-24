package services

import (
	"ubible/database"
	"ubible/models"
	"ubible/verseparser"
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const VersesDirectory = "./verses"

type VerseFile struct {
	Theme     string     `json:"theme"`
	Questions []Question `json:"questions"`
}

type Question struct {
	Text          string   `json:"text"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
	Difficulty    string   `json:"difficulty"`
	Reference     string   `json:"reference"`
	ThemeName     string   `json:"theme_name,omitempty"`
}

type Verse struct {
	Reference string `json:"reference"`
	Text      string `json:"text"`
}

var (
	questionsByTheme = make(map[string][]Question)
	questionsLock    sync.RWMutex
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// InitVerseData loads verses and then cleans up deleted themes
func InitVerseData() {
	if err := LoadVersesFromFiles(); err != nil {
		log.Printf("Error loading verses: %v", err)
	}
	if err := LoadVersesFromTXT(); err != nil {
		log.Printf("Error loading TXT verses: %v", err)
	}
	if err := CleanupDeletedThemes(); err != nil {
		log.Printf("Theme cleanup failed: %v", err)
	}
}

func LoadVersesFromFiles() error {
	if _, err := os.Stat(VersesDirectory); os.IsNotExist(err) {
		log.Println("Verses directory not found, creating it...")
		if err := os.MkdirAll(VersesDirectory, 0755); err != nil {
			return fmt.Errorf("failed to create verses directory: %w", err)
		}
		if err := createSampleVerseFile(); err != nil {
			return fmt.Errorf("failed to create sample file: %w", err)
		}
		return nil
	}

	files, err := filepath.Glob(filepath.Join(VersesDirectory, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to read verses directory: %w", err)
	}

	if len(files) == 0 {
		log.Println("No JSON verse files found in verses directory")
		if err := createSampleVerseFile(); err != nil {
			return fmt.Errorf("failed to create sample file: %w", err)
		}
		return nil
	}

	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	for _, file := range files {
		log.Printf("Loading verses from JSON: %s", file)

		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Failed to read file %s: %v", file, err)
			continue
		}

		var verseFile VerseFile
		if err := json.Unmarshal(data, &verseFile); err != nil {
			log.Printf("Failed to parse JSON in %s: %v", file, err)
			continue
		}

		if verseFile.Theme == "" {
			log.Printf("Skipping file %s: missing theme name", file)
			continue
		}

		var theme models.Theme
		if err := db.Where("name = ?", verseFile.Theme).First(&theme).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				theme = models.Theme{
					Name:        verseFile.Theme,
					Description: fmt.Sprintf("Questions about %s", verseFile.Theme),
				}
				if err := db.Create(&theme).Error; err != nil {
					log.Printf("Failed to create theme %s: %v", verseFile.Theme, err)
					continue
				}
				log.Printf("Created theme: %s", theme.Name)
			} else {
				log.Printf("Database error checking theme: %v", err)
				continue
			}
		}

		for _, q := range verseFile.Questions {
			if q.Text == "" || q.CorrectAnswer == "" || len(q.Options) < 2 {
				log.Printf("Skipping invalid question in %s", file)
				continue
			}

			var existing models.Question
			if err := db.Where("text = ? AND theme_id = ?", q.Text, theme.ID).First(&existing).Error; err == nil {
				continue
			}

			wa := make([]string, 0, len(q.Options))
			for _, opt := range q.Options {
				opt = strings.TrimSpace(opt)
				if opt != "" && opt != q.CorrectAnswer {
					wa = append(wa, opt)
				}
			}
			wa = dedup(wa)

			wrongAnswersJSON, err := json.Marshal(wa)
			if err != nil {
				log.Printf("Failed to marshal wrong answers: %v", err)
				continue
			}

			question := models.Question{
				ThemeID:       theme.ID,
				Text:          strings.TrimSpace(q.Text),
				WrongAnswers:  string(wrongAnswersJSON),
				CorrectAnswer: strings.TrimSpace(q.CorrectAnswer),
				Difficulty:    strings.TrimSpace(q.Difficulty),
				Reference:     strings.TrimSpace(q.Reference),
			}

			if err := db.Create(&question).Error; err != nil {
				log.Printf("Failed to create question: %v", err)
				continue
			}
		}

		StoreQuestionsInMemory(theme.Name, verseFile.Questions)
		log.Printf("Successfully loaded verses from %s", filepath.Base(file))
	}

	return nil
}

func LoadVersesFromTXT() error {
	files, err := filepath.Glob(filepath.Join(VersesDirectory, "*.txt"))
	if err != nil {
		return fmt.Errorf("failed to read verses directory: %w", err)
	}

	if len(files) == 0 {
		log.Println("No TXT verse files found in verses directory")
		return nil
	}

	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	allowDirect := os.Getenv("VERSE_FORMAT_ALLOW_DIRECT") == "true"
	allowQA := os.Getenv("VERSE_FORMAT_ALLOW_QA") == "true"
	generateCompletion := os.Getenv("GENERATE_COMPLETION_QUESTIONS") == "true"

	for _, file := range files {
		log.Printf("Loading verses from TXT: %s", file)

		themeName := strings.TrimSuffix(filepath.Base(file), ".txt")
		themeName = strings.ReplaceAll(themeName, "_", " ")
		themeName = strings.Title(themeName)

		var theme models.Theme
		if err := db.Where("name = ?", themeName).First(&theme).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				theme = models.Theme{
					Name:        themeName,
					Description: fmt.Sprintf("Questions about %s", themeName),
				}
				if err := db.Create(&theme).Error; err != nil {
					log.Printf("Failed to create theme %s: %v", themeName, err)
					continue
				}
				log.Printf("Created theme: %s", theme.Name)
			} else {
				log.Printf("Database error: %v", err)
				continue
			}
		}

		verses, badLines, err := parseVerseFile(file, allowDirect)
		if err != nil {
			log.Printf("Failed to parse verse file %s: %v", file, err)
			continue
		}
		for _, bl := range badLines {
			log.Printf("WARN %s:%d: does not match 'N. <Reference> — <Text>'", file, bl)
		}

		if len(verses) > 0 {
			log.Printf("Detected verse format, generating %d-%d questions from %d verses...", len(verses)*1, len(verses)*2, len(verses))
			if err := generateQuestionsFromVerses(db, theme, verses, generateCompletion); err != nil {
				log.Printf("Failed to generate questions: %v", err)
				continue
			}
		} else if allowQA {
			log.Printf("No verses detected, trying Q&A format...")
			if err := parseQAFormat(db, theme, file); err != nil {
				log.Printf("Failed to parse Q&A format: %v", err)
				continue
			}
		} else {
			log.Printf("No verses found and Q&A disabled. Skipping: %s", file)
		}

		log.Printf("Successfully loaded TXT verses from %s", filepath.Base(file))
	}

	return nil
}

func parseVerseFile(filePath string, _ bool) ([]Verse, []int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var verses []Verse
	var badLines []int
	scanner := bufio.NewScanner(file)

	// Example accepted formats:
	// 1. N. John 3:16 — For God so loved the world ...
	// 2. John 3:16 — For God so loved the world ...    (when VERSE_FORMAT_ALLOW_DIRECT=true)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		line := strings.TrimSpace(raw)

		// Normalize special Unicode spaces to regular spaces
		line = strings.ReplaceAll(line, "\u202F", " ") // Narrow no-break space
		line = strings.ReplaceAll(line, "\u00A0", " ") // Non-breaking space
		line = strings.ReplaceAll(line, "–", "-")      // En dash
		line = strings.ReplaceAll(line, "—", "-")      // Em dash

		if line == "" {
			continue
		}

		if ref, text := verseparser.ParseVerseSmart(line); ref != "" && text != "" {
			verses = append(verses, Verse{Reference: ref, Text: text})
			continue
		}

		badLines = append(badLines, lineNum)
	}

	if err := scanner.Err(); err != nil {
		return nil, badLines, fmt.Errorf("scanner error: %w", err)
	}

	return verses, badLines, nil
}

func parseQAFormat(db *gorm.DB, theme models.Theme, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentQuestion *models.Question
	var currentOptions []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Q:") {
			if currentQuestion != nil && len(currentOptions) > 0 {
				if err := saveQuestion(db, currentQuestion, currentOptions); err != nil {
					log.Printf("Failed to save question: %v", err)
				}
			}
			currentQuestion = &models.Question{
				ThemeID:    theme.ID,
				Text:       strings.TrimSpace(line[2:]),
				Difficulty: "medium",
			}
			currentOptions = []string{}
		} else if currentQuestion != nil && (strings.HasPrefix(line, "A:") ||
			strings.HasPrefix(line, "B:") ||
			strings.HasPrefix(line, "C:") ||
			strings.HasPrefix(line, "D:")) {
			answer := strings.TrimSpace(line[2:])
			currentOptions = append(currentOptions, answer)
			if strings.HasPrefix(line, "A:") {
				currentQuestion.CorrectAnswer = answer
			}
		}
	}

	if currentQuestion != nil && len(currentOptions) > 0 {
		if err := saveQuestion(db, currentQuestion, currentOptions); err != nil {
			return fmt.Errorf("failed to save last question: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

func generateQuestionsFromVerses(db *gorm.DB, theme models.Theme, verses []Verse, _ bool) error {
	if len(verses) < 4 {
		return fmt.Errorf("not enough verses to generate questions (need at least 4, got %d)", len(verses))
	}

	questionsCreated := 0

	for i, verse := range verses {
		// Generate 2 questions per verse:
		// Q1: Verse text → pick reference
		// Q2: Reference → pick verse text
		questions := []models.Question{
			generateVerseToReferenceQuestion(verse, verses, i, db),
			generateReferenceToVerseQuestion(verse, verses, i, db),
		}

		for _, question := range questions {
			if strings.TrimSpace(question.Text) == "" {
				continue
			}
			question.ThemeID = theme.ID

			var existing models.Question
			if err := db.Where("text = ? AND theme_id = ?", question.Text, question.ThemeID).First(&existing).Error; err == nil {
				continue
			}

			if err := db.Create(&question).Error; err != nil {
				log.Printf("Failed to create question: %v", err)
				continue
			}
			questionsCreated++
		}
	}

	log.Printf("Created %d questions from %d verses", questionsCreated, len(verses))
	return nil
}

// generateVerseToReferenceQuestion: Shows verse text → user picks reference
func generateVerseToReferenceQuestion(correct Verse, allVerses []Verse, _ int, db *gorm.DB) models.Question {
	wrongRefs := generateWrongReferencesFromDB(db, correct.Reference, allVerses)
	wrongAnswersJSON, _ := json.Marshal(dedup(wrongRefs))

	cleanText := strings.TrimPrefix(correct.Text, "- ")
	cleanText = strings.TrimSpace(cleanText)

	return models.Question{
		Text:          cleanText,
		CorrectAnswer: strings.TrimSpace(correct.Reference),
		WrongAnswers:  string(wrongAnswersJSON),
		Reference:     strings.TrimSpace(correct.Reference),
		Difficulty:    "medium",
	}
}

// generateReferenceToVerseQuestion: Shows reference → user picks verse text
func generateReferenceToVerseQuestion(correct Verse, allVerses []Verse, _ int, db *gorm.DB) models.Question {
	cleanText := strings.TrimPrefix(correct.Text, "- ")
	cleanText = strings.TrimSpace(cleanText)

	wrongTexts := generateWrongVersesFromDB(db, cleanText, correct.Reference, allVerses)
	wrongAnswersJSON, _ := json.Marshal(dedup(wrongTexts))

	return models.Question{
		Text:          strings.TrimSpace(correct.Reference),
		CorrectAnswer: cleanText,
		WrongAnswers:  string(wrongAnswersJSON),
		Reference:     strings.TrimSpace(correct.Reference),
		Difficulty:    "medium",
	}
}

// Unused function - kept for future use
// func splitAtWordBoundary(s string) (first, second string) {
// 	runes := []rune(strings.TrimSpace(s))
// 	if len(runes) == 0 {
// 		return "", ""
// 	}
// 	mid := len(runes) / 2

// 	i := mid
// 	for i > 0 && !unicode.IsSpace(runes[i]) {
// 		i--
// 	}
// 	if i == 0 {
// 		i = mid
// 		for i < len(runes) && !unicode.IsSpace(runes[i]) {
// 			i++
// 		}
// 		if i >= len(runes) {
// 			i = mid
// 		}
// 	}

// 	first = strings.TrimSpace(string(runes[:i]))
// 	second = strings.TrimSpace(string(runes[i:]))
// 	return
// }

// Unused function - kept for future use
// func generateVerseCompletionQuestion(correct Verse, allVerses []Verse, excludeIndex int) models.Question {
// 	trimmed := strings.TrimSpace(correct.Text)
// 	if len([]rune(trimmed)) < 40 {
// 		return models.Question{}
// 	}

// 	firstPart, secondPart := splitAtWordBoundary(trimmed)
// 	if firstPart == "" || secondPart == "" {
// 		return models.Question{}
// 	}

// 	randomVerses := getRandomVerses(allVerses, excludeIndex, 3)
// 	options := []string{secondPart}

// 	for _, v := range randomVerses {
// 		vtrim := strings.TrimSpace(v.Text)
// 		if len([]rune(vtrim)) < 40 {
// 			continue
// 		}
// 		_, wrongEnd := splitAtWordBoundary(vtrim)
// 		if wrongEnd != "" {
// 			options = append(options, wrongEnd)
// 		}
// 	}
// 	if len(options) < 4 {
// 		return models.Question{}
// 	}

// 	shuffleStrings(options)
// 	wrongAnswers := make([]string, 0, 3)
// 	for _, opt := range options {
// 		if opt != secondPart {
// 			wrongAnswers = append(wrongAnswers, opt)
// 		}
// 	}
// 	wrongAnswersJSON, _ := json.Marshal(dedup(wrongAnswers))

// 	return models.Question{
// 		Text:          fmt.Sprintf("Complete this verse: %s…", firstPart),
// 		CorrectAnswer: secondPart,
// 		WrongAnswers:  string(wrongAnswersJSON),
// 		Reference:     strings.TrimSpace(correct.Reference),
// 		Difficulty:    "hard",
// 	}
// }

func saveQuestion(db *gorm.DB, question *models.Question, options []string) error {
	if question == nil || len(options) == 0 {
		return fmt.Errorf("invalid question or options")
	}

	wrongAnswers := []string{}
	for _, opt := range options {
		opt = strings.TrimSpace(opt)
		if opt != "" && opt != strings.TrimSpace(question.CorrectAnswer) {
			wrongAnswers = append(wrongAnswers, opt)
		}
	}
	wrongAnswers = dedup(wrongAnswers)

	wrongAnswersJSON, err := json.Marshal(wrongAnswers)
	if err != nil {
		return fmt.Errorf("failed to marshal wrong answers: %w", err)
	}
	question.WrongAnswers = string(wrongAnswersJSON)

	var existing models.Question
	if err := db.Where("text = ? AND theme_id = ?", strings.TrimSpace(question.Text), question.ThemeID).First(&existing).Error; err == nil {
		return nil
	}

	if err := db.Create(question).Error; err != nil {
		return fmt.Errorf("failed to create question: %w", err)
	}

	return nil
}

func getRandomVerses(verses []Verse, excludeIndex int, count int) []Verse {
	if len(verses) <= count {
		result := make([]Verse, 0, len(verses))
		for i, v := range verses {
			if i != excludeIndex {
				result = append(result, v)
			}
		}
		return result
	}

	available := make([]Verse, 0, len(verses)-1)
	for i, v := range verses {
		if i != excludeIndex {
			available = append(available, v)
		}
	}

	if len(available) <= count {
		return available
	}

	rand.Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	return available[:count]
}

// Unused function - kept for future use
// func shuffleStrings(slice []string) {
// 	rand.Shuffle(len(slice), func(i, j int) {
// 		slice[i], slice[j] = slice[j], slice[i]
// 	})
// }

func dedup(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func createSampleVerseFile() error {
	sample := VerseFile{
		Theme: "Creation",
		Questions: []Question{
			{
				Text: "In the beginning, God created the heavens and the earth. What was the earth initially described as?",
				Options: []string{
					"Formless and empty",
					"Beautiful and ordered",
					"Filled with life",
					"Covered in light",
				},
				CorrectAnswer: "Formless and empty",
				Difficulty:    "easy",
				Reference:     "Genesis 1:1-2",
			},
		},
	}

	data, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sample: %w", err)
	}

	filename := filepath.Join(VersesDirectory, "sample_creation.json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write sample file: %w", err)
	}

	log.Printf("Created sample verse file: %s", filename)
	return nil
}

func GetRandomQuestions(count int, themes []string, difficulty string) ([]Question, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var questions []models.Question
	query := db.Model(&models.Question{}).Preload("Theme")

	if len(themes) > 0 && themes[0] != "All" {
		var themeIDs []uint
		var dbThemes []models.Theme
		if err := db.Where("name IN ?", themes).Find(&dbThemes).Error; err == nil {
			for _, t := range dbThemes {
				themeIDs = append(themeIDs, t.ID)
			}
			if len(themeIDs) > 0 {
				query = query.Where("theme_id IN ?", themeIDs)
			}
		}
	}

	if difficulty != "" && difficulty != "all" {
		query = query.Where("difficulty = ?", difficulty)
	}

	if err := query.Order("RANDOM()").Limit(count).Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch questions: %w", err)
	}

	result := make([]Question, len(questions))
	for i, q := range questions {
		var wrongAnswers []string
		if q.WrongAnswers != "" {
			_ = json.Unmarshal([]byte(q.WrongAnswers), &wrongAnswers)
		}

		options := append([]string{strings.TrimSpace(q.CorrectAnswer)}, wrongAnswers...)
		rand.Shuffle(len(options), func(i, j int) {
			options[i], options[j] = options[j], options[i]
		})

		result[i] = Question{
			Text:          strings.TrimSpace(q.Text),
			Options:       options,
			CorrectAnswer: strings.TrimSpace(q.CorrectAnswer),
			Difficulty:    strings.TrimSpace(q.Difficulty),
			Reference:     strings.TrimSpace(q.Reference),
			ThemeName:     strings.TrimSpace(q.Theme.Name),
		}
	}

	return result, nil
}

func StoreQuestionsInMemory(theme string, questions []Question) {
	questionsLock.Lock()
	defer questionsLock.Unlock()
	questionsByTheme[theme] = questions
}

func CleanupDeletedThemes() error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	var existingThemes []models.Theme
	if err := db.Find(&existingThemes).Error; err != nil {
		return fmt.Errorf("failed to fetch existing themes: %w", err)
	}

	validThemes := make(map[string]bool)

	jsonFiles, _ := filepath.Glob(filepath.Join(VersesDirectory, "*.json"))
	txtFiles, _ := filepath.Glob(filepath.Join(VersesDirectory, "*.txt"))
	allFiles := append(jsonFiles, txtFiles...)

	for _, file := range allFiles {
		themeName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		themeName = strings.ReplaceAll(themeName, "_", " ")
		themeName = strings.Title(themeName)
		validThemes[themeName] = true
	}

	for _, theme := range existingThemes {
		if !validThemes[theme.Name] {
			log.Printf("Cleaning up deleted theme: %s", theme.Name)
			if err := db.Where("theme_id = ?", theme.ID).Delete(&models.Question{}).Error; err != nil {
				log.Printf("Failed to delete questions for theme %s: %v", theme.Name, err)
			}
			if err := db.Delete(&theme).Error; err != nil {
				log.Printf("Failed to delete theme %s: %v", theme.Name, err)
			}
		}
	}

	return nil
}

func InitVerseService() {}

// generateWrongReferencesFromDB generates wrong references from same theme verses
func generateWrongReferencesFromDB(_ *gorm.DB, correctRef string, themeVerses []Verse) []string {
	wrong := []string{}
	used := map[string]bool{correctRef: true}

	// Get all other verses from the same theme
	randomVerses := getRandomVerses(themeVerses, -1, len(themeVerses))
	for _, v := range randomVerses {
		if len(wrong) >= 3 {
			break
		}
		if !used[v.Reference] && v.Reference != correctRef {
			wrong = append(wrong, v.Reference)
			used[v.Reference] = true
		}
	}

	// Fallback: generate random references if not enough verses in theme
	if len(wrong) < 3 {
		books := []string{"John", "Matthew", "Mark", "Luke", "Romans", "Genesis", "Psalms", "Proverbs", "Isaiah", "Jeremiah"}
		for len(wrong) < 3 {
			book := books[rand.Intn(len(books))]
			chapter := rand.Intn(20) + 1
			verse := rand.Intn(30) + 1
			ref := fmt.Sprintf("%s %d:%d", book, chapter, verse)
			if !used[ref] {
				wrong = append(wrong, ref)
				used[ref] = true
			}
		}
	}

	return wrong
}

// generateWrongVersesFromDB generates wrong verse texts from same theme verses
func generateWrongVersesFromDB(_ *gorm.DB, correctText string, _ string, themeVerses []Verse) []string {
	wrong := []string{}
	used := map[string]bool{correctText: true}

	// Get all other verses from the same theme
	randomVerses := getRandomVerses(themeVerses, -1, len(themeVerses))
	for _, v := range randomVerses {
		if len(wrong) >= 3 {
			break
		}
		cleanText := strings.TrimPrefix(v.Text, "- ")
		cleanText = strings.TrimSpace(cleanText)
		if !used[cleanText] && cleanText != correctText && cleanText != "" {
			wrong = append(wrong, cleanText)
			used[cleanText] = true
		}
	}

	// Fallback: generate generic text if not enough verses in theme
	for len(wrong) < 3 {
		wrong = append(wrong, fmt.Sprintf("Alternative verse text %d", len(wrong)+1))
	}

	return wrong
}

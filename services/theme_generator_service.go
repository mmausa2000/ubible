package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// BibleBook represents a book in the KJV Bible
type BibleBook struct {
	Abbrev   string     `json:"abbrev"`
	Name     string     `json:"name"`
	Chapters [][]string `json:"chapters"`
}

// SearchResult represents a verse found in search
type SearchResult struct {
	Reference string `json:"reference"`
	Text      string `json:"text"`
	Book      string `json:"book"`
	Chapter   int    `json:"chapter"`
	Verse     int    `json:"verse"`
}

// GeneratedTheme represents a complete generated theme
type GeneratedTheme struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Introduction string `json:"introduction"`
	KeyInsights  string `json:"key_insights"`
	Application  string `json:"application"`
	Prayer       string `json:"prayer"`
	Conclusion   string `json:"conclusion"`
}

// ThemeGeneratorRequest represents the request for theme generation
type ThemeGeneratorRequest struct {
	Keywords  []string `json:"keywords"`
	Count     int      `json:"count"`
	Testament string   `json:"testament"` // "OT", "NT", or "BOTH"
	Books     []string `json:"books"`     // Specific books to search in
}

var (
	bibleData      []BibleBook
	bibleDataError error
	otBooks        = map[string]bool{
		"genesis": true, "exodus": true, "leviticus": true, "numbers": true, "deuteronomy": true,
		"joshua": true, "judges": true, "ruth": true, "1 samuel": true, "2 samuel": true,
		"1 kings": true, "2 kings": true, "1 chronicles": true, "2 chronicles": true,
		"ezra": true, "nehemiah": true, "esther": true, "job": true, "psalms": true,
		"proverbs": true, "ecclesiastes": true, "song of solomon": true, "isaiah": true,
		"jeremiah": true, "lamentations": true, "ezekiel": true, "daniel": true, "hosea": true,
		"joel": true, "amos": true, "obadiah": true, "jonah": true, "micah": true,
		"nahum": true, "habakkuk": true, "zephaniah": true, "haggai": true, "zechariah": true, "malachi": true,
	}
)

// LoadBibleData loads the KJV Bible data from kjv-full.json
func LoadBibleData() error {
	if len(bibleData) > 0 {
		return nil // Already loaded
	}

	if bibleDataError != nil {
		return bibleDataError
	}

	file, err := os.ReadFile("./verses/kjv-full.json")
	if err != nil {
		bibleDataError = fmt.Errorf("failed to read kjv-full.json: %w", err)
		return bibleDataError
	}

	if err := json.Unmarshal(file, &bibleData); err != nil {
		bibleDataError = fmt.Errorf("failed to parse kjv-full.json: %w", err)
		return bibleDataError
	}

	log.Printf("Loaded %d books from KJV Bible", len(bibleData))
	return nil
}

// SearchVerses searches for verses matching keywords
func SearchVerses(req ThemeGeneratorRequest) ([]SearchResult, error) {
	if err := LoadBibleData(); err != nil {
		return nil, err
	}

	var results []SearchResult
	count := req.Count
	if count <= 0 {
		count = 20
	}
	if count > 500 {
		count = 500
	}

	// Normalize books filter
	bookFilter := make(map[string]bool)
	for _, book := range req.Books {
		bookFilter[strings.ToLower(strings.TrimSpace(book))] = true
	}

	// Search through all books
	for _, book := range bibleData {
		// Skip if we have enough results
		if len(results) >= count {
			break
		}

		bookNameLower := strings.ToLower(book.Name)

		// Filter by testament
		if req.Testament == "OT" && !otBooks[bookNameLower] {
			continue
		}
		if req.Testament == "NT" && otBooks[bookNameLower] {
			continue
		}

		// Filter by specific books
		if len(bookFilter) > 0 && !bookFilter[bookNameLower] {
			continue
		}

		// Search through chapters and verses
		for chapterNum, verses := range book.Chapters {
			for verseNum, verseText := range verses {
				if len(results) >= count {
					break
				}

				// Check if verse matches any keyword (case-insensitive)
				matches := false
				verseTextLower := strings.ToLower(verseText)
				for _, keyword := range req.Keywords {
					keywordLower := strings.ToLower(strings.TrimSpace(keyword))
					if keywordLower != "" && strings.Contains(verseTextLower, keywordLower) {
						matches = true
						break
					}
				}

				if matches {
					reference := fmt.Sprintf("%s %d:%d", book.Name, chapterNum+1, verseNum+1)
					results = append(results, SearchResult{
						Reference: reference,
						Text:      verseText,
						Book:      book.Name,
						Chapter:   chapterNum + 1,
						Verse:     verseNum + 1,
					})
				}
			}
		}
	}

	log.Printf("Found %d verses matching keywords: %v", len(results), req.Keywords)
	return results, nil
}

// GenerateTheme generates a theme based on search results
func GenerateTheme(keywords []string, verses []SearchResult) GeneratedTheme {
	keywordsStr := strings.Join(keywords, ", ")
	keywordsTitle := strings.Title(strings.Join(keywords, " and "))

	// Generate introduction
	introduction := generateIntroduction(keywordsStr, len(verses))

	// Generate key insights
	keyInsights := generateKeyInsights(keywords, verses)

	// Generate application
	application := generateApplication(keywords)

	// Generate prayer
	prayer := generatePrayer(keywords)

	// Generate conclusion
	conclusion := generateConclusion(keywordsStr, len(verses))

	return GeneratedTheme{
		Title:        fmt.Sprintf("%s in Scripture", keywordsTitle),
		Description:  fmt.Sprintf("A comprehensive study exploring %s through %d Bible verses", keywordsStr, len(verses)),
		Introduction: introduction,
		KeyInsights:  keyInsights,
		Application:  application,
		Prayer:       prayer,
		Conclusion:   conclusion,
	}
}

// generateIntroduction creates an introduction for the theme
func generateIntroduction(keywords string, verseCount int) string {
	return fmt.Sprintf(`This theme explores the biblical teaching on %s. Through %d carefully selected verses, we will discover what God's Word reveals about this important topic.

The Scriptures provide rich insight into %s, showing us God's wisdom and guidance for our lives. As we study these verses together, may we grow in understanding and faith.`, keywords, verseCount, keywords)
}

// generateKeyInsights analyzes verses and creates key insights
func generateKeyInsights(keywords []string, verses []SearchResult) string {
	if len(verses) == 0 {
		return "No verses found to generate insights."
	}

	insights := []string{
		"Key Insights:",
	}

	// Group verses by book
	bookCounts := make(map[string]int)
	for _, v := range verses {
		bookCounts[v.Book]++
	}

	// Find most mentioned books
	var topBooks []string
	for book, count := range bookCounts {
		if count >= 3 {
			topBooks = append(topBooks, fmt.Sprintf("%s (%d verses)", book, count))
		}
	}

	if len(topBooks) > 0 {
		insights = append(insights, fmt.Sprintf("\n1. This topic is prominently featured in: %s", strings.Join(topBooks, ", ")))
	} else {
		insights = append(insights, fmt.Sprintf("\n1. These %d verses span across multiple books of the Bible, showing the breadth of this teaching.", len(verses)))
	}

	// Analyze testament distribution
	otCount := 0
	ntCount := 0
	for _, v := range verses {
		if otBooks[strings.ToLower(v.Book)] {
			otCount++
		} else {
			ntCount++
		}
	}

	if otCount > 0 && ntCount > 0 {
		insights = append(insights, fmt.Sprintf("\n2. The theme appears in both the Old Testament (%d verses) and New Testament (%d verses), demonstrating its continuity throughout Scripture.", otCount, ntCount))
	} else if otCount > 0 {
		insights = append(insights, fmt.Sprintf("\n2. All %d verses come from the Old Testament, showing the foundational teaching on this topic.", otCount))
	} else {
		insights = append(insights, fmt.Sprintf("\n2. All %d verses come from the New Testament, revealing the fulfillment and application of this teaching.", ntCount))
	}

	// Add keyword-specific insights
	keywordStr := strings.Join(keywords, " and ")
	insights = append(insights, fmt.Sprintf("\n3. The recurring themes of %s throughout these passages reveal God's consistent character and unchanging truth.", keywordStr))

	// Sample verse reference
	if len(verses) > 0 {
		insights = append(insights, fmt.Sprintf("\n4. A key verse to remember is %s: \"%s\"", verses[0].Reference, truncateText(verses[0].Text, 150)))
	}

	return strings.Join(insights, "")
}

// generateApplication creates practical application points
func generateApplication(keywords []string) string {
	keywordStr := strings.Join(keywords, " and ")

	applications := []string{
		"Practical Application:",
		"\n1. Meditate daily on what the Bible teaches about " + keywordStr + ".",
		"\n2. Ask God to help you apply these truths in your daily life.",
		"\n3. Share these verses with others who may be seeking wisdom on this topic.",
		"\n4. Memorize key verses to strengthen your faith and understanding.",
		"\n5. Reflect on how these teachings can transform your relationship with God and others.",
	}

	return strings.Join(applications, "")
}

// generatePrayer creates a prayer based on the theme
func generatePrayer(keywords []string) string {
	keywordStr := strings.Join(keywords, " and ")

	return fmt.Sprintf(`A Prayer:

Heavenly Father, thank You for Your Word that teaches us about %s. Help us to understand these truths more deeply and to live according to Your will. Grant us wisdom to apply what we have learned, and strengthen our faith as we seek to honor You in all things. In Jesus' name, Amen.`, keywordStr)
}

// generateConclusion creates a conclusion for the theme
func generateConclusion(keywords string, verseCount int) string {
	return fmt.Sprintf(`As we conclude this study on %s, let us remember that God's Word is living and active. These %d verses are not just ancient texts, but powerful truths that speak to us today.

May you continue to study Scripture, grow in faith, and experience the transforming power of God's Word in your life. Return to these verses often, and let them guide you in your walk with the Lord.`, keywords, verseCount)
}

// truncateText truncates text to maxLength and adds ellipsis if needed
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type U BibleVerse struct {
	ID          uint   `gorm:"primaryKey"`
	Book        string `gorm:"index;not null"`
	Chapter     int    `gorm:"index;not null"`
	Verse       int    `gorm:"index;not null"`
	Text        string `gorm:"type:text;not null"`
	Translation string `gorm:"index;default:'KJV'"`
}

type JSONBook struct {
	Abbrev   string     `json:"abbrev"`
	Chapters [][]string `json:"chapters"`
}

var bookNames = map[string]string{
	"gn": "Genesis", "ex": "Exodus", "lv": "Leviticus", "nm": "Numbers", "dt": "Deuteronomy",
	"js": "Joshua", "jud": "Judges", "rt": "Ruth", "1sm": "1 Samuel", "2sm": "2 Samuel",
	"1kgs": "1 Kings", "2kgs": "2 Kings", "1ch": "1 Chronicles", "2ch": "2 Chronicles",
	"ezr": "Ezra", "ne": "Nehemiah", "et": "Esther", "job": "Job", "ps": "Psalms", "prv": "Proverbs",
	"ec": "Ecclesiastes", "so": "Song of Solomon", "is": "Isaiah", "jr": "Jeremiah",
	"lm": "Lamentations", "ez": "Ezekiel", "dn": "Daniel", "ho": "Hosea", "jl": "Joel",
	"am": "Amos", "ob": "Obadiah", "jo": "Jonah", "mi": "Micah", "na": "Nahum", "hk": "Habakkuk",
	"zp": "Zephaniah", "hg": "Haggai", "zc": "Zechariah", "ml": "Malachi",
	"mt": "Matthew", "mk": "Mark", "lk": "Luke", "jn": "John", "act": "Acts", "rm": "Romans",
	"1co": "1 Corinthians", "2co": "2 Corinthians", "gl": "Galatians", "eph": "Ephesians",
	"ph": "Philippians", "cl": "Colossians", "1ts": "1 Thessalonians", "2ts": "2 Thessalonians",
	"1tm": "1 Timothy", "2tm": "2 Timothy", "tt": "Titus", "phm": "Philemon", "hb": "Hebrews",
	"jm": "James", "1pe": "1 Peter", "2pe": "2 Peter", "1jo": "1 John", "2jo": "2 John",
	"3jo": "3 John", "jd": "Jude", "re": "Revelation",
}

func main() {
	db, err := gorm.Open(sqlite.Open("./data/ubible_quiz.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := db.AutoMigrate(&U BibleVerse{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	jsonPath := "./verses/kjv-full.json"
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatal("Failed to read JSON file:", err)
	}

	var books []JSONBook
	if err := json.Unmarshal(data, &books); err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}

	fmt.Printf("Found %d books\n\n", len(books))

	var verses []U BibleVerse
	
	for _, book := range books {
		bookName := bookNames[book.Abbrev]
		if bookName == "" {
			bookName = book.Abbrev
		}
		fmt.Printf("Processing: %s\n", bookName)
		
		for chapterNum, chapter := range book.Chapters {
			for verseNum, verseText := range chapter {
				verses = append(verses, U BibleVerse{
					Book:        bookName,
					Chapter:     chapterNum + 1,
					Verse:       verseNum + 1,
					Text:        verseText,
					Translation: "KJV",
				})
			}
		}
	}

	fmt.Printf("\nTotal verses to import: %d\n\n", len(verses))

	batchSize := 500
	for i := 0; i < len(verses); i += batchSize {
		end := i + batchSize
		if end > len(verses) {
			end = len(verses)
		}

		batch := verses[i:end]
		if err := db.Create(&batch).Error; err != nil {
			log.Printf("Error inserting batch %d-%d: %v\n", i, end, err)
		} else {
			fmt.Printf("Inserted verses %d-%d\n", i+1, end)
		}
	}

	fmt.Println("\n✓ Migration completed successfully!")
	
	var count int64
	db.Model(&U BibleVerse{}).Count(&count)
	fmt.Printf("✓ Total verses in database: %d\n", count)
}

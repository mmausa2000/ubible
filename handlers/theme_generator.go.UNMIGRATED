package handlers

import (
	"log"
	"strings"
	"ubible/services"

	"github.com/gofiber/fiber/v2"
)

// GenerateTheme handles POST /api/themes/generate
// Generates a Bible theme without using external APIs
func GenerateTheme(c *fiber.Ctx) error {
	var req services.ThemeGeneratorRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate request
	if len(req.Keywords) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "At least one keyword is required",
		})
	}

	// Clean keywords
	cleanedKeywords := []string{}
	for _, kw := range req.Keywords {
		kw = strings.TrimSpace(kw)
		if kw != "" {
			cleanedKeywords = append(cleanedKeywords, kw)
		}
	}

	if len(cleanedKeywords) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "At least one valid keyword is required",
		})
	}

	req.Keywords = cleanedKeywords

	// Set defaults
	if req.Count <= 0 {
		req.Count = 20
	}
	if req.Count > 500 {
		req.Count = 500
	}
	if req.Testament == "" {
		req.Testament = "BOTH"
	}

	// Validate testament
	if req.Testament != "OT" && req.Testament != "NT" && req.Testament != "BOTH" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Testament must be 'OT', 'NT', or 'BOTH'",
		})
	}

	log.Printf("Generating theme with keywords: %v, count: %d, testament: %s, books: %v",
		req.Keywords, req.Count, req.Testament, req.Books)

	// Search for verses
	verses, err := services.SearchVerses(req)
	if err != nil {
		log.Printf("Error searching verses: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to search verses: " + err.Error(),
		})
	}

	if len(verses) == 0 {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "No verses found matching the specified keywords",
		})
	}

	// Generate theme
	theme := services.GenerateTheme(req.Keywords, verses)

	log.Printf("Successfully generated theme '%s' with %d verses", theme.Title, len(verses))

	return c.JSON(fiber.Map{
		"success": true,
		"verses":  verses,
		"theme":   theme,
		"count":   len(verses),
	})
}

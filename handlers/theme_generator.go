package handlers

import (
	"log"
	"net/http"
	"strings"
	"ubible/services"
	"ubible/utils"
)

// GenerateTheme handles POST /api/themes/generate
// Generates a Bible theme without using external APIs
func GenerateTheme(w http.ResponseWriter, r *http.Request) {
	var req services.ThemeGeneratorRequest

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if len(req.Keywords) == 0 {
		utils.JSONError(w, http.StatusBadRequest, "At least one keyword is required")
		return
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
		utils.JSONError(w, http.StatusBadRequest, "At least one valid keyword is required")
		return
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
		utils.JSONError(w, http.StatusBadRequest, "Testament must be 'OT', 'NT', or 'BOTH'")
		return
	}

	log.Printf("Generating theme with keywords: %v, count: %d, testament: %s, books: %v",
		req.Keywords, req.Count, req.Testament, req.Books)

	// Search for verses
	verses, err := services.SearchVerses(req)
	if err != nil {
		log.Printf("Error searching verses: %v", err)
		utils.JSONError(w, http.StatusInternalServerError, "Failed to search verses: "+err.Error())
		return
	}

	if len(verses) == 0 {
		utils.JSONError(w, http.StatusNotFound, "No verses found matching the specified keywords")
		return
	}

	// Generate theme
	theme := services.GenerateTheme(req.Keywords, verses)

	log.Printf("Successfully generated theme '%s' with %d verses", theme.Title, len(verses))

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"verses":  verses,
		"theme":   theme,
		"count":   len(verses),
	})
}

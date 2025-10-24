// handlers/game.go - Secure game URL handler with access control
package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// HandleGameURL handles requests to secure game URLs
// Validates that the user is authorized to access the game
func HandleGameURL(c *fiber.Ctx) error {
	gameID := c.Params("gameID")

	if gameID == "" {
		return c.Redirect("/")
	}

	// Get the game session
	gameSession, exists := GetGameSession(gameID)
	if !exists {
		// Game doesn't exist, redirect to homepage
		return c.Redirect("/")
	}

	// Check if session has expired
	gameSession.mu.RLock()
	expired := time.Now().After(gameSession.ExpiresAt)
	gameSession.mu.RUnlock()

	if expired {
		// Session expired, redirect to homepage
		return c.Redirect("/")
	}

	// Get player ID from query param or cookie
	// This allows sharing the link, but still requires authorization
	playerID := c.Query("player_id")
	if playerID == "" {
		// Try to get from cookie/session
		playerID = c.Cookies("player_id")
	}

	if playerID == "" {
		// No player ID provided, redirect to homepage with message
		return c.Redirect("/?error=no_player_id")
	}

	// Check if player is authorized for this game
	gameSession.mu.RLock()
	authorized := gameSession.AuthorizedPlayers[playerID]
	gameSession.mu.RUnlock()

	if !authorized {
		// Player not authorized for this game
		// This is the Chess.com-style redirect behavior
		return c.Redirect("/?error=unauthorized")
	}

	// Player is authorized! Set cookie and serve the quiz page
	c.Cookie(&fiber.Cookie{
		Name:     "player_id",
		Value:    playerID,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "game_id",
		Value:    gameID,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: false, // Allow JavaScript access
		SameSite: "Lax",
	})

	// Serve the quiz.html page
	return c.SendFile("./static/quiz.html")
}

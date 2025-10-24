// handlers/debug.go - Debug endpoints for troubleshooting multiplayer
package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// DebugRoomInfo represents room information for debugging
type DebugRoomInfo struct {
	RoomCode      string   `json:"room_code"`
	Host          string   `json:"host"`
	PlayerCount   int      `json:"player_count"`
	MaxPlayers    int      `json:"max_players"`
	State         string   `json:"state"`
	GameID        string   `json:"game_id"`
	GameURL       string   `json:"game_url"`
	PlayerIDs     []string `json:"player_ids"`
	PlayerNames   []string `json:"player_names"`
}

// GetActiveRooms returns a list of all active multiplayer rooms
func GetActiveRooms(c *fiber.Ctx) error {
	mu.RLock()
	defer mu.RUnlock()

	roomList := make([]DebugRoomInfo, 0, len(rooms))

	for _, room := range rooms {
		room.mu.RLock()

		playerIDs := make([]string, 0, len(room.Players))
		playerNames := make([]string, 0, len(room.Players))

		for _, player := range room.Players {
			playerIDs = append(playerIDs, player.ID)
			playerNames = append(playerNames, player.Username)
		}

		info := DebugRoomInfo{
			RoomCode:    room.Code,
			Host:        room.Host,
			PlayerCount: len(room.Players),
			MaxPlayers:  room.MaxPlayers,
			State:       room.State,
			GameID:      room.GameID,
			GameURL:     room.GameURL,
			PlayerIDs:   playerIDs,
			PlayerNames: playerNames,
		}

		room.mu.RUnlock()
		roomList = append(roomList, info)
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"total_rooms": len(roomList),
		"rooms":       roomList,
		"timestamp":   time.Now(),
	})
}

// GetGameSessions returns a list of all active game sessions
func GetGameSessions(c *fiber.Ctx) error {
	mu.RLock()
	defer mu.RUnlock()

	sessionList := make([]fiber.Map, 0, len(gameSessions))

	for gameID, session := range gameSessions {
		session.mu.RLock()

		authorizedPlayers := make([]string, 0, len(session.AuthorizedPlayers))
		for playerID := range session.AuthorizedPlayers {
			authorizedPlayers = append(authorizedPlayers, playerID)
		}

		info := fiber.Map{
			"game_id":            gameID,
			"room_code":          session.RoomCode,
			"created_at":         session.CreatedAt,
			"expires_at":         session.ExpiresAt,
			"expired":            time.Now().After(session.ExpiresAt),
			"authorized_players": authorizedPlayers,
			"player_count":       len(authorizedPlayers),
		}

		session.mu.RUnlock()
		sessionList = append(sessionList, info)
	}

	return c.JSON(fiber.Map{
		"success":        true,
		"total_sessions": len(sessionList),
		"sessions":       sessionList,
		"timestamp":      time.Now(),
	})
}

// GetRoomByCode returns detailed information about a specific room
func GetRoomByCode(c *fiber.Ctx) error {
	roomCode := c.Params("code")

	if roomCode == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Room code is required",
		})
	}

	mu.RLock()
	room, exists := rooms[roomCode]
	mu.RUnlock()

	if !exists {
		// List all active rooms for debugging
		mu.RLock()
		activeCodes := make([]string, 0, len(rooms))
		for code := range rooms {
			activeCodes = append(activeCodes, code)
		}
		mu.RUnlock()

		return c.Status(404).JSON(fiber.Map{
			"success":      false,
			"error":        "Room not found",
			"room_code":    roomCode,
			"active_rooms": activeCodes,
			"total_active": len(activeCodes),
		})
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	players := make([]fiber.Map, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, fiber.Map{
			"id":         player.ID,
			"username":   player.Username,
			"is_ready":   player.IsReady,
			"is_host":    player.IsHost,
			"is_playing": player.IsPlaying,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"room": fiber.Map{
			"code":             room.Code,
			"host":             room.Host,
			"player_count":     len(room.Players),
			"max_players":      room.MaxPlayers,
			"state":            room.State,
			"game_id":          room.GameID,
			"game_url":         room.GameURL,
			"selected_themes":  room.SelectedThemes,
			"question_count":   room.QuestionCount,
			"time_limit":       room.TimeLimit,
			"current_question": room.CurrentQuestion,
			"players":          players,
		},
	})
}

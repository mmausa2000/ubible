// ~/Documents/CODING/ubible/handlers/multiplayer.go
package handlers

import (
	"crypto/rand"
	"sync"

	"github.com/google/uuid"

	"github.com/gofiber/websocket/v2"
)

type Player struct {
	ID        string
	Username  string
	Conn      *websocket.Conn
	Room      string
	IsReady   bool
	IsHost    bool
	IsPlaying bool
	mu        sync.RWMutex
}

type Room struct {
	Code       string
	Host       string
	Players    map[string]*Player
	MaxPlayers int
	State      string
	mu         sync.RWMutex
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var (
	rooms   = make(map[string]*Room)
	players = make(map[*websocket.Conn]*Player)
	mu      sync.RWMutex
)

func HandleWebSocket(c *websocket.Conn) {
	playerID := c.Query("player_id")
	username := c.Query("username")

	if playerID == "" {
		playerID = generateID()
	}
	if username == "" {
		username = "Player" + playerID[:6]
	}

	player := &Player{
		ID:       playerID,
		Username: username,
		Conn:     c,
	}

	mu.Lock()
	players[c] = player
	mu.Unlock()

	defer func() {
		handleDisconnect(player)
		c.Close()
	}()

	send(c, "connected", map[string]interface{}{
		"player_id": playerID,
		"username":  username,
	})

	for {
		var msg Message
		if err := c.ReadJSON(&msg); err != nil {
			break
		}
		handleMessage(player, msg)
	}
}

func handleMessage(player *Player, msg Message) {
	switch msg.Type {
	case "create_room":
		handleCreateRoom(player, msg.Payload)
	case "join_room":
		handleJoinRoom(player, msg.Payload)
	case "player_ready":
		handlePlayerReady(player)
	case "find_match":
		handleFindMatch(player, msg.Payload)
	case "leave_room":
		handleLeaveRoom(player)
	case "start_game":
		handleStartGame(player)
	}
}

func handleStartGame(player *Player) {
	player.mu.RLock()
	roomCode := player.Room
	player.mu.RUnlock()

	if roomCode == "" {
		return
	}

	mu.RLock()
	room, exists := rooms[roomCode]
	mu.RUnlock()

	if !exists {
		return
	}

	// Ensure only the host can start the game
	if room.Host != player.ID {
		send(player.Conn, "error", map[string]interface{}{"error": "Only host can start the game"})
		return
	}

	startGame(room)
}

func handleCreateRoom(player *Player, payload interface{}) {
	data := parsePayload(payload)
	maxPlayers := getInt(data, "max_players", 10)
	if maxPlayers > 10 {
		maxPlayers = 10
	}

	hostIsPlaying := true
	if val, ok := data["host_is_playing"]; ok {
		if boolVal, ok := val.(bool); ok {
			hostIsPlaying = boolVal
		}
	}

	roomCode := generateRoomCode()
	room := &Room{
		Code:       roomCode,
		Host:       player.ID,
		Players:    make(map[string]*Player),
		MaxPlayers: maxPlayers,
		State:      "waiting",
	}

	mu.Lock()
	rooms[roomCode] = room
	mu.Unlock()

	room.mu.Lock()
	room.Players[player.ID] = player
	room.mu.Unlock()

	player.mu.Lock()
	player.Room = roomCode
	player.IsHost = true
	player.IsReady = true
	player.IsPlaying = hostIsPlaying
	player.mu.Unlock()

	send(player.Conn, "room_created", map[string]interface{}{
		"room_code":   roomCode,
		"host":        room.Host,
		"players":     getPlayerList(room),
		"max_players": maxPlayers,
	})

	broadcastRoomUpdate(room)
}

func handleJoinRoom(player *Player, payload interface{}) {
	data := parsePayload(payload)
	roomCode := getString(data, "room_code", "")

	mu.RLock()
	room, exists := rooms[roomCode]
	mu.RUnlock()

	if !exists {
		send(player.Conn, "error", map[string]interface{}{"error": "Room not found"})
		return
	}

	room.mu.Lock()
	if len(room.Players) >= room.MaxPlayers {
		room.mu.Unlock()
		send(player.Conn, "error", map[string]interface{}{"error": "Room is full"})
		return
	}
	room.Players[player.ID] = player
	room.mu.Unlock()

	player.mu.Lock()
	player.Room = roomCode
	player.IsReady = false
	player.IsPlaying = true
	player.mu.Unlock()

	send(player.Conn, "room_joined", map[string]interface{}{
		"room_code": roomCode,
		"host":      room.Host,
		"players":   getPlayerList(room),
	})

	broadcastToRoom(room, "player_joined", map[string]interface{}{
		"player":       player.Username,
		"player_count": len(room.Players),
		"players":      getPlayerList(room),
	})

	broadcastRoomUpdate(room)
}

func handlePlayerReady(player *Player) {
	player.mu.Lock()
	player.IsReady = true
	roomCode := player.Room
	isHost := player.IsHost
	player.mu.Unlock()

	mu.RLock()
	room, exists := rooms[roomCode]
	mu.RUnlock()

	if !exists {
		return
	}

	broadcastToRoom(room, "player_ready_update", map[string]interface{}{
		"player_id": player.ID,
		"players":   getPlayerList(room),
	})

	room.mu.RLock()
	playingPlayers := 0
	readyPlayers := 0

	for _, p := range room.Players {
		p.mu.RLock()
		if p.IsPlaying {
			playingPlayers++
			if p.IsReady {
				readyPlayers++
			}
		}
		p.mu.RUnlock()
	}

	allReady := playingPlayers >= 2 && readyPlayers == playingPlayers
	room.mu.RUnlock()

	if allReady && isHost {
		startGame(room)
	}
}

func handleFindMatch(player *Player, payload interface{}) {
	send(player.Conn, "searching", map[string]interface{}{"players_waiting": 0})
}

func handleLeaveRoom(player *Player) {
	player.mu.RLock()
	roomCode := player.Room
	player.mu.RUnlock()

	if roomCode == "" {
		return
	}

	mu.RLock()
	room, exists := rooms[roomCode]
	mu.RUnlock()

	if !exists {
		return
	}

	room.mu.Lock()
	delete(room.Players, player.ID)
	playerCount := len(room.Players)

	// Reassign host if host left
	if room.Host == player.ID && playerCount > 0 {
		for _, p := range room.Players {
			room.Host = p.ID
			p.mu.Lock()
			p.IsHost = true
			p.mu.Unlock()
			break
		}
	}
	room.mu.Unlock()

	player.mu.Lock()
	player.Room = ""
	player.mu.Unlock()

	// Delete room if empty
	if playerCount == 0 {
		mu.Lock()
		delete(rooms, roomCode)
		mu.Unlock()
	} else {
		broadcastRoomUpdate(room)
	}
}

func startGame(room *Room) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	playingPlayersList := make([]map[string]interface{}, 0)
	for _, p := range room.Players {
		p.mu.RLock()
		if p.IsPlaying {
			playingPlayersList = append(playingPlayersList, map[string]interface{}{
				"player_id": p.ID,
				"username":  p.Username,
				"is_ready":  p.IsReady,
				"is_host":   p.IsHost,
			})
		}
		p.mu.RUnlock()
	}

	gameID := generateID()

	for _, p := range room.Players {
		p.mu.RLock()
		isPlaying := p.IsPlaying
		p.mu.RUnlock()

		send(p.Conn, "game_start", map[string]interface{}{
			"room_code":  room.Code,
			"players":    playingPlayersList,
			"game_id":    gameID,
			"is_playing": isPlaying,
		})
	}
}

func handleDisconnect(player *Player) {
	handleLeaveRoom(player)
	mu.Lock()
	delete(players, player.Conn)
	mu.Unlock()
}

func broadcastToRoom(room *Room, msgType string, payload interface{}) {
	room.mu.RLock()
	defer room.mu.RUnlock()
	for _, p := range room.Players {
		send(p.Conn, msgType, payload)
	}
}

func broadcastRoomUpdate(room *Room) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	for _, p := range room.Players {
		send(p.Conn, "room_update", map[string]interface{}{
			"room_code":   room.Code,
			"host":        room.Host,
			"players":     getPlayerList(room),
			"max_players": room.MaxPlayers,
		})
	}
}

func getPlayerList(room *Room) []map[string]interface{} {
	list := make([]map[string]interface{}, 0)
	for _, p := range room.Players {
		p.mu.RLock()
		list = append(list, map[string]interface{}{
			"player_id":  p.ID,
			"username":   p.Username,
			"is_ready":   p.IsReady,
			"is_host":    p.IsHost,
			"is_playing": p.IsPlaying,
		})
		p.mu.RUnlock()
	}
	return list
}

func send(conn *websocket.Conn, msgType string, payload interface{}) {
	msg := Message{Type: msgType, Payload: payload}
	conn.WriteJSON(msg)
}

func generateRoomCode() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

func generateID() string {
	return uuid.NewString()
}

func parsePayload(payload interface{}) map[string]interface{} {
	if payload == nil {
		return make(map[string]interface{})
	}
	if data, ok := payload.(map[string]interface{}); ok {
		return data
	}
	return make(map[string]interface{})
}

func getString(data map[string]interface{}, key string, defaultVal string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

func getInt(data map[string]interface{}, key string, defaultVal int) int {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return defaultVal
}

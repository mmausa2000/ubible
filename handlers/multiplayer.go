// ~/Documents/CODING/ubible/handlers/multiplayer.go
package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"ubible/database"
	"ubible/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const (
	// WebSocket timeouts
	writeWait  = 10 * time.Second // Time allowed to write a message
	pingPeriod = 15 * time.Second // Send pings at this interval

	// Send channel buffer size
	sendBufferSize = 256
)

type Player struct {
	ID        string          // UUID for in-game identity
	UserID    *uint           // Database user ID (nil for guests)
	Username  string
	IsGuest   bool            // True for unauthenticated players
	Conn      *websocket.Conn
	Room      string
	IsReady   bool
	IsHost    bool
	IsPlaying bool
	send      chan Message   // Buffered channel for outbound messages
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
}

type Room struct {
	Code           string
	Host           string
	Players        map[string]*Player
	MaxPlayers     int
	State          string
	SelectedThemes []int  `json:"selected_themes"` // Host's selected theme IDs
	QuestionCount  int    `json:"question_count"`
	TimeLimit      int    `json:"time_limit"`
	GameID         string `json:"game_id"`    // Unique game identifier
	GameURL        string `json:"game_url"`   // Unique game URL
	GameToken      string `json:"game_token"` // Secure access token

	// Per-question state tracking
	CurrentQuestion   int             `json:"current_question"` // 0-indexed
	QuestionStartTime time.Time       `json:"-"`                // When current question started (for timeout)
	QuestionTimer     *time.Timer     `json:"-"`                // Timer for current question
	PlayersAnswered   map[string]bool `json:"players_answered"` // playerID → has answered current Q
	PlayerScores      map[string]int  `json:"player_scores"`    // playerID → current score

	mu sync.RWMutex
}

// GameSession stores secure game session data
type GameSession struct {
	GameID            string
	RoomCode          string
	Token             string
	AuthorizedPlayers map[string]bool // playerID -> authorized
	CreatedAt         time.Time
	ExpiresAt         time.Time
	mu                sync.RWMutex
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var (
	rooms        = make(map[string]*Room)
	players      = make(map[*websocket.Conn]*Player)
	gameSessions = make(map[string]*GameSession) // gameID -> GameSession
	mu           sync.RWMutex
)

// WebSocketHandler is a pure net/http handler for WebSocket connections
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Authenticate using JWT from cookie or Authorization header
	var userID *uint
	var username string
	var isGuest bool = true

	// Try to get JWT token from Authorization header
	authHeader := r.Header.Get("Authorization")
	var tokenString string

	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString = parts[1]
		}
	}

	// Fall back to cookie
	if tokenString == "" {
		if cookie, err := r.Cookie("token"); err == nil {
			tokenString = cookie.Value
		}
	}

	// Parse JWT if present
	if tokenString != "" {
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "ubible-secret-change-in-production"
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("invalid signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				// Extract user info from JWT
				if userIDVal, ok := claims["user_id"].(float64); ok {
					uid := uint(userIDVal)
					userID = &uid
				}
				if usernameVal, ok := claims["username"].(string); ok {
					username = usernameVal
				}
				if isGuestVal, ok := claims["is_guest"].(bool); ok {
					isGuest = isGuestVal
				}
			}
		}
	}

	// Handle WebSocket upgrade and connection
	handleWebSocket(w, r, userID, username, isGuest)
}

// handleWebSocket handles nhooyr.io/websocket connections (net/http compatible)
func handleWebSocket(w http.ResponseWriter, r *http.Request, userID *uint, username string, isGuest bool) {
	// Accept WebSocket connection
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // TODO: Add proper origin checking in production
	})
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}

	ctx := r.Context()

	// Generate player ID
	playerID := r.URL.Query().Get("player_id")
	if playerID == "" {
		playerID = GeneratePlayerID()
	}

	// Use authenticated username, fall back to query param for guests
	if username == "" {
		username = r.URL.Query().Get("username")
		if username == "" {
			username = "Guest" + playerID[:6]
		}
	}

	playerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	player := &Player{
		ID:       playerID,
		UserID:   userID,
		Username: username,
		IsGuest:  isGuest,
		Conn:     conn,
		send:     make(chan Message, sendBufferSize),
		ctx:      playerCtx,
		cancel:   cancel,
	}

	mu.Lock()
	players[conn] = player
	mu.Unlock()

	log.Printf("🎮 Player connected: %s (ID: %s, UserID: %v, Guest: %v)", username, playerID, userID, isGuest)

	// Send initial connection message
	player.sendMessage("connected", map[string]interface{}{
		"player_id": playerID,
		"username":  username,
		"is_guest":  isGuest,
		"user_id":   userID,
	})

	// Start write pump in separate goroutine
	go player.writePump()

	// Start read pump (blocking)
	player.readPump()

	// Cleanup when connection closes
	mu.Lock()
	delete(players, conn)
	mu.Unlock()

	if player.Room != "" {
		handleLeaveRoom(player)
	}

	close(player.send)
	log.Printf("🔌 Player disconnected: %s (ID: %s, UserID: %v)", player.Username, player.ID, player.UserID)
}

// Deprecated: HandleFiberWebSocketWithAuth - use HandleWebSocket instead
func HandleFiberWebSocketWithAuth(conn *websocket.Conn, playerID, username string, userID *uint, isGuest bool) {
	if playerID == "" {
		playerID = generateID()
	}
	if username == "" {
		if userID != nil {
			username = "User" + string(rune(*userID))
		} else {
			username = "Guest" + playerID[:6]
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	player := &Player{
		ID:       playerID,
		UserID:   userID,
		Username: username,
		IsGuest:  isGuest,
		Conn:     conn,
		send:     make(chan Message, sendBufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}

	mu.Lock()
	players[conn] = player
	mu.Unlock()

	log.Printf("🎮 Player connected: %s (ID: %s, UserID: %v, Guest: %v)", username, playerID, userID, isGuest)

	// Send initial connection message
	player.sendMessage("connected", map[string]interface{}{
		"player_id": playerID,
		"username":  username,
		"is_guest":  isGuest,
		"user_id":   userID,
	})

	// Start write pump in separate goroutine
	go player.writePump()

	// Start read pump (blocking)
	player.readPump()

	// Cleanup when connection closes
	mu.Lock()
	delete(players, conn)
	mu.Unlock()

	if player.Room != "" {
		handleLeaveRoom(player)
	}

	close(player.send)
	log.Printf("🔌 Player disconnected: %s (ID: %s, UserID: %v)", player.Username, player.ID, player.UserID)
}

// HandleFiberWebSocket handles WebSocket connections using Fiber's WebSocket (backward compatibility)
// Deprecated: Use HandleFiberWebSocketWithAuth for authenticated connections
func HandleFiberWebSocket(conn *websocket.Conn, playerID, username string) {
	HandleFiberWebSocketWithAuth(conn, playerID, username, nil, true)
}

// OLD NHOOYR-BASED HANDLERS - COMMENTED OUT (Using Fiber WebSocket now)
/*
func HandleWebSocket(conn *websocket.Conn, playerID, username string) {
	if playerID == "" {
		playerID = generateID()
	}
	if username == "" {
		username = "Player" + playerID[:6]
	}

	ctx, cancel := context.WithCancel(context.Background())

	player := &Player{
		ID:       playerID,
		Username: username,
		Conn:     conn,
		send:     make(chan Message, sendBufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}

	mu.Lock()
	players[conn] = player
	mu.Unlock()

	// Send initial connection message
	player.sendMessage("connected", map[string]interface{}{
		"player_id": playerID,
		"username":  username,
	})

	// Start write pump in separate goroutine
	go player.writePump()

	// Start ping ticker in separate goroutine
	go player.pingPump()

	// Read pump runs in current goroutine (blocks until disconnect)
	player.readPump()

	// Cleanup
	handleDisconnect(player)
}

// readPump handles incoming messages from the WebSocket connection (nhooyr)
func (p *Player) readPump() {
	defer func() {
		p.cancel()
		p.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		var msg Message
		err := wsjson.Read(p.ctx, p.Conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				log.Printf("WebSocket closed normally for player %s", p.ID)
			} else {
				log.Printf("WebSocket error for player %s: %v", p.ID, err)
			}
			break
		}

		handleMessage(p, msg)
	}
}

// writePump handles outgoing messages to the WebSocket connection (nhooyr)
func (p *Player) writePump() {
	defer func() {
		p.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case msg, ok := <-p.send:
			if !ok {
				return
			}

			ctx, cancel := context.WithTimeout(p.ctx, writeWait)
			err := wsjson.Write(ctx, p.Conn, msg)
			cancel()

			if err != nil {
				log.Printf("Write error for player %s: %v", p.ID, err)
				return
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// pingPump sends periodic ping messages to keep connection alive (nhooyr)
func (p *Player) pingPump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(p.ctx, writeWait)
			err := p.Conn.Ping(ctx)
			cancel()

			if err != nil {
				log.Printf("Ping error for player %s: %v", p.ID, err)
				return
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// sendMessage queues a message to be sent (non-blocking with bounded queue) (nhooyr)
func (p *Player) sendMessage(msgType string, payload interface{}) {
	msg := Message{Type: msgType, Payload: payload}

	select {
	case p.send <- msg:
		// Message queued successfully
	default:
		// Send buffer full - drop message and log warning
		log.Printf("⚠️ Send buffer full for player %s, dropping non-critical message type: %s", p.ID, msgType)
	}
}
*/

// sendMessage queues a message to be sent to the player via WebSocket
func (p *Player) sendMessage(msgType string, payload interface{}) {
	msg := Message{Type: msgType, Payload: payload}

	select {
	case p.send <- msg:
		// Message queued successfully
	default:
		// Send buffer full - drop message and log warning
		log.Printf("⚠️ Send buffer full for player %s, dropping message type: %s", p.ID, msgType)
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
	case "reconnect":
		handleReconnect(player, msg.Payload)
	case "player_quit":
		handlePlayerQuit(player, msg.Payload)
	case "submit_answer":
		handleSubmitAnswer(player, msg.Payload)
	case "opponent_answered":
		// Legacy event - treat same as submit_answer
		handleSubmitAnswer(player, msg.Payload)
	case "ping":
		// Send pong response for latency measurement
		player.sendMessage("pong", map[string]interface{}{})
	}
}

func handleStartGame(player *Player) {
	// Must be host
	player.mu.RLock()
	roomCode := player.Room
	isHost := player.IsHost
	player.mu.RUnlock()

	if roomCode == "" {
		player.sendMessage("error", map[string]interface{}{"error": "Not in a room"})
		return
	}

	if !isHost {
		player.sendMessage("error", map[string]interface{}{"error": "Only host can start the game"})
		return
	}

	// Load room
	mu.RLock()
	room, exists := rooms[roomCode]
	mu.RUnlock()

	if !exists {
		player.sendMessage("error", map[string]interface{}{"error": "Room not found"})
		return
	}

	// Require at least 2 playing players and all must be ready
	room.mu.RLock()
	playingCount := 0
	readyCount := 0
	for _, p := range room.Players {
		p.mu.RLock()
		if p.IsPlaying {
			playingCount++
			if p.IsReady {
				readyCount++
			}
		}
		p.mu.RUnlock()
	}
	room.mu.RUnlock()

	if playingCount < 2 {
		player.sendMessage("error", map[string]interface{}{
			"error":   "Need at least 2 players to start",
			"message": "Waiting for more players to join...",
		})
		return
	}

	if readyCount != playingCount {
		player.sendMessage("error", map[string]interface{}{
			"error":   "All players must be ready",
			"message": fmt.Sprintf("%d/%d players ready", readyCount, playingCount),
		})
		return
	}

	log.Printf("🎮 Starting game in room %s with %d ready players", roomCode, playingCount)
	startGame(room)
}

func handleCreateRoom(player *Player, payload interface{}) {
	log.Printf("🏠 [CREATE_ROOM] Player %s (%s) attempting to create room", player.ID, player.Username)

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

	// Get host's theme settings
	selectedThemes := getIntArray(data, "theme_ids")
	if len(selectedThemes) == 0 {
		selectedThemes = getIntArray(data, "selected_themes")
	}
	questionCount := getInt(data, "question_count", 10)
	timeLimit := getInt(data, "time_limit", 10)

	log.Printf("🏠 [CREATE_ROOM] Settings: maxPlayers=%d, themes=%v, questions=%d, timeLimit=%d, hostPlaying=%v",
		maxPlayers, selectedThemes, questionCount, timeLimit, hostIsPlaying)

	roomCode := generateRoomCode()
	gameID := generateID()
	gameToken := generateSecureToken()

	room := &Room{
		Code:            roomCode,
		Host:            player.ID,
		Players:         make(map[string]*Player),
		MaxPlayers:      maxPlayers,
		State:           "waiting",
		SelectedThemes:  selectedThemes,
		QuestionCount:   questionCount,
		TimeLimit:       timeLimit,
		GameID:          gameID,
		GameURL:         "/game/" + gameID,
		GameToken:       gameToken,
		CurrentQuestion: 0,
		PlayersAnswered: make(map[string]bool),
		PlayerScores:    make(map[string]int),
	}

	// Create game session for access control
	gameSession := &GameSession{
		GameID:            gameID,
		RoomCode:          roomCode,
		Token:             gameToken,
		AuthorizedPlayers: make(map[string]bool),
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(24 * time.Hour), // 24 hour expiry
	}
	gameSession.AuthorizedPlayers[player.ID] = true

	mu.Lock()
	gameSessions[gameID] = gameSession
	mu.Unlock()

	mu.Lock()
	rooms[roomCode] = room
	totalRooms := len(rooms)
	mu.Unlock()

	log.Printf("✅ [CREATE_ROOM] Room created: code=%s, gameID=%s, gameURL=%s", roomCode, gameID, room.GameURL)
	log.Printf("📊 [ROOM_STATS] Total active rooms: %d", totalRooms)

	room.mu.Lock()
	room.Players[player.ID] = player
	room.mu.Unlock()

	player.mu.Lock()
	player.Room = roomCode
	player.IsHost = true
	player.IsReady = true
	player.IsPlaying = hostIsPlaying
	player.mu.Unlock()

	log.Printf("✅ [CREATE_ROOM] Host %s added to room %s", player.Username, roomCode)

	player.sendMessage("room_created", map[string]interface{}{
		"room_code":   roomCode,
		"host":        room.Host,
		"players":     getPlayerList(room),
		"max_players": maxPlayers,
		"game_url":    room.GameURL,
		"game_token":  room.GameToken,
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
		player.sendMessage("error", map[string]interface{}{"error": "Room not found"})
		return
	}

	room.mu.Lock()
	if len(room.Players) >= room.MaxPlayers {
		room.mu.Unlock()
		player.sendMessage("error", map[string]interface{}{"error": "Room is full"})
		return
	}
	room.Players[player.ID] = player
	room.mu.Unlock()

	player.mu.Lock()
	player.Room = roomCode
	player.IsReady = false
	player.IsPlaying = true
	player.mu.Unlock()

	// Authorize player for game session
	gameID := extractGameIDFromURL(room.GameURL)
	mu.RLock()
	gameSession, exists := gameSessions[gameID]
	mu.RUnlock()

	if exists {
		gameSession.mu.Lock()
		gameSession.AuthorizedPlayers[player.ID] = true
		gameSession.mu.Unlock()
	}

	player.sendMessage("room_joined", map[string]interface{}{
		"room_code":       roomCode,
		"host":            room.Host,
		"players":         getPlayerList(room),
		"selected_themes": room.SelectedThemes,
		"question_count":  room.QuestionCount,
		"time_limit":      room.TimeLimit,
		"game_url":        room.GameURL,
		"game_token":      room.GameToken,
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

func handleFindMatch(player *Player, _ interface{}) {
	player.sendMessage("searching", map[string]interface{}{"players_waiting": 0})
}

// handleReconnect allows players to rejoin their game after navigating to the game page
func handleReconnect(player *Player, payload interface{}) {
	data := parsePayload(payload)

	gameID, ok := data["game_id"].(string)
	if !ok || gameID == "" {
		log.Printf("⚠️  Player %s reconnect failed: missing game_id", player.ID)
		player.sendMessage("error", map[string]interface{}{"error": "Missing game_id"})
		return
	}

	// Find the room associated with this game ID
	mu.RLock()
	var targetRoom *Room
	for _, room := range rooms {
		if room.GameID == gameID {
			targetRoom = room
			break
		}
	}
	mu.RUnlock()

	if targetRoom == nil {
		log.Printf("⚠️  Player %s reconnect failed: game %s not found", player.ID, gameID)
		player.sendMessage("error", map[string]interface{}{"error": "Game not found"})
		return
	}

	// Verify player is authorized for this game
	mu.RLock()
	session, exists := gameSessions[gameID]
	mu.RUnlock()

	if !exists {
		log.Printf("⚠️  Player %s reconnect failed: no session for game %s", player.ID, gameID)
		player.sendMessage("error", map[string]interface{}{"error": "Game session not found"})
		return
	}

	session.mu.RLock()
	authorized := session.AuthorizedPlayers[player.ID]
	session.mu.RUnlock()

	if !authorized {
		log.Printf("⚠️  Player %s not authorized for game %s", player.ID, gameID)
		player.sendMessage("error", map[string]interface{}{"error": "Not authorized for this game"})
		return
	}

	// Re-associate player with room
	player.mu.Lock()
	player.Room = targetRoom.Code
	player.mu.Unlock()

	// Update player in room's player map
	targetRoom.mu.Lock()
	targetRoom.Players[player.ID] = player
	targetRoom.mu.Unlock()

	log.Printf("✅ Player %s reconnected to game %s (room %s)", player.ID, gameID, targetRoom.Code)

	// Prepare current game state
	targetRoom.mu.RLock()
	currentQuestion := targetRoom.CurrentQuestion
	questionCount := targetRoom.QuestionCount
	roomState := targetRoom.State
	playerScores := make(map[string]int)
	for pid, score := range targetRoom.PlayerScores {
		playerScores[pid] = score
	}
	targetRoom.mu.RUnlock()

	isGameStarted := roomState == "playing"

	// Send confirmation with full game state
	player.sendMessage("reconnected", map[string]interface{}{
		"game_id":          gameID,
		"room_code":        targetRoom.Code,
		"success":          true,
		"game_started":     isGameStarted,
		"current_question": currentQuestion,
		"question_count":   questionCount,
		"player_scores":    playerScores,
		"players":          getPlayerList(targetRoom),
	})

	// If game is in progress, send current question to help client sync
	if isGameStarted && currentQuestion < questionCount {
		log.Printf("📍 Sending current question sync: Q%d/%d", currentQuestion+1, questionCount)
		player.sendMessage("question_sync", map[string]interface{}{
			"question_index": currentQuestion,
			"total":          questionCount,
		})
	}
}

// handleSubmitAnswer processes player answers and broadcasts to room
func handleSubmitAnswer(player *Player, payload interface{}) {
	player.mu.RLock()
	roomCode := player.Room
	player.mu.RUnlock()

	if roomCode == "" {
		log.Printf("⚠️  Player %s tried to submit answer but not in a room", player.ID)
		return
	}

	mu.RLock()
	room, exists := rooms[roomCode]
	mu.RUnlock()

	if !exists {
		log.Printf("⚠️  Player %s tried to submit answer but room %s doesn't exist", player.ID, roomCode)
		return
	}

	// Verify player is in this room and playing
	room.mu.RLock()
	_, inRoom := room.Players[player.ID]
	room.mu.RUnlock()

	if !inRoom {
		log.Printf("⚠️  Player %s not authorized for room %s", player.ID, roomCode)
		return
	}

	// Parse answer data - accept both snake_case and camelCase
	data := parsePayload(payload)
	questionIndex := getInt(data, "questionIndex", getInt(data, "question_index", -1))
	isCorrect := getBool(data, "correct", getBool(data, "is_correct", false))
	score := getInt(data, "score", 0)

	room.mu.Lock()

	// Validate question index matches current question
	if questionIndex != room.CurrentQuestion {
		log.Printf("⚠️  Player %s submitted answer for Q%d but room is on Q%d", player.ID, questionIndex, room.CurrentQuestion)
		room.mu.Unlock()
		return
	}

	// Mark player as answered for current question
	room.PlayersAnswered[player.ID] = true

	// Update player score
	if _, exists := room.PlayerScores[player.ID]; !exists {
		room.PlayerScores[player.ID] = 0
	}
	room.PlayerScores[player.ID] += score

	// Count playing players who have answered
	playingCount := 0
	answeredCount := 0
	for _, p := range room.Players {
		p.mu.RLock()
		if p.IsPlaying {
			playingCount++
			if room.PlayersAnswered[p.ID] {
				answeredCount++
			}
		}
		p.mu.RUnlock()
	}

	allAnswered := playingCount > 0 && answeredCount == playingCount

	room.mu.Unlock()

	// Broadcast this player's answer to all players in room
	broadcastToRoom(room, "answer_submitted", map[string]interface{}{
		"player_id":      player.ID,
		"username":       player.Username,
		"question_index": questionIndex,
		"correct":        isCorrect,
		"score":          score,
		"all_answered":   allAnswered,
	})

	// Also send as opponent_answered for legacy compatibility
	broadcastToRoom(room, "opponent_answered", map[string]interface{}{
		"playerId":      player.ID,
		"username":      player.Username,
		"questionIndex": questionIndex,
		"correct":       isCorrect,
		"score":         score,
		"allAnswered":   allAnswered,
	})

	log.Printf("📝 Player %s answered Q%d (correct: %v, score: %d) - %d/%d answered, allAnswered=%v",
		player.ID, questionIndex, isCorrect, score, answeredCount, playingCount, allAnswered)

	// If all players have answered, prepare for next question
	if allAnswered {
		log.Printf("✅ All players answered Q%d, advancing to next question", questionIndex)

		room.mu.Lock()
		room.CurrentQuestion++

		// Clear answered flags for next question
		room.PlayersAnswered = make(map[string]bool)

		nextQuestion := room.CurrentQuestion
		totalQuestions := room.QuestionCount
		gameComplete := nextQuestion >= totalQuestions

		room.mu.Unlock()

		if gameComplete {
			log.Printf("🏁 Game complete in room %s", roomCode)
			handleGameComplete(room)
		} else {
			log.Printf("➡️  Room %s advancing to Q%d/%d", roomCode, nextQuestion+1, totalQuestions)

			// Broadcast ready for next question
			log.Printf("📢 Broadcasting next_question event: Q%d/%d to room %s", nextQuestion, totalQuestions, roomCode)
			broadcastToRoom(room, "next_question", map[string]interface{}{
				"question_index": nextQuestion,
				"total":          totalQuestions,
			})

			// Start timer for next question
			startQuestionTimer(room)
		}
	}
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

// handlePlayerQuit handles when a player quits mid-game
func handlePlayerQuit(player *Player, _ interface{}) {
	player.mu.RLock()
	roomCode := player.Room
	player.mu.RUnlock()

	if roomCode == "" {
		log.Printf("⚠️  Player %s tried to quit but not in a room", player.ID)
		return
	}

	mu.RLock()
	room, exists := rooms[roomCode]
	mu.RUnlock()

	if !exists {
		log.Printf("⚠️  Player %s tried to quit but room %s doesn't exist", player.ID, roomCode)
		return
	}

	log.Printf("🚪 Player %s (%s) quit from room %s", player.ID, player.Username, roomCode)

	// Mark player as not playing (but keep in room for stats)
	player.mu.Lock()
	player.IsPlaying = false
	player.mu.Unlock()

	// Broadcast to other players
	broadcastToRoom(room, "player_quit", map[string]interface{}{
		"player_id": player.ID,
		"username":  player.Username,
	})

	// Also send opponent_left for backward compatibility
	broadcastToRoom(room, "opponent_left", map[string]interface{}{
		"player_id": player.ID,
		"username":  player.Username,
	})

	// Check if game should continue
	room.mu.RLock()
	playingCount := 0
	for _, p := range room.Players {
		p.mu.RLock()
		if p.IsPlaying {
			playingCount++
		}
		p.mu.RUnlock()
	}
	room.mu.RUnlock()

	switch playingCount {
	case 0:
		// No players left playing, end the game
		log.Printf("🏁 Game in room %s ended - all players quit", roomCode)
		mu.Lock()
		delete(rooms, roomCode)
		mu.Unlock()
	case 1:
		// Only one player left, they can continue solo
		log.Printf("🎯 Game in room %s continues with 1 player", roomCode)
	}

	// Player stays connected but leaves the room
	handleLeaveRoom(player)
}

// fetchQuestionsForRoom retrieves questions from database for multiplayer game
func fetchQuestionsForRoom(room *Room) []map[string]interface{} {
	db := database.GetDB()
	if db == nil {
		log.Printf("⚠️  Database not available for room %s", room.Code)
		return []map[string]interface{}{}
	}

	// Get question count and theme settings from room
	questionCount := room.QuestionCount
	if questionCount == 0 {
		questionCount = 10 // default
	}

	// Build query
	query := db.Model(&models.Question{}).Preload("Theme")

	// Filter by selected themes if any
	if len(room.SelectedThemes) > 0 {
		query = query.Where("theme_id IN ?", room.SelectedThemes)
	}

	// Fetch all matching questions
	var questions []models.Question
	if err := query.Find(&questions).Error; err != nil {
		log.Printf("⚠️  Error fetching questions for room %s: %v", room.Code, err)
		return []map[string]interface{}{}
	}

	// Use deterministic seed based on GameID (fallback to room.Code if empty)
	seedString := room.GameID
	if seedString == "" {
		seedString = room.Code
	}
	sum := sha256.Sum256([]byte(seedString))
	seed := int64(binary.BigEndian.Uint64(sum[:8]))
	rng := mathrand.New(mathrand.NewSource(seed))

	// Shuffle questions deterministically
	rng.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})

	log.Printf("🎲 Shuffled %d questions for game %s with deterministic seed", len(questions), seedString)

	// Take only the requested number of questions
	if len(questions) > questionCount {
		questions = questions[:questionCount]
	}

	// Convert to response format
	result := make([]map[string]interface{}, 0, len(questions))
	for _, q := range questions {
		var wrongAnswers []string
		if q.WrongAnswers != "" {
			if err := json.Unmarshal([]byte(q.WrongAnswers), &wrongAnswers); err != nil {
				log.Printf("⚠️  Error unmarshaling wrong answers: %v", err)
				wrongAnswers = []string{}
			}
		}

		// Build 4 options (1 correct + 3 wrong)
		options := make([]string, 0, 4)
		options = append(options, q.CorrectAnswer)

		// Add wrong answers
		for i := 0; i < 3 && i < len(wrongAnswers); i++ {
			options = append(options, wrongAnswers[i])
		}

		// Shuffle options
		rng.Shuffle(len(options), func(i, j int) {
			options[i], options[j] = options[j], options[i]
		})

		themeName := ""
		if q.Theme.ID != 0 {
			themeName = q.Theme.Name
		}

		result = append(result, map[string]interface{}{
			"id":             q.ID,
			"theme_id":       q.ThemeID,
			"theme_name":     themeName,
			"text":           q.Text,
			"correct_answer": q.CorrectAnswer,
			"options":        options,
			"reference":      q.Reference,
			"difficulty":     q.Difficulty,
		})
	}

	return result
}

// startQuestionTimer starts a timer for the current question
func startQuestionTimer(room *Room) {
	room.mu.Lock()

	// Stop any existing timer
	if room.QuestionTimer != nil {
		room.QuestionTimer.Stop()
	}

	timeLimit := room.TimeLimit
	if timeLimit == 0 {
		timeLimit = 10 // default 10 seconds
	}

	room.QuestionStartTime = time.Now()
	currentQuestion := room.CurrentQuestion
	room.mu.Unlock()

	// Create timer for question timeout
	room.mu.Lock()
	room.QuestionTimer = time.AfterFunc(time.Duration(timeLimit)*time.Second, func() {
		handleQuestionTimeout(room, currentQuestion)
	})
	room.mu.Unlock()

	log.Printf("⏱️  Started timer for Q%d in room %s (%ds)", currentQuestion+1, room.Code, timeLimit)
}

// handleQuestionTimeout handles when time runs out for a question
func handleQuestionTimeout(room *Room, expectedQuestion int) {
	room.mu.Lock()

	// Verify we're still on the same question (prevent race conditions)
	if room.CurrentQuestion != expectedQuestion {
		room.mu.Unlock()
		return
	}

	log.Printf("⏰ Question %d timed out in room %s", expectedQuestion+1, room.Code)

	// Count how many players haven't answered
	unansweredCount := 0
	playingCount := 0
	for _, p := range room.Players {
		p.mu.RLock()
		if p.IsPlaying {
			playingCount++
			if !room.PlayersAnswered[p.ID] {
				unansweredCount++
			}
		}
		p.mu.RUnlock()
	}

	log.Printf("⏰ Timeout: %d/%d players didn't answer", unansweredCount, playingCount)

	// Clear PlayersAnswered for next question
	room.PlayersAnswered = make(map[string]bool)

	// Check if there are more questions
	questionsRemaining := room.QuestionCount - (room.CurrentQuestion + 1)

	if questionsRemaining > 0 {
		// Advance to next question
		room.CurrentQuestion++
		nextQuestion := room.CurrentQuestion
		totalQuestions := room.QuestionCount
		room.mu.Unlock()

		log.Printf("➡️  Room %s auto-advancing to Q%d/%d (timeout)", room.Code, nextQuestion+1, totalQuestions)

		// Broadcast next question
		broadcastToRoom(room, "next_question", map[string]interface{}{
			"question_index": nextQuestion,
			"total":          totalQuestions,
			"reason":         "timeout",
		})

		// Start timer for next question
		startQuestionTimer(room)
	} else {
		// Game complete
		room.mu.Unlock()
		log.Printf("🏁 Game complete in room %s (timeout on last question)", room.Code)
		handleGameComplete(room)
	}
}

// handleGameComplete handles end-of-game logic and persists results
func handleGameComplete(room *Room) {
	room.mu.Lock()

	// Stop timer if running
	if room.QuestionTimer != nil {
		room.QuestionTimer.Stop()
		room.QuestionTimer = nil
	}

	// Collect final scores and determine winner
	finalScores := make([]map[string]interface{}, 0)
	maxScore := 0
	winnerID := ""

	for pid, score := range room.PlayerScores {
		if p, exists := room.Players[pid]; exists {
			finalScores = append(finalScores, map[string]interface{}{
				"player_id": pid,
				"username":  p.Username,
				"score":     score,
			})

			if score > maxScore {
				maxScore = score
				winnerID = pid
			}
		}
	}

	roomCode := room.Code
	questionCount := room.QuestionCount
	timeLimit := room.TimeLimit
	room.mu.Unlock()

	log.Printf("🏁 Game complete in room %s - Final scores: %+v, Winner: %s", roomCode, finalScores, winnerID)

	// Broadcast game complete
	broadcastToRoom(room, "game_complete", map[string]interface{}{
		"final_scores": finalScores,
		"winner_id":    winnerID,
	})

	// Persist game results to database (async)
	go func() {
		db := database.GetDB()
		if db == nil {
			log.Printf("⚠️  Cannot persist game results - database not available")
			return
		}

		// Process results for each player
		for pid, score := range room.PlayerScores {
			room.mu.RLock()
			p, exists := room.Players[pid]
			room.mu.RUnlock()

			if !exists {
				continue
			}

			// Skip guests - they don't have persistent accounts
			if p.IsGuest || p.UserID == nil {
				log.Printf("⏭️  Skipping result persistence for guest player %s (%s)", pid, p.Username)
				continue
			}

			userID := *p.UserID
			won := pid == winnerID
			isPerfect := score == (questionCount * 100) // Assuming 100 points per question
			correctAnswers := score / 100               // Rough estimate based on 100 points per correct answer

			// Calculate XP and FP
			xp := score / 10
			faithPoints := score / 20
			if won {
				xp += 50 // Bonus for winning
				faithPoints += 25
			}
			if isPerfect {
				xp += 100 // Bonus for perfect game
				faithPoints += 50
			}

			// Start transaction for atomicity
			tx := db.Begin()

			// Create attempt record
			attempt := models.Attempt{
				UserID:         userID,
				ThemeID:        0, // Multiplayer games may use multiple themes
				Score:          score,
				CorrectAnswers: correctAnswers,
				TotalQuestions: questionCount,
				TimeElapsed:    questionCount * timeLimit,
				IsPerfect:      isPerfect,
				Difficulty:     "multiplayer",
				XPEarned:       xp,
				FPEarned:       faithPoints,
			}

			if err := tx.Create(&attempt).Error; err != nil {
				tx.Rollback()
				log.Printf("⚠️  Failed to persist game result for player %s (%s): %v", pid, p.Username, err)
				continue
			}

			// Update user progression (wins, losses, XP, FP, rating)
			var user models.User
			if err := tx.First(&user, userID).Error; err != nil {
				tx.Rollback()
				log.Printf("⚠️  Failed to load user %d for progression update: %v", userID, err)
				continue
			}

			oldLevel := user.Level
			user.TotalGames++

			if won {
				user.Wins++
				user.CurrentStreak++
				if user.CurrentStreak > user.BestStreak {
					user.BestStreak = user.CurrentStreak
				}
			} else {
				user.Losses++
				user.CurrentStreak = 0
			}

			if isPerfect {
				user.PerfectGames++
			}

			user.XP += xp
			user.FaithPoints += faithPoints

			// Simple rating adjustment (ELO-like)
			if won {
				user.Rating += 0.1
			} else {
				user.Rating -= 0.05
			}

			// Keep rating in bounds
			if user.Rating > 10.0 {
				user.Rating = 10.0
			} else if user.Rating < 0.0 {
				user.Rating = 0.0
			}

			// Handle level ups
			for {
				xpNeeded := (user.Level * user.Level * 100) // Simple level formula
				if user.XP >= xpNeeded && user.Level < 100 {
					user.Level++
					user.XP -= xpNeeded
					levelReward := 50 + (user.Level * 10)
					user.FaithPoints += levelReward
					log.Printf("🎉 Player %s leveled up to %d!", p.Username, user.Level)
				} else {
					break
				}
			}

			if err := tx.Save(&user).Error; err != nil {
				tx.Rollback()
				log.Printf("⚠️  Failed to update user progression for %s: %v", p.Username, err)
				continue
			}

			if err := tx.Commit().Error; err != nil {
				log.Printf("⚠️  Failed to commit game results for %s: %v", p.Username, err)
				continue
			}

			log.Printf("✅ Persisted game for %s (UserID: %d) - Score: %d, Won: %v, XP: +%d, FP: +%d, Level: %d→%d",
				p.Username, userID, score, won, xp, faithPoints, oldLevel, user.Level)
		}
	}()
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

	// Use the existing GameID from room (already stored in gameSessions)
	gameID := room.GameID

	log.Printf("🎮 Starting game %s for room %s with %d players", gameID, room.Code, len(playingPlayersList))

	// Fetch questions for the game (synchronized for all players)
	questions := fetchQuestionsForRoom(room)
	log.Printf("📚 Fetched %d questions for game %s", len(questions), gameID)

	// Non-blocking broadcast of game start
	for _, p := range room.Players {
		p.mu.RLock()
		isPlaying := p.IsPlaying
		p.mu.RUnlock()

		p.sendMessage("game_start", map[string]interface{}{
			"room_code":  room.Code,
			"players":    playingPlayersList,
			"game_id":    gameID,
			"is_playing": isPlaying,
			"questions":  questions, // ✅ Include synchronized questions
		})
	}

	// Start timer for first question
	startQuestionTimer(room)
}

// broadcastToRoom sends a message to all players in a room (non-blocking)
func broadcastToRoom(room *Room, msgType string, payload interface{}) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	// Non-blocking broadcast - one slow player doesn't affect others
	for _, p := range room.Players {
		p.sendMessage(msgType, payload)
	}
}

// broadcastRoomUpdate sends room state to all players (non-blocking)
func broadcastRoomUpdate(room *Room) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	payload := map[string]interface{}{
		"room_code":       room.Code,
		"host":            room.Host,
		"players":         getPlayerList(room),
		"max_players":     room.MaxPlayers,
		"selected_themes": room.SelectedThemes,
		"question_count":  room.QuestionCount,
		"time_limit":      room.TimeLimit,
	}

	// Non-blocking broadcast
	for _, p := range room.Players {
		p.sendMessage("room_update", payload)
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

// GeneratePlayerID creates a unique player ID
func GeneratePlayerID() string {
	return uuid.NewString()
}

// Deprecated: Use GeneratePlayerID instead
func generateID() string {
	return GeneratePlayerID()
}

func parsePayload(payload interface{}) map[string]interface{} {
	if payload == nil {
		return make(map[string]interface{})
	}
	if data, ok := payload.(map[string]interface{}); ok {
		return data
	}
	// Try to parse as JSON if it's a string or bytes
	if str, ok := payload.(string); ok {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(str), &data); err == nil {
			return data
		}
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

func getBool(data map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := data[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultVal
}

func getIntArray(data map[string]interface{}, key string) []int {
	if val, ok := data[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]int, 0, len(arr))
			for _, item := range arr {
				switch v := item.(type) {
				case int:
					result = append(result, v)
				case float64:
					result = append(result, int(v))
				}
			}
			return result
		}
	}
	return []int{}
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

// extractGameIDFromURL extracts the game ID from a game URL path
func extractGameIDFromURL(gameURL string) string {
	// gameURL format: "/game/{gameID}"
	if len(gameURL) > 6 && gameURL[:6] == "/game/" {
		return gameURL[6:]
	}
	return ""
}

// GetGameSession returns a game session by gameID (exported for use in other handlers)
func GetGameSession(gameID string) (*GameSession, bool) {
	mu.RLock()
	defer mu.RUnlock()
	session, exists := gameSessions[gameID]
	return session, exists
}

// Fiber WebSocket-specific methods
func (p *Player) readPump() {
	defer func() {
		p.cancel()
		p.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		var msg Message
		err := wsjson.Read(p.ctx, p.Conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		log.Printf("📨 Received message: %s from %s", msg.Type, p.Username)
		handleMessage(p, msg)
	}
}

func (p *Player) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		p.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case msg, ok := <-p.send:
			if !ok {
				p.Conn.Close(websocket.StatusNormalClosure, "channel closed")
				return
			}

			writeCtx, cancel := context.WithTimeout(p.ctx, writeWait)
			err := wsjson.Write(writeCtx, p.Conn, msg)
			cancel()

			if err != nil {
				log.Printf("❌ Error writing to WebSocket: %v", err)
				return
			}

		case <-ticker.C:
			// Send ping
			pingCtx, cancel := context.WithTimeout(p.ctx, writeWait)
			err := p.Conn.Ping(pingCtx)
			cancel()

			if err != nil {
				log.Printf("❌ Ping failed: %v", err)
				return
			}

		case <-p.ctx.Done():
			return
		}
	}
}

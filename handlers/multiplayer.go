// ~/Documents/CODING/ubible/handlers/multiplayer.go
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	mathrand "math/rand"
	"sync"
	"time"
	"ubible/database"
	"ubible/models"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

const (
	// WebSocket timeouts
	writeWait  = 10 * time.Second // Time allowed to write a message
	pingPeriod = 15 * time.Second // Send pings at this interval

	// Send channel buffer size
	sendBufferSize = 256
)

type Player struct {
	ID        string
	Username  string
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
	CurrentQuestion int             `json:"current_question"` // 0-indexed
	PlayersAnswered map[string]bool `json:"players_answered"` // playerID → has answered current Q
	PlayerScores    map[string]int  `json:"player_scores"`    // playerID → current score

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

// HandleFiberWebSocket handles WebSocket connections using Fiber's WebSocket
func HandleFiberWebSocket(conn *websocket.Conn, playerID, username string) {
	if playerID == "" {
		playerID = generateID()
	}
	if username == "" {
		username = "Player" + playerID[:6]
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	player.sendFiberMessage("connected", map[string]interface{}{
		"player_id": playerID,
		"username":  username,
	})

	// Start write pump in separate goroutine
	go player.writeFiberPump()

	// Start read pump (blocking)
	player.readFiberPump()

	// Cleanup when connection closes
	mu.Lock()
	delete(players, conn)
	mu.Unlock()

	if player.Room != "" {
		handleLeaveRoom(player)
	}

	close(player.send)
	log.Printf("Player disconnected: %s (%s)", player.Username, player.ID)
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

// sendMessage is a wrapper that uses the Fiber implementation
func (p *Player) sendMessage(msgType string, payload interface{}) {
	p.sendFiberMessage(msgType, payload)
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
		player.sendMessage("error", map[string]interface{}{"error": "Only host can start the game"})
		return
	}

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

	// Send confirmation
	player.sendMessage("reconnected", map[string]interface{}{
		"game_id":   gameID,
		"room_code": targetRoom.Code,
		"success":   true,
	})
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

	// Parse answer data
	data := parsePayload(payload)
	questionIndex := getInt(data, "questionIndex", -1)
	isCorrect := getBool(data, "correct", false)
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

	log.Printf("📝 Player %s answered Q%d (correct: %v, score: %d) - %d/%d answered",
		player.ID, questionIndex, isCorrect, score, answeredCount, playingCount)

	// If all players have answered, prepare for next question
	if allAnswered {
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

			// Broadcast game over with final scores
			room.mu.RLock()
			finalScores := make([]map[string]interface{}, 0)
			for pid, score := range room.PlayerScores {
				if p, exists := room.Players[pid]; exists {
					finalScores = append(finalScores, map[string]interface{}{
						"player_id": pid,
						"username":  p.Username,
						"score":     score,
					})
				}
			}
			room.mu.RUnlock()

			broadcastToRoom(room, "game_complete", map[string]interface{}{
				"final_scores": finalScores,
			})
		} else {
			log.Printf("➡️  Room %s advancing to Q%d/%d", roomCode, nextQuestion+1, totalQuestions)

			// Broadcast ready for next question
			broadcastToRoom(room, "next_question", map[string]interface{}{
				"question_index": nextQuestion,
				"total":          totalQuestions,
			})
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

	// Shuffle questions using consistent random seed based on game ID
	source := mathrand.NewSource(time.Now().UnixNano())
	rng := mathrand.New(source)
	rng.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})

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
func (p *Player) readFiberPump() {
	defer func() {
		p.cancel()
		p.Conn.Close()
	}()

	for {
		var msg Message
		err := p.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		log.Printf("Received message: %s from %s", msg.Type, p.Username)
		handleMessage(p, msg)
	}
}

func (p *Player) writeFiberPump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		p.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-p.send:
			if !ok {
				p.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			p.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := p.Conn.WriteJSON(msg); err != nil {
				log.Printf("Error writing to WebSocket: %v", err)
				return
			}

		case <-ticker.C:
			p.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := p.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Player) sendFiberMessage(msgType string, payload interface{}) {
	message := Message{
		Type:    msgType,
		Payload: payload,
	}

	select {
	case p.send <- message:
	default:
		log.Printf("Warning: Send buffer full for player %s", p.ID)
	}
}

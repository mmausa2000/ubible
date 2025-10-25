# Database Tracking Integration Guide

## Overview
This document outlines all the integration points where database tracking has been added to the multiplayer handlers.

---

## ✅ Completed Integrations

### 1. Room Creation (handleCreateRoom)
**Location**: handlers/multiplayer.go:637-666
**What's Tracked**:
- Game record created with all metadata
- Host added as first player
- Room creation event logged

**Code Added**:
```go
// Create game record
services.MultiplayerDB.CreateGame(gameID, roomCode, gameURL, hostPlayerID, maxPlayers, questionCount, timeLimit, selectedThemes)

// Add host as player
services.MultiplayerDB.AddPlayer(gameID, playerID, username, userID, isGuest, isHost=true, isPlaying)

// Log event
services.MultiplayerDB.LogEvent(gameID, "room_created", playerID, nil, eventData, 0)
```

---

## 📝 Remaining Integration Points

### 2. Player Join (handleJoinRoom)
**Location**: handlers/multiplayer.go:~680
**Needed**:
```go
// After player joins room
services.MultiplayerDB.AddPlayer(gameID, player.ID, player.Username, player.UserID, player.IsGuest, false, true)
services.MultiplayerDB.LogEvent(gameID, "player_joined", player.ID, nil, map[string]interface{}{
    "username": player.Username,
}, room.MessageSeq)
```

### 3. Player Ready (handlePlayerReady)
**Location**: handlers/multiplayer.go:~750
**Needed**:
```go
// After player.IsReady = true
services.MultiplayerDB.UpdatePlayerReady(room.GameID, player.ID, true)
services.MultiplayerDB.LogEvent(room.GameID, "player_ready", player.ID, nil, nil, room.MessageSeq)
```

### 4. Game Start (startGame)
**Location**: handlers/multiplayer.go:~1470
**Needed**:
```go
// At start of startGame function
services.MultiplayerDB.StartGame(room.GameID)
services.MultiplayerDB.LogEvent(room.GameID, "game_started", "", nil, map[string]interface{}{
    "player_count": len(room.Players),
}, room.MessageSeq)
```

### 5. Answer Submission (handleSubmitAnswer)
**Location**: handlers/multiplayer.go:~900
**Needed**:
```go
// After score calculation
currentQuestion := &room.CurrentQuestion
services.MultiplayerDB.UpdatePlayerScore(room.GameID, player.ID, newScore, correctCount, wrongCount)
services.MultiplayerDB.LogEvent(room.GameID, "answer_submitted", player.ID, currentQuestion, map[string]interface{}{
    "is_correct": isCorrect,
    "score": newScore,
}, room.MessageSeq)
```

### 6. Question Advancement (handleSubmitAnswer - after all answered)
**Location**: handlers/multiplayer.go:~950
**Needed**:
```go
// When advancing to next question
services.MultiplayerDB.UpdateCurrentQuestion(room.GameID, nextQuestion)
services.MultiplayerDB.LogEvent(room.GameID, "question_advanced", "", &nextQuestion, map[string]interface{}{
    "question_index": nextQuestion,
    "total": totalQuestions,
}, room.MessageSeq)
```

### 7. Game Completion (handleGameComplete)
**Location**: handlers/multiplayer.go:~950 (or create new function)
**Needed**:
```go
// When game completes
services.MultiplayerDB.CompleteGame(room.GameID)
services.MultiplayerDB.LogEvent(room.GameID, "game_completed", "", nil, map[string]interface{}{
    "total_players": len(room.Players),
}, room.MessageSeq)
```

### 8. Player Disconnect (handlePlayerDisconnect or in onclose)
**Location**: handlers/multiplayer.go:~1650 (WebSocket close handler)
**Needed**:
```go
// When player disconnects
if player.Room != "" {
    room := rooms[player.Room]
    if room != nil {
        services.MultiplayerDB.RecordPlayerDisconnect(room.GameID, player.ID)
        services.MultiplayerDB.LogEvent(room.GameID, "player_disconnected", player.ID, nil, nil, room.MessageSeq)
    }
}
```

### 9. Player Reconnect (handleReconnect)
**Location**: handlers/multiplayer.go:~800
**Needed**:
```go
// After successful reconnect
services.MultiplayerDB.RecordPlayerReconnect(gameID, player.ID)
services.MultiplayerDB.LogEvent(gameID, "player_reconnected", player.ID, nil, nil, targetRoom.MessageSeq)
```

### 10. Player Leave (handleLeaveRoom)
**Location**: handlers/multiplayer.go:~970
**Needed**:
```go
// When player leaves
if room != nil {
    services.MultiplayerDB.RecordPlayerLeft(room.GameID, player.ID)
    services.MultiplayerDB.LogEvent(room.GameID, "player_left", player.ID, nil, nil, room.MessageSeq)
}
```

---

## Event Types Reference

```go
const (
    EventRoomCreated        = "room_created"
    EventPlayerJoined       = "player_joined"
    EventPlayerLeft         = "player_left"
    EventPlayerReady        = "player_ready"
    EventGameStarted        = "game_started"
    EventAnswerSubmitted    = "answer_submitted"
    EventQuestionAdvanced   = "question_advanced"
    EventGameCompleted      = "game_completed"
    EventPlayerDisconnected = "player_disconnected"
    EventPlayerReconnected  = "player_reconnected"
)
```

---

## Integration Checklist

- [x] Create database models
- [x] Create database migration
- [x] Create database service layer
- [x] Add import in multiplayer.go
- [x] Integrate: Room creation
- [ ] Integrate: Player join
- [ ] Integrate: Player ready
- [ ] Integrate: Game start
- [ ] Integrate: Answer submission
- [ ] Integrate: Question advancement
- [ ] Integrate: Game completion
- [ ] Integrate: Player disconnect
- [ ] Integrate: Player reconnect
- [ ] Integrate: Player leave
- [ ] Create debugger UI
- [ ] Create admin API endpoints
- [ ] Test end-to-end

---

## Next Steps

1. Complete remaining integrations (items 2-10 above)
2. Build debugger UI to display database records
3. Create admin API endpoints for querying game data
4. Test with real multiplayer game
5. Verify all events are logged correctly

---

## Files Modified

**Created**:
- `models/multiplayer.go` - Database models
- `services/multiplayer_db.go` - Database service
- `DB_TRACKING_INTEGRATION_GUIDE.md` - This file

**Modified**:
- `database/migrate.go` - Added multiplayer table migrations and indexes
- `handlers/multiplayer.go` - Added services import and room creation tracking

**Pending**:
- `handlers/multiplayer.go` - Add remaining 9 integration points
- `static/debugger.html` - Add database query UI
- Create admin API handlers

---

## Usage Example

Once fully integrated, you'll be able to:

```sql
-- Get all recent games
SELECT * FROM multiplayer_games ORDER BY created_at DESC LIMIT 10;

-- Get players for a specific game
SELECT * FROM multiplayer_game_players WHERE game_id = (
    SELECT id FROM multiplayer_games WHERE game_id = 'uuid-here'
);

-- Get event timeline for a game
SELECT * FROM multiplayer_game_events
WHERE game_id = (SELECT id FROM multiplayer_games WHERE game_id = 'uuid-here')
ORDER BY sequence_num ASC;

-- Get player statistics
SELECT
    player_id,
    COUNT(*) as games_played,
    AVG(final_score) as avg_score,
    AVG(correct_answers) as avg_correct,
    SUM(CASE WHEN placement = 1 THEN 1 ELSE 0 END) as wins
FROM multiplayer_game_players
GROUP BY player_id;
```

---

**Status**: 1/10 integrations complete
**Next**: Complete player join integration

# Database Tracking Testing Guide

## Quick Start

Follow these steps to test the database tracking system for multiplayer games.

---

## Prerequisites

- ✅ PostgreSQL running
- ✅ Database `ubible` exists
- ✅ Server builds successfully (`go build main.go`)

---

## Step-by-Step Testing

### 1. Check Database Status

```bash
./test_db_tracking.sh
```

**Expected Output**:
- ✅ PostgreSQL is running
- ⚠️ Tables not found (will be created on server start)

---

### 2. Start the Server

```bash
go run main.go
```

**Watch for**:
```
🔄 Running database migrations...
✅ Core migrations completed
✅ Multiplayer tracking migrations completed
✅ Core indexes created successfully
✅ Multiplayer tracking indexes created successfully
✅ All migrations completed successfully
```

---

### 3. Verify Tables Were Created

In another terminal:

```bash
psql -d ubible -c "\dt multiplayer*"
```

**Expected Output**:
```
                     List of relations
 Schema |           Name              | Type  |     Owner
--------+-----------------------------+-------+---------------
 public | multiplayer_game_events     | table | alberickecha
 public | multiplayer_game_players    | table | alberickecha
 public | multiplayer_games           | table | alberickecha
```

---

### 4. Create a Multiplayer Room

**Option A: Using Debugger (Recommended)**
1. Open browser: `http://localhost:3000/debugger`
2. Click **"🔌 Connect"**
3. Click **"Create Room"**
4. Note the room code displayed

**Option B: Using Browser Console**
```javascript
// On http://localhost:3000
const ws = new WebSocket('ws://localhost:4000/ws?player_id=test123&username=TestUser');
ws.onopen = () => {
    ws.send(JSON.stringify({
        type: 'create_room',
        payload: {
            max_players: 10,
            question_count: 10,
            time_limit: 10,
            theme_ids: [1]
        }
    }));
};
ws.onmessage = (e) => console.log('Received:', e.data);
```

---

### 5. Verify Database Record Was Created

```bash
# Quick check
psql -d ubible -c "SELECT game_id, room_code, status, created_at FROM multiplayer_games ORDER BY created_at DESC LIMIT 1;"
```

**Expected Output**:
```
              game_id               | room_code |  status  |         created_at
------------------------------------+-----------+----------+----------------------------
 a1b2c3d4-e5f6-7890-abcd-ef1234567890| ABC123    | waiting  | 2025-10-24 23:30:15.123456
```

---

### 6. Verify Player Record

```bash
psql -d ubible -c "SELECT player_id, username, is_host, is_guest, joined_at FROM multiplayer_game_players ORDER BY joined_at DESC LIMIT 1;"
```

**Expected Output**:
```
  player_id  |  username  | is_host | is_guest |         joined_at
-------------+------------+---------+----------+----------------------------
 player_xxx  | TestPlayer | t       | f        | 2025-10-24 23:30:15.234567
```

---

### 7. Verify Event Was Logged

```bash
psql -d ubible -c "SELECT event_type, player_id, timestamp FROM multiplayer_game_events ORDER BY timestamp DESC LIMIT 1;"
```

**Expected Output**:
```
  event_type   |  player_id  |         timestamp
---------------+-------------+----------------------------
 room_created  | player_xxx  | 2025-10-24 23:30:15.345678
```

---

### 8. Run Comprehensive Query

```bash
psql -d ubible -f query_multiplayer_db.sql
```

**This will show**:
- Recent games (last 5)
- Players (last 10)
- Recent events (last 15)
- Statistics summary
- Active games

---

## Verification Checklist

After creating a room, verify:

- [ ] Game record exists in `multiplayer_games` table
- [ ] Game has correct `game_id`, `room_code`, `status='waiting'`
- [ ] Player record exists in `multiplayer_game_players` table
- [ ] Player is marked as `is_host=true`
- [ ] Event record exists in `multiplayer_game_events` table
- [ ] Event type is `room_created`
- [ ] All timestamps are recent (within last few minutes)

---

## Troubleshooting

### Tables Not Created

**Check server logs for migration errors**:
```bash
# Look for:
❌ Failed to run multiplayer tracking migrations
```

**Solution**: Check PostgreSQL connection and permissions

### No Database Records After Creating Room

**Check server logs for**:
```bash
# Look for:
📊 DB: Created game record: ID=xxx, Room=ABC123
📊 DB: Player xxx joined game xxx
```

**If you see**:
```bash
⚠️ Failed to create game in database: [error]
```

**Common issues**:
1. Missing import in multiplayer.go
2. Database connection issue
3. Migration didn't run

**Verify import**:
```go
// In handlers/multiplayer.go
import (
    // ...
    "ubible/services"  // This must be present
)
```

### Query Returns No Results

**Check if tables exist**:
```bash
psql -d ubible -c "\dt"
```

**Check if server is running**:
```bash
ps aux | grep "go run main.go"
```

---

## Advanced Testing

### Test Full Game Flow (Manual)

1. Create room via debugger
2. Join with second browser
3. Both players ready
4. Check database for:
   - Player count = 2
   - Events for player_joined, player_ready

### Monitor Database in Real-Time

```bash
# In one terminal, watch for new records
watch -n 1 'psql -d ubible -c "SELECT COUNT(*) as games, (SELECT COUNT(*) FROM multiplayer_game_players) as players, (SELECT COUNT(*) FROM multiplayer_game_events) as events FROM multiplayer_games;"'
```

### Clear Test Data

```bash
# WARNING: This deletes all multiplayer game data
psql -d ubible -c "
TRUNCATE TABLE multiplayer_game_events CASCADE;
TRUNCATE TABLE multiplayer_game_players CASCADE;
TRUNCATE TABLE multiplayer_games CASCADE;
"
```

---

## What's Tracked Currently

✅ **Implemented**:
- Room creation → `multiplayer_games` record
- Host joins → `multiplayer_game_players` record
- Room created event → `multiplayer_game_events` record

⏳ **Not Yet Implemented** (see DB_TRACKING_INTEGRATION_GUIDE.md):
- Player join (non-host)
- Player ready
- Game start
- Answer submission
- Question advancement
- Game completion
- Player disconnect/reconnect
- Player leave

---

## Success Criteria

Your database tracking is working if:

1. ✅ Tables created on server start
2. ✅ Creating a room inserts record in `multiplayer_games`
3. ✅ Host appears in `multiplayer_game_players`
4. ✅ Event logged in `multiplayer_game_events`
5. ✅ Queries return correct data
6. ✅ No errors in server logs

---

## Next Steps

Once basic tracking is verified:

1. Complete remaining integrations (see DB_TRACKING_INTEGRATION_GUIDE.md)
2. Build debugger UI to display database records
3. Create admin API endpoints
4. Add automated tests

---

## Quick Reference Commands

```bash
# Check database status
./test_db_tracking.sh

# View all data
psql -d ubible -f query_multiplayer_db.sql

# Count records
psql -d ubible -c "SELECT (SELECT COUNT(*) FROM multiplayer_games) as games, (SELECT COUNT(*) FROM multiplayer_game_players) as players, (SELECT COUNT(*) FROM multiplayer_game_events) as events;"

# View latest game
psql -d ubible -c "SELECT * FROM multiplayer_games ORDER BY created_at DESC LIMIT 1;"

# View latest player
psql -d ubible -c "SELECT * FROM multiplayer_game_players ORDER BY joined_at DESC LIMIT 1;"

# View latest event
psql -d ubible -c "SELECT * FROM multiplayer_game_events ORDER BY timestamp DESC LIMIT 1;"
```

---

**Ready to test!** Start the server and create a room to see database tracking in action. 🚀

#!/bin/bash

# Test Database Tracking for Multiplayer Games
# This script verifies that database tracking is working correctly

echo "🧪 Testing Multiplayer Database Tracking"
echo "========================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if PostgreSQL is running
echo -e "${BLUE}📊 Step 1: Checking PostgreSQL connection...${NC}"
psql -d ubible -c "SELECT version();" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ PostgreSQL is running${NC}"
else
    echo -e "${RED}❌ PostgreSQL is not running or database 'ubible' doesn't exist${NC}"
    echo "   Please start PostgreSQL and create the database:"
    echo "   createdb ubible"
    exit 1
fi
echo ""

# Check if multiplayer tables exist
echo -e "${BLUE}📊 Step 2: Checking if multiplayer tracking tables exist...${NC}"
TABLES=$(psql -d ubible -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('multiplayer_games', 'multiplayer_game_players', 'multiplayer_game_events');")
if [ "$TABLES" -eq 3 ]; then
    echo -e "${GREEN}✅ All 3 multiplayer tables exist${NC}"
else
    echo -e "${YELLOW}⚠️  Tables not found (expected 3, found $TABLES)${NC}"
    echo "   The tables will be created when you start the server"
fi
echo ""

# Show table structures
echo -e "${BLUE}📊 Step 3: Table structures:${NC}"
echo ""
echo -e "${YELLOW}Table: multiplayer_games${NC}"
psql -d ubible -c "\d multiplayer_games" 2>/dev/null || echo "   (Table will be created on server start)"
echo ""

echo -e "${YELLOW}Table: multiplayer_game_players${NC}"
psql -d ubible -c "\d multiplayer_game_players" 2>/dev/null || echo "   (Table will be created on server start)"
echo ""

echo -e "${YELLOW}Table: multiplayer_game_events${NC}"
psql -d ubible -c "\d multiplayer_game_events" 2>/dev/null || echo "   (Table will be created on server start)"
echo ""

# Check for existing data
echo -e "${BLUE}📊 Step 4: Checking for existing game data...${NC}"
GAME_COUNT=$(psql -d ubible -t -c "SELECT COUNT(*) FROM multiplayer_games;" 2>/dev/null || echo "0")
echo "   Total games in database: $GAME_COUNT"

if [ "$GAME_COUNT" -gt 0 ]; then
    echo ""
    echo -e "${GREEN}📋 Recent games:${NC}"
    psql -d ubible -c "SELECT game_id, room_code, status, created_at FROM multiplayer_games ORDER BY created_at DESC LIMIT 5;" 2>/dev/null
fi
echo ""

# Provide next steps
echo -e "${BLUE}📊 Step 5: Next steps to test database tracking:${NC}"
echo ""
echo "1. Start the server:"
echo -e "   ${YELLOW}go run main.go${NC}"
echo ""
echo "2. Open the debugger:"
echo -e "   ${YELLOW}http://localhost:3000/debugger${NC}"
echo ""
echo "3. Click 'Connect' and then 'Create Room'"
echo ""
echo "4. Check the database:"
echo -e "   ${YELLOW}psql -d ubible -c \"SELECT * FROM multiplayer_games ORDER BY created_at DESC LIMIT 1;\"${NC}"
echo ""
echo "5. View all players in the game:"
echo -e "   ${YELLOW}psql -d ubible -c \"SELECT * FROM multiplayer_game_players ORDER BY joined_at DESC LIMIT 5;\"${NC}"
echo ""
echo "6. View game events:"
echo -e "   ${YELLOW}psql -d ubible -c \"SELECT event_type, player_id, timestamp FROM multiplayer_game_events ORDER BY timestamp DESC LIMIT 10;\"${NC}"
echo ""
echo -e "${GREEN}✅ Database tracking test script completed!${NC}"
echo ""
echo "Once you create a room, you can verify it's tracked with:"
echo -e "${BLUE}./test_db_tracking.sh${NC} (run this script again)"

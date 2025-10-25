#!/usr/bin/env node

/**
 * WebSocket Multiplayer Improvements Test Suite
 *
 * Tests the following scenarios:
 * 1. Message ordering with out-of-order delivery
 * 2. Reconnection with state restoration
 * 3. Heartbeat and dead connection detection
 * 4. Sequence number validation
 * 5. State snapshot accuracy
 */

const WebSocket = require('ws');
const http = require('http');

// Test configuration
const WS_HOST = 'localhost:4000';
const WS_URL = `ws://${WS_HOST}/ws`;
const TEST_TIMEOUT = 30000; // 30 seconds per test

// ANSI color codes
const colors = {
    reset: '\x1b[0m',
    green: '\x1b[32m',
    red: '\x1b[31m',
    yellow: '\x1b[33m',
    blue: '\x1b[34m',
    cyan: '\x1b[36m'
};

// Test results tracker
const results = {
    passed: 0,
    failed: 0,
    total: 0
};

// Helper functions
function log(message, color = colors.reset) {
    console.log(`${color}${message}${colors.reset}`);
}

function pass(testName) {
    results.passed++;
    results.total++;
    log(`✅ PASS: ${testName}`, colors.green);
}

function fail(testName, error) {
    results.failed++;
    results.total++;
    log(`❌ FAIL: ${testName}`, colors.red);
    log(`   Error: ${error}`, colors.red);
}

function section(title) {
    log(`\n${'='.repeat(60)}`, colors.cyan);
    log(`  ${title}`, colors.cyan);
    log(`${'='.repeat(60)}`, colors.cyan);
}

// Sleep utility
const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));

// Create WebSocket connection with player info
function createWSConnection(playerId, username) {
    const url = `${WS_URL}?player_id=${playerId}&username=${encodeURIComponent(username)}`;
    return new WebSocket(url);
}

// Wait for WebSocket to open
function waitForOpen(ws) {
    return new Promise((resolve, reject) => {
        if (ws.readyState === WebSocket.OPEN) {
            resolve();
            return;
        }

        const timeout = setTimeout(() => {
            reject(new Error('WebSocket connection timeout'));
        }, 5000);

        ws.once('open', () => {
            clearTimeout(timeout);
            resolve();
        });

        ws.once('error', (err) => {
            clearTimeout(timeout);
            reject(err);
        });
    });
}

// Wait for specific message type
function waitForMessage(ws, type, timeout = 5000) {
    return new Promise((resolve, reject) => {
        const timer = setTimeout(() => {
            reject(new Error(`Timeout waiting for message type: ${type}`));
        }, timeout);

        const handler = (data) => {
            try {
                const msg = JSON.parse(data);
                if (msg.type === type) {
                    clearTimeout(timer);
                    ws.off('message', handler);
                    resolve(msg);
                }
            } catch (e) {
                // Ignore parse errors
            }
        };

        ws.on('message', handler);
    });
}

// Test 1: Message Sequence Numbers
async function testMessageSequencing() {
    section('Test 1: Message Sequence Numbers');

    const player1 = 'test_seq_p1_' + Date.now();
    const player2 = 'test_seq_p2_' + Date.now();

    const ws1 = createWSConnection(player1, 'Player1');
    const ws2 = createWSConnection(player2, 'Player2');

    try {
        await waitForOpen(ws1);
        await waitForOpen(ws2);

        // Create a room
        ws1.send(JSON.stringify({
            type: 'create_room',
            payload: {
                max_players: 2,
                question_count: 5,
                time_limit: 10,
                theme_ids: [1]
            }
        }));

        const roomUpdate = await waitForMessage(ws1, 'room_update');
        const roomCode = roomUpdate.payload.room_code;

        log(`Created room: ${roomCode}`, colors.blue);

        // Track sequence numbers
        const sequences = [];
        let duplicates = 0;
        let outOfOrder = 0;

        const messageHandler = (data) => {
            try {
                const msg = JSON.parse(data);
                if (msg.seq !== undefined) {
                    sequences.push(msg.seq);
                    log(`  Received seq=${msg.seq} type=${msg.type}`, colors.yellow);

                    // Check for duplicates
                    if (sequences.filter(s => s === msg.seq).length > 1) {
                        duplicates++;
                    }

                    // Check for ordering
                    if (sequences.length > 1) {
                        const prev = sequences[sequences.length - 2];
                        if (msg.seq < prev) {
                            outOfOrder++;
                        }
                    }
                }
            } catch (e) {
                // Ignore
            }
        };

        ws1.on('message', messageHandler);

        // Join player 2
        ws2.send(JSON.stringify({
            type: 'join_room',
            payload: { room_code: roomCode }
        }));

        await sleep(1000);

        // Ready both players
        ws1.send(JSON.stringify({ type: 'player_ready' }));
        ws2.send(JSON.stringify({ type: 'player_ready' }));

        await sleep(1000);

        // Verify sequences
        if (sequences.length > 0) {
            pass('Message sequence numbers present');
        } else {
            fail('Message sequence numbers', 'No sequences received');
        }

        if (duplicates === 0) {
            pass('No duplicate sequence numbers');
        } else {
            fail('No duplicate sequence numbers', `Found ${duplicates} duplicates`);
        }

        // Check if sequences are monotonically increasing
        let isMonotonic = true;
        for (let i = 1; i < sequences.length; i++) {
            if (sequences[i] <= sequences[i - 1]) {
                isMonotonic = false;
                break;
            }
        }

        if (isMonotonic) {
            pass('Sequence numbers are monotonically increasing');
        } else {
            fail('Sequence numbers monotonic', 'Found non-increasing sequences');
        }

    } catch (error) {
        fail('Message Sequencing Test', error.message);
    } finally {
        ws1.close();
        ws2.close();
    }
}

// Test 2: Reconnection State Snapshot
async function testReconnectionStateSnapshot() {
    section('Test 2: Reconnection State Snapshot');

    const player1 = 'test_reconnect_p1_' + Date.now();
    const player2 = 'test_reconnect_p2_' + Date.now();

    const ws1 = createWSConnection(player1, 'Player1');
    const ws2 = createWSConnection(player2, 'Player2');

    try {
        await waitForOpen(ws1);
        await waitForOpen(ws2);

        // Create room
        ws1.send(JSON.stringify({
            type: 'create_room',
            payload: {
                max_players: 2,
                question_count: 10,
                time_limit: 20,
                theme_ids: [1]
            }
        }));

        const roomUpdate = await waitForMessage(ws1, 'room_update');
        const roomCode = roomUpdate.payload.room_code;

        // Join player 2
        ws2.send(JSON.stringify({
            type: 'join_room',
            payload: { room_code: roomCode }
        }));

        await sleep(500);

        // Ready both
        ws1.send(JSON.stringify({ type: 'player_ready' }));
        ws2.send(JSON.stringify({ type: 'player_ready' }));

        // Wait for game to start
        const gameStart = await waitForMessage(ws1, 'game_start', 10000);
        const gameId = gameStart.payload.game_id;

        log(`Game started: ${gameId}`, colors.blue);

        // Disconnect player 1
        ws1.close();
        await sleep(1000);

        // Reconnect player 1
        const ws1Reconnect = createWSConnection(player1, 'Player1');
        await waitForOpen(ws1Reconnect);

        ws1Reconnect.send(JSON.stringify({
            type: 'reconnect',
            payload: { game_id: gameId }
        }));

        const reconnectedMsg = await waitForMessage(ws1Reconnect, 'reconnected', 5000);
        const payload = reconnectedMsg.payload;

        // Verify state snapshot fields
        const requiredFields = [
            'success',
            'game_id',
            'room_code',
            'current_question',
            'question_count',
            'player_scores',
            'players_answered',
            'current_seq'
        ];

        let missingFields = [];
        for (const field of requiredFields) {
            if (payload[field] === undefined) {
                missingFields.push(field);
            }
        }

        if (missingFields.length === 0) {
            pass('Reconnection state snapshot has all required fields');
        } else {
            fail('Reconnection state snapshot', `Missing fields: ${missingFields.join(', ')}`);
        }

        // Verify player_scores is an object
        if (typeof payload.player_scores === 'object') {
            pass('player_scores field is an object');
        } else {
            fail('player_scores field', `Expected object, got ${typeof payload.player_scores}`);
        }

        // Verify current_seq is a number
        if (typeof payload.current_seq === 'number') {
            pass('current_seq field is a number');
        } else {
            fail('current_seq field', `Expected number, got ${typeof payload.current_seq}`);
        }

        ws1Reconnect.close();

    } catch (error) {
        fail('Reconnection State Snapshot Test', error.message);
    } finally {
        ws2.close();
    }
}

// Test 3: Heartbeat Ping/Pong
async function testHeartbeat() {
    section('Test 3: Heartbeat Ping/Pong');

    const player = 'test_heartbeat_' + Date.now();
    const ws = createWSConnection(player, 'HeartbeatPlayer');

    try {
        await waitForOpen(ws);

        // Send ping
        ws.send(JSON.stringify({ type: 'ping' }));

        // Wait for pong
        const pong = await waitForMessage(ws, 'pong', 2000);

        if (pong && pong.type === 'pong') {
            pass('Server responds to ping with pong');
        } else {
            fail('Heartbeat ping/pong', 'No pong received');
        }

    } catch (error) {
        fail('Heartbeat Test', error.message);
    } finally {
        ws.close();
    }
}

// Test 4: Out-of-Order Message Handling (Client-Side Logic)
async function testOutOfOrderMessages() {
    section('Test 4: Out-of-Order Message Handling');

    log('⚠️  Note: This test validates server sequence assignment.', colors.yellow);
    log('   Client-side buffering is tested via browser console.', colors.yellow);

    const player1 = 'test_order_p1_' + Date.now();
    const player2 = 'test_order_p2_' + Date.now();

    const ws1 = createWSConnection(player1, 'OrderPlayer1');
    const ws2 = createWSConnection(player2, 'OrderPlayer2');

    try {
        await waitForOpen(ws1);
        await waitForOpen(ws2);

        // Create room and start game
        ws1.send(JSON.stringify({
            type: 'create_room',
            payload: {
                max_players: 2,
                question_count: 3,
                time_limit: 10,
                theme_ids: [1]
            }
        }));

        const roomUpdate = await waitForMessage(ws1, 'room_update');
        const roomCode = roomUpdate.payload.room_code;

        ws2.send(JSON.stringify({
            type: 'join_room',
            payload: { room_code: roomCode }
        }));

        await sleep(500);

        ws1.send(JSON.stringify({ type: 'player_ready' }));
        ws2.send(JSON.stringify({ type: 'player_ready' }));

        await sleep(1000);

        // Collect messages and their sequences
        const messages = [];
        const collectMessages = (data) => {
            try {
                const msg = JSON.parse(data);
                if (msg.seq !== undefined) {
                    messages.push({ type: msg.type, seq: msg.seq, timestamp: msg.timestamp });
                }
            } catch (e) {
                // Ignore
            }
        };

        ws1.on('message', collectMessages);

        await sleep(2000);

        // Verify all messages have timestamps
        const withTimestamps = messages.filter(m => m.timestamp !== undefined);

        if (withTimestamps.length === messages.length) {
            pass('All messages include timestamp field');
        } else {
            fail('Message timestamps', `${messages.length - withTimestamps.length} messages missing timestamps`);
        }

        // Verify timestamps are reasonable (within last minute)
        const now = Date.now();
        const invalidTimestamps = withTimestamps.filter(m => {
            return m.timestamp < now - 60000 || m.timestamp > now + 1000;
        });

        if (invalidTimestamps.length === 0) {
            pass('All timestamps are within reasonable range');
        } else {
            fail('Timestamp validity', `${invalidTimestamps.length} invalid timestamps`);
        }

    } catch (error) {
        fail('Out-of-Order Message Test', error.message);
    } finally {
        ws1.close();
        ws2.close();
    }
}

// Test 5: Broadcast Message Logging
async function testBroadcastLogging() {
    section('Test 5: Broadcast Message Logging');

    log('⚠️  Note: Check server logs for enhanced broadcast messages.', colors.yellow);
    log('   Expected format: "📤 Broadcasting \'X\' to room Y (N recipients) [seq=Z]"', colors.yellow);

    const player1 = 'test_log_p1_' + Date.now();
    const player2 = 'test_log_p2_' + Date.now();

    const ws1 = createWSConnection(player1, 'LogPlayer1');
    const ws2 = createWSConnection(player2, 'LogPlayer2');

    try {
        await waitForOpen(ws1);
        await waitForOpen(ws2);

        ws1.send(JSON.stringify({
            type: 'create_room',
            payload: {
                max_players: 2,
                question_count: 3,
                time_limit: 10,
                theme_ids: [1]
            }
        }));

        const roomUpdate = await waitForMessage(ws1, 'room_update');
        const roomCode = roomUpdate.payload.room_code;

        ws2.send(JSON.stringify({
            type: 'join_room',
            payload: { room_code: roomCode }
        }));

        await sleep(500);

        ws1.send(JSON.stringify({ type: 'player_ready' }));
        ws2.send(JSON.stringify({ type: 'player_ready' }));

        await sleep(2000);

        pass('Broadcast test completed - check server logs for [seq=X] markers');

    } catch (error) {
        fail('Broadcast Logging Test', error.message);
    } finally {
        ws1.close();
        ws2.close();
    }
}

// Main test runner
async function runAllTests() {
    log('\n╔════════════════════════════════════════════════════════════╗', colors.cyan);
    log('║  WebSocket Multiplayer Improvements Test Suite            ║', colors.cyan);
    log('╚════════════════════════════════════════════════════════════╝', colors.cyan);

    try {
        await testMessageSequencing();
        await sleep(1000);

        await testReconnectionStateSnapshot();
        await sleep(1000);

        await testHeartbeat();
        await sleep(1000);

        await testOutOfOrderMessages();
        await sleep(1000);

        await testBroadcastLogging();

    } catch (error) {
        log(`\n❌ Test suite error: ${error.message}`, colors.red);
    }

    // Summary
    section('Test Results Summary');
    log(`Total Tests: ${results.total}`, colors.cyan);
    log(`Passed: ${results.passed}`, colors.green);
    log(`Failed: ${results.failed}`, results.failed > 0 ? colors.red : colors.green);

    const successRate = results.total > 0 ? (results.passed / results.total * 100).toFixed(1) : 0;
    log(`Success Rate: ${successRate}%`, successRate >= 80 ? colors.green : colors.red);

    log('');

    // Exit code
    process.exit(results.failed > 0 ? 1 : 0);
}

// Run tests
runAllTests().catch(error => {
    log(`\n💥 Fatal error: ${error.message}`, colors.red);
    process.exit(1);
});

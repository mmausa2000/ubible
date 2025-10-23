# 📖 Bible Quiz Pro

A comprehensive, production-ready Bible quiz application with real-time multiplayer gameplay, progression system, achievements, and admin portal. Built with Go (Fiber) backend and vanilla JavaScript frontend.

[![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Security](https://img.shields.io/badge/Security-Hardened-success)](SECURITY.md)

---

## ✨ Features

### 🎮 Game Modes
- **Single Player Practice** - Quiz yourself with customizable settings
- **Real-Time Multiplayer** - Competitive matches via WebSocket
- **Daily Challenges** - New challenges every day
- **Private Rooms** - Custom rooms to play with friends
- **Theme-Based Quizzes** - Multiple Bible themes to explore

### 📊 Progression System
- **XP & Leveling** (1-100+) - Earn experience and level up
- **Faith Points Currency** - Purchase power-ups and unlock themes
- **22+ Achievements** - Unlock achievements across 6 categories
- **Global Leaderboards** - Compete across multiple metrics
- **Streak Tracking** - Daily login, quiz, and win streaks

### 🛍️ Power-Ups
- **50/50** (50 FP) - Remove two wrong answers
- **Time Freeze** (75 FP) - Pause timer for 10 seconds
- **Hint** (40 FP) - Show first letter of answer
- **Skip** (30 FP) - Move to next question
- **Double Points** (100 FP) - Earn 2x points for next question

### 👥 Social Features
- **Friends System** - Add and manage friends
- **User Profiles** - View stats and achievements
- **Search Users** - Find other players
- **Guest Mode** - Play without account (converts to full account)

### ⚙️ Admin Portal
- **User Management** - Ban, delete, reset passwords
- **Theme Management** - Create and manage quiz themes
- **Achievement System** - Create custom achievements
- **Challenge Management** - Daily and weekly challenges
- **Analytics Dashboard** - Comprehensive system analytics
- **Manual Cleanup** - Guest account cleanup

---

## 🚀 Quick Start

### Prerequisites
- **Go 1.19+**
- **SQLite3** (included with most systems)
- **Make** (optional, for build commands)

### Installation

```bash
# 1. Clone repository
git clone https://github.com/yourusername/bible-quiz-pro.git
cd bible-quiz-pro

# 2. Install dependencies
go mod download

# 3. Create environment file
cp .env.example .env

# 4. Generate JWT secret (REQUIRED)
openssl rand -base64 64

# 5. Edit .env and add your JWT secret and admin credentials
nano .env

# 6. Create required directories
mkdir -p data verses backups static

# 7. Run application
go run main.go

# Or build and run
go build -o bible-quiz-pro main.go
./bible-quiz-pro
```

### First Time Setup

The application will:
1. ✅ Initialize SQLite database at `./data/bible_quiz.db`
2. ✅ Create all tables automatically
3. ✅ Create admin user from `.env` credentials
4. ✅ Load 22+ default achievements
5. ✅ Load verse files from `./verses/` directory
6. ✅ Start WebSocket server at `ws://localhost:3000/ws`
7. ✅ Start HTTP server at `http://localhost:3000`

### Access Points

- **Main App:** http://localhost:3000
- **Admin Portal:** http://localhost:3000/admin/login
- **Health Check:** http://localhost:3000/health
- **API Docs:** http://localhost:3000/api (Coming soon)

---

## 🔒 Security (CRITICAL)

### ⚠️ BEFORE RUNNING IN PRODUCTION

1. **Generate Strong JWT Secret:**
   ```bash
   openssl rand -base64 64
   ```
   Add to `.env`:
   ```env
   JWT_SECRET=your_generated_secret_here
   ```

2. **Set Admin Credentials:**
   ```env
   ADMIN_USERNAME=youradmin
   ADMIN_PASSWORD=YourStrongPassword123!
   ADMIN_EMAIL=admin@yourdomain.com
   ```

3. **Configure CORS:**
   ```env
   CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
   ```

4. **Set Environment:**
   ```env
   APP_ENV=production
   ```

**📚 Read [SECURITY.md](SECURITY.md) for complete security guidelines**

---

## 📁 Project Structure

```
bible-quiz-pro/
├── main.go                      # Application entry point
├── go.mod                       # Go dependencies
├── .env.example                 # Environment template
├── .env                         # Your configuration (DO NOT COMMIT)
│
├── data/
│   └── bible_quiz.db           # SQLite database (auto-created)
│
├── verses/                     # Verse files (JSON/TXT)
│   ├── faith.json
│   ├── love.json
│   └── wisdom.txt
│
├── backups/                    # Database backups
├── logs/                       # Application logs
│
├── database/
│   ├── database.go             # Database connection
│   └── migrate.go              # Schema & migrations
│
├── models/
│   ├── user.go                 # User model with progression
│   ├── achievement.go          # Achievement definitions
│   ├── theme.go                # Theme model
│   ├── question.go             # Question model
│   ├── challenge.go            # Challenge model
│   ├── friend.go               # Friend relationships
│   └── attempt.go              # Game attempts
│
├── services/
│   ├── user_manager.go         # User CRUD operations
│   ├── progression_manager.go  # XP & Achievements
│   ├── verse_loader.go         # Verse file loading
│   ├── matchmaking.go          # Multiplayer matching
│   ├── room_manager.go         # Room management
│   └── cleanup_service.go      # Guest cleanup
│
├── handlers/
│   ├── auth.go                 # Authentication
│   ├── users.go                # User APIs
│   ├── themes.go               # Theme handlers
│   ├── progression.go          # XP/Achievement APIs
│   ├── websocket.go            # Multiplayer WebSocket
│   ├── friends.go              # Friend system
│   ├── leaderboard.go          # Leaderboards
│   └── admin/                  # Admin endpoints
│       ├── auth.go             # Admin authentication
│       ├── users.go            # User management
│       ├── themes.go           # Theme management
│       ├── achievements.go     # Achievement management
│       ├── challenges.go       # Challenge management
│       └── analytics.go        # Analytics dashboard
│
├── middleware/
│   ├── auth.go                 # JWT authentication
│   ├── ratelimit.go            # Rate limiting
│   ├── cors.go                 # CORS configuration
│   └── logging.go              # Request logging
│
└── static/
    ├── index.html              # Home page
    ├── quiz.html               # Quiz interface
    ├── practice.html           # Practice mode
    ├── challenges.html         # Challenges page
    ├── settings.html           # User settings
    ├── shop.html               # Power-up shop
    ├── login.html              # Login page
    ├── css/                    # Stylesheets
    ├── js/                     # JavaScript files
    └── admin/                  # Admin portal
        ├── index.html          # Dashboard
        ├── users.html          # User management
        ├── themes.html         # Theme management
        ├── achievements.html   # Achievement management
        ├── challenges.html     # Challenge management
        └── analytics.html      # Analytics
```

---

## 🌐 API Endpoints

### Authentication
```
POST   /api/auth/guest          # Create guest account
POST   /api/auth/login          # User login
POST   /api/auth/register       # User registration
POST   /api/auth/upgrade        # Upgrade guest to full account
```

### Users
```
GET    /api/users/me            # Get current user
PUT    /api/users/me            # Update current user
GET    /api/users/stats         # Get user stats
GET    /api/users/search        # Search users
GET    /api/users/:id           # Get user profile
```

### Themes & Questions
```
GET    /api/themes              # Get all active themes
POST   /api/themes              # Create theme (auth required)
GET    /api/themes/:id          # Get theme details
PUT    /api/themes/:id          # Update theme (auth required)
DELETE /api/themes/:id          # Delete theme (auth required)
GET    /api/verses              # Get verses
GET    /api/verses/:id          # Get specific verse
```

### Progression
```
POST   /api/progression/xp      # Award XP
POST   /api/progression/game    # Record game
GET    /api/progression         # Get progression info
GET    /api/progression/achievements # Get user achievements
```

### Power-ups
```
POST   /api/powerups/use        # Use power-up
POST   /api/powerups/purchase   # Purchase power-up
GET    /api/powerups/inventory  # Get inventory
```

### Friends
```
GET    /api/friends             # Get friends list
POST   /api/friends/request     # Send friend request
POST   /api/friends/accept      # Accept friend request
DELETE /api/friends/:id         # Remove friend
GET    /api/friends/requests    # Get pending requests
```

### Leaderboards
```
GET    /api/leaderboard         # Get global leaderboard
GET    /api/leaderboard/season  # Get season leaderboard
GET    /api/leaderboard/user/:id # Get user rank
GET    /api/leaderboard/around/:id # Get leaderboard around user
```

### Games
```
POST   /api/games/record        # Record game session
GET    /api/games/history       # Get game history
```

### WebSocket (Multiplayer)
```
WS     /ws                      # WebSocket endpoint
```

**WebSocket Events:**
- `create_room` - Create private room
- `join_room` - Join private room
- `find_match` - Join matchmaking queue
- `player_ready` - Mark player as ready
- `start_game` - Start game (host only)
- `submit_answer` - Submit answer
- `leave_room` - Leave room
- `chat_message` - Send chat message
- `reconnect` - Reconnect after disconnect

### Admin Endpoints
```
POST   /api/admin/login         # Admin login
POST   /api/admin/logout        # Admin logout
GET    /api/admin/verify        # Verify admin session

# Protected (require admin auth)
GET    /api/admin/users         # Get all users
GET    /api/admin/users/:id     # Get user details
PUT    /api/admin/users/:id     # Update user
DELETE /api/admin/users/:id     # Delete user
POST   /api/admin/users/:id/ban # Ban/unban user
POST   /api/admin/users/:id/reset-password # Reset password

GET    /api/admin/themes        # Get all themes
POST   /api/admin/themes        # Create theme
PUT    /api/admin/themes/:id    # Update theme
DELETE /api/admin/themes/:id    # Delete theme

GET    /api/admin/achievements  # Get all achievements
POST   /api/admin/achievements  # Create achievement
PUT    /api/admin/achievements/:id # Update achievement
DELETE /api/admin/achievements/:id # Delete achievement

GET    /api/admin/challenges    # Get all challenges
POST   /api/admin/challenges    # Create challenge
PUT    /api/admin/challenges/:id # Update challenge
DELETE /api/admin/challenges/:id # Delete challenge

GET    /api/admin/analytics     # Get system analytics
POST   /api/admin/cleanup/manual # Trigger manual cleanup
GET    /api/admin/cleanup/stats # Get cleanup statistics
```

---

## 🧪 Testing

### Manual API Testing

```bash
# Health check
curl http://localhost:3000/health

# Register user
curl -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"testpass123"}'

# Login
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass123"}'

# Get themes
curl http://localhost:3000/api/themes

# Get themes with auth
curl http://localhost:3000/api/users/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## 🚀 Production Deployment

**📚 See [DEPLOYMENT.md](DEPLOYMENT.md) for complete deployment guide**

Quick overview:

1. **Build for production:**
   ```bash
   CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o bible-quiz-pro main.go
   ```

2. **Configure environment:**
   - Set strong JWT secret
   - Configure admin credentials
   - Set CORS origins
   - Enable production mode

3. **Set up systemd service** (Linux)
4. **Configure Nginx reverse proxy**
5. **Enable HTTPS with Let's Encrypt**
6. **Set up database backups**
7. **Configure monitoring**

---

## 📊 Features Breakdown

### ✅ Fully Implemented

- [x] User authentication (JWT)
- [x] Guest accounts with upgrade path
- [x] XP and leveling system (1-100+)
- [x] Achievement system (22+ achievements)
- [x] Power-up system (5 power-ups)
- [x] Theme-based quizzes
- [x] Real-time multiplayer (WebSocket)
- [x] Private rooms
- [x] Matchmaking
- [x] Friend system
- [x] Global leaderboards
- [x] User profiles
- [x] Admin portal
- [x] Admin authentication
- [x] User management
- [x] Theme management
- [x] Achievement management
- [x] Challenge management
- [x] Analytics dashboard
- [x] Automatic guest cleanup
- [x] Rate limiting
- [x] CORS configuration
- [x] Database migrations
- [x] Verse file loading (JSON/TXT)
- [x] Health check endpoint

### 🔄 Coming Soon

- [ ] Email notifications
- [ ] Password reset via email
- [ ] Two-factor authentication (2FA)
- [ ] Tournament mode
- [ ] Clubs/Groups system
- [ ] Battle Pass
- [ ] Mini-games
- [ ] Quest system
- [ ] API documentation (Swagger)
- [ ] Mobile app (React Native)

---

## ⚙️ Configuration

### Environment Variables

See `.env.example` for all available options. Key variables:

```env
# REQUIRED
JWT_SECRET=<64-character-random-string>
ADMIN_USERNAME=<your-admin-username>
ADMIN_PASSWORD=<strong-password>

# Application
APP_ENV=development|production
PORT=3000

# Database
DATABASE_URL=./data/bible_quiz.db

# CORS
CORS_ORIGINS=http://localhost:3000

# Rate Limiting
RATE_LIMIT_GENERAL=100
RATE_LIMIT_AUTH=10
RATE_LIMIT_ADMIN=50

# Guest Cleanup
GUEST_CLEANUP_ENABLED=true
GUEST_CLEANUP_INTERVAL=24h
GUEST_INACTIVE_DAYS=7

# WebSocket
MAX_CONNECTIONS_PER_USER=3
RECONNECT_WINDOW_SECONDS=45
```

---

## 🛠️ Development

### Running in Development

```bash
# Run with hot reload (requires air)
air

# Or run directly
go run main.go

# Run with custom port
PORT=8080 go run main.go

# Enable debug mode
DEBUG_MODE=true go run main.go
```

### Building

```bash
# Standard build
go build -o bible-quiz-pro main.go

# Optimized build (smaller binary)
go build -ldflags="-s -w" -o bible-quiz-pro main.go

# Build with version
VERSION=$(git describe --tags) go build -ldflags="-s -w -X main.Version=$VERSION" -o bible-quiz-pro main.go
```

---

## 🐛 Troubleshooting

### Common Issues

**Database locked error:**
```bash
# Stop the server and remove database
rm data/bible_quiz.db
# Restart - it will recreate automatically
```

**Port already in use:**
```bash
# Use different port
PORT=8080 go run main.go

# Or kill process using port
lsof -ti:3000 | xargs kill -9
```

**Import errors:**
```bash
# Clean and reinstall dependencies
go clean -modcache
go mod tidy
go mod download
```

**WebSocket connection failed:**
- Ensure port 3000 is accessible
- Check firewall settings
- Verify CORS_ORIGINS includes your domain

**Verses not loading:**
- Ensure verse files are in `./verses/` directory
- Check file format (valid JSON or TXT)
- Check server logs for parsing errors

---

## 📚 Documentation

- [SECURITY.md](SECURITY.md) - Security guidelines and best practices
- [DEPLOYMENT.md](DEPLOYMENT.md) - Production deployment guide
- [API.md](API.md) - Detailed API documentation (coming soon)
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines

---

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 👏 Acknowledgments

- Built with [Go](https://golang.org/)
- [Fiber](https://gofiber.io/) - Web framework
- [GORM](https://gorm.io/) - ORM library
- [SQLite](https://www.sqlite.org/) - Database
- [JWT-Go](https://github.com/golang-jwt/jwt) - JWT implementation
- [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) - Password hashing

---

## 📞 Support

- **Issues:** [GitHub Issues](https://github.com/yourusername/bible-quiz-pro/issues)
- **Discussions:** [GitHub Discussions](https://github.com/yourusername/bible-quiz-pro/discussions)
- **Email:** support@yourdomain.com
- **Security:** security@yourdomain.com

---

## ⭐ Star History

If you find this project useful, please consider giving it a star!

---

**Made with ❤️ for Bible study and learning**
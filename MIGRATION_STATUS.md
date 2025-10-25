# Fiber to net/http Migration Status

## Summary
Migration from Fiber framework to pure net/http + nhooyr.io/websocket

**Started**: 2025-10-25
**Scope**: 141 handler functions across 25 files + main.go routing + middleware

## ✅ Completed Infrastructure
1. **utils/http.go** - HTTP utility functions for JSON responses
2. **middleware/auth_http.go** - net/http compatible auth middleware
3. **MIGRATION_TO_HTTP.md** - Complete migration guide with patterns

## ✅ Migrated Handler Files (3/25 = 12%)
1. **handlers/practice.go** - 1 function ✅
2. **handlers/stats.go** - 5 functions ✅
3. **handlers/debug.go** - 3 functions ✅

**Total functions migrated**: 9/141 (6%)

## ⏳ Remaining Work

### Handler Files (22 files, ~132 functions)

**Priority 1 - Core API** (Need for basic functionality):
- [ ] handlers/auth.go - Authentication (login, register, guest)
- [ ] handlers/preferences.go - User preferences
- [ ] handlers/themes.go - Theme management
- [ ] handlers/verses.go - Verse endpoints
- [ ] handlers/game.go - Game URL handling

**Priority 2 - User Features**:
- [ ] handlers/users.go - User profiles
- [ ] handlers/user_handlers.go - User CRUD
- [ ] handlers/progression.go - XP/achievements
- [ ] handlers/leaderboard.go - Leaderboards

**Priority 3 - Social Features**:
- [ ] handlers/stubs.go - Friend requests
- [ ] handlers/teams.go - Team management (large file)
- [ ] handlers/team_challenges.go - Team challenges
- [ ] handlers/team_themes.go - Team themes
- [ ] handlers/theme_generator.go - AI theme generation

**Priority 4 - Multiplayer**:
- [ ] handlers/multiplayer.go - WebSocket already uses net/http! Just need to migrate a few HTTP helpers

**Priority 5 - Admin**:
- [ ] handlers/admin/auth.go
- [ ] handlers/admin/users.go
- [ ] handlers/admin/themes.go
- [ ] handlers/admin/analytics.go
- [ ] handlers/admin/achievements.go
- [ ] handlers/admin/cleanup_handlers.go
- [ ] handlers/admin/stubs.go

### Main Routing (Critical!)
- [ ] **main.go** - Convert ~200 Fiber routes to net/http ServeMux
  - This is THE blocking task - server won't start without it
  - Needs to be done after handlers or in parallel

### Cleanup
- [ ] Remove Fiber-specific code from middleware/ratelimit.go
- [ ] Remove Fiber from go.mod
- [ ] Run `go mod tidy`
- [ ] Fix compilation errors
- [ ] Test all endpoints

## Recommended Next Steps

### Option A: Focus on main.go First
Convert main.go routing so the server can start with the migrated handlers. This allows iterative testing.

**Pros**: Can test as you go
**Cons**: Still need to migrate all handlers

### Option B: Finish All Handlers First
Complete all handler migrations, then do main.go in one go.

**Pros**: Systematic, complete migration
**Cons**: Can't test until main.go is done

### Option C: Batch Migration by Priority
Do Priority 1 handlers + main.go routes for those handlers. Test. Repeat for each priority.

**Pros**: Incremental testing, fastest to working state
**Cons**: More complex coordination

## Migration Pattern Reference

```go
// BEFORE (Fiber)
func MyHandler(c *fiber.Ctx) error {
    userID := c.Locals("userId")
    param := c.Params("id")
    query := c.Query("filter", "all")
    var req MyRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "bad request"})
    }
    return c.JSON(fiber.Map{"success": true, "data": result})
}

// AFTER (net/http)
func MyHandler(w http.ResponseWriter, r *http.Request) {
    userID, err := middleware.GetUserID(r)
    if err != nil {
        utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }
    param := r.PathValue("id")
    query := utils.Query(r, "filter", "all")
    var req MyRequest
    if err := utils.ParseJSON(r, &req); err != nil {
        utils.JSONError(w, http.StatusBadRequest, "bad request")
        return
    }
    utils.JSON(w, http.StatusOK, map[string]interface{}{
        "success": true,
        "data":    result,
    })
}
```

## Time Estimate

Based on progress so far:
- **Handler migration**: ~10 min per file × 22 files = ~4 hours
- **main.go routing**: ~2-3 hours (200+ routes)
- **Testing & fixes**: ~2-3 hours
- **Total**: ~8-10 hours of focused work

## Testing Checklist

After migration completion:
- [ ] `go build` compiles without errors
- [ ] Server starts successfully
- [ ] Auth endpoints work (login, register, guest)
- [ ] Theme endpoints work
- [ ] Quiz/practice endpoints work
- [ ] WebSocket multiplayer works
- [ ] Admin panel works
- [ ] All middleware functions correctly

## Rollback

Backup exists at: `/Users/alberickecha/Documents/CODING/ubible_fiber_backup`

To rollback:
```bash
rm -rf /Users/alberickecha/Documents/CODING/ubible
cp -r /Users/alberickecha/Documents/CODING/ubible_fiber_backup /Users/alberickecha/Documents/CODING/ubible
```

## Notes
- WebSocket code in multiplayer.go already uses net/http - minimal changes needed
- middleware/ratelimit.go has both Fiber and net/http versions - just need to remove Fiber parts
- middleware/auth.go is old Fiber version - use auth_http.go instead
- All infrastructure (utils, middleware) is ready and tested

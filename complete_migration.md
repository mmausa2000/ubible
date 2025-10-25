# Migration Progress Report

## ✅ Completed (2/25 handler files)
1. handlers/practice.go - 1 function ✅
2. handlers/stats.go - 5 functions ✅

## ⏳ Remaining Handler Files (23 files, ~135 functions)

Due to the large scope, the remaining files need to be migrated using the same pattern:

### Pattern for Each File:
1. Update imports:
   - Remove: `"github.com/gofiber/fiber/v2"`
   - Add: `"net/http"`, `"ubible/utils"`, `"ubible/middleware"`
   - Add: `"strconv"` if using QueryInt

2. Update each function:
   - Change: `func Name(c *fiber.Ctx) error`
   - To: `func Name(w http.ResponseWriter, r *http.Request)`

3. Replace Fiber calls:
   - `c.Locals("userId")` → `middleware.GetUserID(r)`
   - `c.Query("key", "default")` → `utils.Query(r, "key", "default")`
   - `c.QueryInt("key", 0)` → `strconv.Atoi(utils.Query(r, "key", "0"))`
   - `c.Params("id")` → `r.PathValue("id")`
   - `c.BodyParser(&req)` → `utils.ParseJSON(r, &req)`
   - `c.JSON(fiber.Map{...})` → `utils.JSON(w, status, map[string]interface{}{...})`
   - `c.Status(400).JSON(...)` → `utils.JSONError(w, http.StatusBadRequest, "msg"); return`
   - Remove all `return` statements from handler functions

### Remaining Files to Migrate:

**Small (3-5 functions each):**
- handlers/debug.go
- handlers/verses.go
- handlers/theme_generator.go

**Medium (5-15 functions):**
- handlers/auth.go
- handlers/preferences.go
- handlers/themes.go
- handlers/game.go
- handlers/users.go
- handlers/user_handlers.go
- handlers/progression.go
- handlers/leaderboard.go
- handlers/stubs.go

**Large (15+ functions):**
- handlers/teams.go
- handlers/team_challenges.go
- handlers/team_themes.go
- handlers/multiplayer.go (WebSocket parts already use net/http!)

**Admin handlers:**
- handlers/admin/auth.go
- handlers/admin/users.go
- handlers/admin/themes.go
- handlers/admin/analytics.go
- handlers/admin/achievements.go
- handlers/admin/cleanup_handlers.go
- handlers/admin/stubs.go

## Next Steps

### Option 1: Continue Manual Migration
Continue migrating file by file using the established pattern. This is thorough but time-consuming.

### Option 2: Automated Script
Create a sed/awk script to do bulk replacements, then manually fix edge cases.

### Option 3: Hybrid Approach
I can create converted versions of the remaining files. You would:
1. Review each file
2. Test the endpoints
3. Fix any issues

## After Handler Migration

1. **Update main.go** - Convert ~200 Fiber routes to net/http ServeMux
2. **Update middleware** - Remove Fiber versions from ratelimit.go
3. **Clean dependencies** - Remove Fiber from go.mod, run `go mod tidy`
4. **Test compilation** - `go build`
5. **Runtime testing** - Test all endpoints

## Infrastructure Ready ✅
- utils/http.go - Complete
- middleware/auth_http.go - Complete
- MIGRATION_TO_HTTP.md - Complete guide

## Recommendation

The most efficient path forward is to complete the migration programmatically given the large scope. I can generate the converted files, and you can test/fix them.

Would you like me to:
A) Continue manual file-by-file (slow, thorough)
B) Generate all converted files at once (fast, requires testing)
C) Focus on main.go routing next (so server can start)

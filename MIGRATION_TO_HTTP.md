# Fiber to net/http Migration Guide

## Summary
Migrating from Fiber framework to standard net/http with nhooyr.io/websocket.

**Scope**: 141 handler functions across 25 files + main.go routing + middleware

## Status
- ✅ HTTP utilities created (`utils/http.go`)
- ✅ net/http auth middleware created (`middleware/auth_http.go`)
- ⏳ Handler migration (0/141 functions)
- ⏳ Main.go routing update
- ⏳ Remove Fiber from go.mod

## Migration Pattern

### Before (Fiber):
```go
func GetSomething(c *fiber.Ctx) error {
    userID := c.Locals("userId")
    param := c.Params("id")
    query := c.Query("filter", "all")

    var req RequestStruct
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data": result,
    })
}
```

### After (net/http):
```go
func GetSomething(w http.ResponseWriter, r *http.Request) {
    userID, err := middleware.GetUserID(r)
    if err != nil {
        utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }

    param := r.PathValue("id") // Go 1.22+ ServeMux
    query := utils.Query(r, "filter", "all")

    var req RequestStruct
    if err := utils.ParseJSON(r, &req); err != nil {
        utils.JSONError(w, http.StatusBadRequest, "Invalid request")
        return
    }

    utils.JSONSuccess(w, map[string]interface{}{
        "data": result,
    })
}
```

## Key Changes

### 1. Function Signature
- Before: `func Name(c *fiber.Ctx) error`
- After: `func Name(w http.ResponseWriter, r *http.Request)`

### 2. Context Values
- Before: `c.Locals("userId")`
- After: `r.Context().Value(middleware.UserIDKey)` or use `middleware.GetUserID(r)`

### 3. URL Parameters
- Before: `c.Params("id")`
- After: `r.PathValue("id")` (Go 1.22+) or use Chi router

### 4. Query Parameters
- Before: `c.Query("key", "default")`
- After: `utils.Query(r, "key", "default")`

### 5. Request Body
- Before: `c.BodyParser(&req)`
- After: `utils.ParseJSON(r, &req)`

### 6. JSON Responses
- Before: `c.JSON(fiber.Map{...})`
- After: `utils.JSON(w, status, data)` or `utils.JSONSuccess(w, data)`

### 7. Error Responses
- Before: `c.Status(400).JSON(fiber.Map{"error": "msg"})`
- After: `utils.JSONError(w, http.StatusBadRequest, "msg")`

### 8. Return Values
- Before: `return c.JSON(...)` or `return err`
- After: Just call the function, no return (void functions)

## Middleware Migration

### Before (Fiber):
```go
func MyMiddleware(c *fiber.Ctx) error {
    // do something
    c.Locals("key", value)
    return c.Next()
}

app.Use(MyMiddleware)
```

### After (net/http):
```go
func MyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // do something
        ctx := context.WithValue(r.Context(), KeyName, value)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

mux.Handle("/path", MyMiddleware(handler))
```

## Routing Migration

### Before (Fiber):
```go
app := fiber.New()
app.Get("/api/users/:id", handlers.GetUser)
app.Post("/api/users", middleware.AuthMiddleware, handlers.CreateUser)
```

### After (net/http with Go 1.22+ patterns):
```go
mux := http.NewServeMux()
mux.HandleFunc("GET /api/users/{id}", handlers.GetUser)
mux.Handle("POST /api/users", middleware.AuthMiddleware(http.HandlerFunc(handlers.CreateUser)))
```

## Files to Migrate

### Handlers (25 files, 141 functions):
1. `handlers/practice.go` - 1 function
2. `handlers/stats.go` - 5 functions
3. `handlers/debug.go` - 3 functions
4. `handlers/game.go` - ? functions
5. `handlers/auth.go` - ? functions
6. ... (continue for all 25 handler files)

### Main Routing:
- `main.go` - ~200 routes to convert

### Middleware:
- ✅ `middleware/auth_http.go` - DONE
- `middleware/ratelimit.go` - Keep net/http version, remove Fiber version

## Migration Order

1. **Phase 1**: Migrate small handler files (1-5 functions each)
   - stats.go
   - debug.go
   - practice.go

2. **Phase 2**: Migrate medium handler files (5-15 functions)
   - auth.go
   - themes.go
   - users.go

3. **Phase 3**: Migrate large handler files
   - teams.go
   - multiplayer.go (already uses net/http for WebSocket)

4. **Phase 4**: Update main.go routing

5. **Phase 5**: Clean up
   - Remove Fiber imports
   - Run `go mod tidy`
   - Test compilation
   - Fix runtime errors

## Testing After Migration

```bash
# Build to check for compilation errors
go build -o ubible_http main.go

# Run and test each endpoint
./ubible_http

# Test critical endpoints:
curl http://localhost:3000/api/themes
curl -H "Authorization: Bearer TOKEN" http://localhost:3000/api/auth/preferences
```

## Rollback Plan

Backup created at: `/Users/alberickecha/Documents/CODING/ubible_fiber_backup`

To rollback:
```bash
rm -rf /Users/alberickecha/Documents/CODING/ubible
cp -r /Users/alberickecha/Documents/CODING/ubible_fiber_backup /Users/alberickecha/Documents/CODING/ubible
```

## Notes

- This is a LARGE migration affecting the entire API surface
- Expected time: Several hours of careful work
- Must test each endpoint after migration
- WebSocket code already uses net/http - no changes needed there
- Consider migrating and testing one handler file at a time

## Recommendation

Given the scope (141 functions), consider:
1. **Option A**: Manual migration file-by-file (safest, most tedious)
2. **Option B**: Use a lightweight router like Chi (easier migration)
3. **Option C**: Keep Fiber (it works fine, just adds one dependency)

The user chose full migration to net/http only.

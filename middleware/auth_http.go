// middleware/auth_http.go - net/http compatible auth middleware
package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
	"ubible/database"
	"ubible/models"
	"ubible/utils"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDKey   contextKey = "userId"
	UsernameKey contextKey = "username"
	IsGuestKey  contextKey = "isGuest"
	IsAdminKey  contextKey = "isAdmin"
)

// AuthMiddleware validates JWT tokens for protected routes
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.JSONError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.JSONError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		tokenString := parts[1]
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "ubible-secret-change-in-production"
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			utils.JSONError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.JSONError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		exp, ok := claims["exp"].(float64)
		if !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
			utils.JSONError(w, http.StatusUnauthorized, "Token expired")
			return
		}

		// Add claims to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims["user_id"])
		ctx = context.WithValue(ctx, UsernameKey, claims["username"])
		ctx = context.WithValue(ctx, IsGuestKey, claims["is_guest"])

		// Update user's last activity
		updateUserActivity(claims["user_id"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminAuthMiddleware validates JWT tokens and checks for admin privileges
func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.JSONError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.JSONError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		tokenString := parts[1]
		jwtSecret := os.Getenv("JWT_SECRET")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			utils.JSONError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.JSONError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		isAdmin, ok := claims["is_admin"].(bool)
		if !ok || !isAdmin {
			utils.JSONError(w, http.StatusForbidden, "Access denied. Admin privileges required.")
			return
		}

		// Add claims to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims["user_id"])
		ctx = context.WithValue(ctx, UsernameKey, claims["username"])
		ctx = context.WithValue(ctx, IsAdminKey, true)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware validates JWT if present, but doesn't require it
func OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No auth header, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		tokenString := parts[1]
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "ubible-secret-change-in-production"
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			// Invalid token, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// Add claims to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims["user_id"])
		ctx = context.WithValue(ctx, UsernameKey, claims["username"])
		ctx = context.WithValue(ctx, IsGuestKey, claims["is_guest"])

		// Update user's last activity
		updateUserActivity(claims["user_id"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts user ID from context
func GetUserID(r *http.Request) (uint, error) {
	userIDRaw := r.Context().Value(UserIDKey)
	if userIDRaw == nil {
		return 0, errors.New("user not authenticated")
	}

	switch v := userIDRaw.(type) {
	case float64:
		return uint(v), nil
	case uint:
		return v, nil
	default:
		return 0, errors.New("invalid user ID type")
	}
}

// GetUsername extracts username from context
func GetUsername(r *http.Request) (string, error) {
	usernameRaw := r.Context().Value(UsernameKey)
	if usernameRaw == nil {
		return "", errors.New("user not authenticated")
	}

	username, ok := usernameRaw.(string)
	if !ok {
		return "", errors.New("invalid username type")
	}

	return username, nil
}

// updateUserActivity updates the user's last_activity timestamp
func updateUserActivity(userIDRaw interface{}) {
	db := database.GetDB()
	if db == nil {
		return
	}

	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		return
	}

	db.Model(&models.User{}).Where("id = ?", userID).Update("last_activity", time.Now())
}

// utils/http.go - HTTP utility functions for net/http
package utils

import (
	"encoding/json"
	"net/http"
)

// JSON sends a JSON response
func JSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// JSONError sends a JSON error response
func JSONError(w http.ResponseWriter, status int, message string) error {
	return JSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// JSONSuccess sends a JSON success response
func JSONSuccess(w http.ResponseWriter, data interface{}) error {
	response := map[string]interface{}{
		"success": true,
	}

	// Merge data into response
	if dataMap, ok := data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			response[k] = v
		}
	} else {
		response["data"] = data
	}

	return JSON(w, http.StatusOK, response)
}

// ParseJSON parses JSON request body
func ParseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Query gets a query parameter
func Query(r *http.Request, key string, defaultValue ...string) string {
	val := r.URL.Query().Get(key)
	if val == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return val
}

// Param gets a URL parameter (to be used with router that supports it)
func Param(r *http.Request, key string) string {
	return r.PathValue(key) // Go 1.22+ ServeMux pattern matching
}

// handlers/admin/stubs.go
package admin

import (
	"net/http"
)

// GetAllUsers - alias for GetUsers (for compatibility)
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	GetUsers(w, r)
}

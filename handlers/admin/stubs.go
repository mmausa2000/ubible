// handlers/admin/stubs.go
package admin

import (
	"github.com/gofiber/fiber/v2"
)

// GetAllUsers - alias for GetUsers (for compatibility)
func GetAllUsers(c *fiber.Ctx) error {
	return GetUsers(c)
}

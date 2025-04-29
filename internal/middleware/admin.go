package middleware

import (
	"backend-go/internal/models"

	"github.com/gofiber/fiber/v2"
)

func AdminMiddleware(c *fiber.Ctx) error {
    role := c.Locals("userRole").(models.UserRole)
    
    if role != models.RoleAdmin {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Admin access required",
        })
    }
    
    return c.Next()
}
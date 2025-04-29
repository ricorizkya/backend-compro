package middleware

import (
	"backend-go/internal/database"
	"backend-go/internal/models"
	"context"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(c *fiber.Ctx) error {
    authHeader := c.Get("Authorization")
    if authHeader == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Authorization header required",
        })
    }

    tokenString := ExtractToken(authHeader)
    if tokenString == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid token format",
        })
    }

    claims, err := validateToken(tokenString)
    if err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid token",
        })
    }

    var exists bool
    err = database.DB.QueryRow(context.Background(),
        "SELECT EXISTS(SELECT 1 FROM token_blacklist WHERE token = $1)",
        tokenString,
    ).Scan(&exists)
    
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to check token status",
        })
    }
    
    if exists {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Token revoked",
        })
    }

    // Simpan claims di context
    c.Locals("userID", claims.UserID)
    c.Locals("userRole", claims.Role)
    
    return c.Next()
}

func ExtractToken(header string) string {
    if len(header) > 7 && header[:7] == "Bearer " {
        return header[7:]
    }
    return ""
}

func validateToken(tokenString string) (*models.Claims, error) {
    claims := &models.Claims{}
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        return []byte(os.Getenv("JWT_SECRET")), nil
    })

    if err != nil || !token.Valid {
        return nil, fiber.ErrUnauthorized
    }

    return claims, nil
}
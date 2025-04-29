package handlers

import (
	"backend-go/internal/middleware"
	"backend-go/internal/models"
	"context"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
    db *pgxpool.Pool
}

func NewAuthHandler(db *pgxpool.Pool) *AuthHandler {
    return &AuthHandler{db: db}
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user and get JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      models.LoginRequest  true  "Login Credentials"
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
    var req models.LoginRequest
    
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // Cari user berdasarkan username
    var user models.User
    query := `SELECT id, password, role FROM users WHERE username = $1 AND deleted_at IS NULL`
    err := h.db.QueryRow(context.Background(), query, req.Username).Scan(
        &user.ID,
        &user.Password,
        &user.Role,
    )

    if err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid username or password",
        })
    }

    // Bandingkan password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid username or password",
        })
    }

    // Buat JWT token
    claims := models.Claims{
        UserID: user.ID,
        Role:   user.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signedToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to generate token",
        })
    }

    return c.JSON(fiber.Map{
        "token":   signedToken,
        "expires": claims.ExpiresAt.Time.Format(time.RFC3339),
    })
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
    authHeader := c.Get("Authorization")
    tokenString := middleware.ExtractToken(authHeader)
    
    if tokenString == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "No token provided",
        })
    }

    // Parse token untuk mendapatkan expiry time
    claims := &models.Claims{}
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        return []byte(os.Getenv("JWT_SECRET")), nil
    })
    
    if err != nil || !token.Valid {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid token",
        })
    }

    // Masukkan token ke blacklist
    _, err = h.db.Exec(context.Background(),
        `INSERT INTO token_blacklist (token, expires_at)
         VALUES ($1, $2)`,
        tokenString,
        claims.ExpiresAt.Time,
    )
    
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to logout",
        })
    }

    return c.JSON(fiber.Map{
        "message": "Successfully logged out",
    })
}
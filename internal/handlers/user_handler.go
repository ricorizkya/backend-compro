package handlers

import (
	"backend-go/internal/models"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	db *pgxpool.Pool
}

func NewUserHandler(db *pgxpool.Pool) *UserHandler {
	return &UserHandler{db: db}
}

// CreateUser membuat user baru
// @Summary      Create new user
// @Description  Create new user account
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user  body      models.CreateRequest  true  "User Data"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
    createdBy := c.Locals("userID").(int)
    var req models.CreateRequest
	
	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validasi input
	if validationErr := validateUserInput(req); validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(validationErr)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Set default role jika kosong
	if req.Role == "" {
		req.Role = models.RoleUser
	}

	// Eksekusi query
	query := `
		INSERT INTO users (
			name, 
			phone, 
			username, 
			password, 
			role,
			created_by
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var userID int
	err = h.db.QueryRow(context.Background(), query,
        req.Name,
        req.Phone,
        req.Username,
        string(hashedPassword),
        req.Role,
        createdBy,
    ).Scan(&userID)

	if err != nil {
		// Handle unique constraint violation
		if isUniqueConstraintViolation(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Username or phone number already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":      userID,
		"message": "User created successfully",
	})
}

func validateUserInput(req models.CreateRequest) fiber.Map {
	errors := make(fiber.Map)
	
	if len(req.Name) < 3 {
		errors["name"] = "Name must be at least 3 characters"
	}
	
	if req.Phone == "" {
		errors["phone"] = "Phone number is required"
	}
	
	if req.Username == "" {
		errors["username"] = "Username is required"
	}
	
	if len(req.Password) < 8 {
		errors["password"] = "Password must be at least 8 characters"
	}
	
	if len(errors) > 0 {
		return fiber.Map{"errors": errors}
	}
	return nil
}

func isUniqueConstraintViolation(err error) bool {
	// Error code 23505 adalah unique_violation di PostgreSQL
	return err.Error()[0:5] == "ERROR" && err.Error()[6:10] == "23505"
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
    userID := c.Params("id")
    id, err := strconv.Atoi(userID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid user ID",
        })
    }

    var req models.UpdateRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // Validasi minimal satu field diupdate
    if req.Name == "" && req.Phone == "" && req.Username == "" && 
       req.Password == "" && req.Role == "" && req.Status == nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "At least one field must be provided",
        })
    }

    // Validasi role
    if req.Role != "" && !isValidRole(req.Role) {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid role",
        })
    }

    // Hash password jika diupdate
    var hashedPassword string
    if req.Password != "" {
        hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to process password",
            })
        }
        hashedPassword = string(hash)
    }

    // Dapatkan user yang melakukan edit dari middleware auth
    editedBy := c.Locals("userID").(int)

    query, args := buildUpdateQuery(req, hashedPassword, editedBy, id)
    
    result, err := h.db.Exec(context.Background(), query, args...)
    if err != nil {
        if isUniqueConstraintViolation(err) {
            return c.Status(fiber.StatusConflict).JSON(fiber.Map{
                "error": "Username or phone number already exists",
            })
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to update user",
        })
    }

    if result.RowsAffected() == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    return c.JSON(fiber.Map{
        "message": "User updated successfully",
    })
}

func buildUpdateQuery(req models.UpdateRequest, hashedPassword string, editedBy, id int) (string, []interface{}) {
    var query strings.Builder
    args := make([]interface{}, 0)
    counter := 1

    query.WriteString("UPDATE users SET ")
    
    if req.Name != "" {
        query.WriteString(fmt.Sprintf("name = $%d, ", counter))
        args = append(args, req.Name)
        counter++
    }
    
    if req.Phone != "" {
        query.WriteString(fmt.Sprintf("phone = $%d, ", counter))
        args = append(args, req.Phone)
        counter++
    }
    
    if req.Username != "" {
        query.WriteString(fmt.Sprintf("username = $%d, ", counter))
        args = append(args, req.Username)
        counter++
    }
    
    if hashedPassword != "" {
        query.WriteString(fmt.Sprintf("password = $%d, ", counter))
        args = append(args, hashedPassword)
        counter++
    }
    
    if req.Role != "" {
        query.WriteString(fmt.Sprintf("role = $%d, ", counter))
        args = append(args, req.Role)
        counter++
    }
    
    if req.Status != nil {
        query.WriteString(fmt.Sprintf("status = $%d, ", counter))
        args = append(args, *req.Status)
        counter++
    }
    
    // Tambahkan edited_by dan haphis koma terakhir
    query.WriteString(fmt.Sprintf(
        "edited_by = $%d ",
        counter,
    ))
    args = append(args, editedBy)
    counter++
    
    query.WriteString(fmt.Sprintf("WHERE id = $%d", counter))
    args = append(args, id)
    
    return query.String(), args
}

func isValidRole(role models.UserRole) bool {
	switch role {
	case models.RoleAdmin, models.RoleStaff, models.RoleUser:
		return true
	default:
		return false
	}
}

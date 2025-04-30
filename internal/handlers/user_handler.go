package handlers

import (
	"backend-go/internal/models"
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

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

// UpdateUser godoc
// @Summary      Update user data
// @Description  Update existing user's information
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      int                  true  "User ID"
// @Param        user body      models.UpdateRequest true  "User Data"
// @Security     ApiKeyAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
    userID := c.Params("id")
    targetID, err := strconv.Atoi(userID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid user ID format",
        })
    }

    // Dapatkan ID user yang melakukan request dari JWT
    requesterID := c.Locals("userID").(int)
    requesterRole := c.Locals("userRole").(models.UserRole)

    // Authorization check
    if requesterRole != models.RoleAdmin && requesterID != targetID {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "You can only update your own profile",
        })
    }

    var req models.UpdateRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // Validasi input
    if validationErr := validateUpdateRequest(req); validationErr != nil {
        return c.Status(fiber.StatusBadRequest).JSON(validationErr)
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

    // Build dynamic query
    query, args := buildUpdateQuery(req, hashedPassword, requesterID, targetID, requesterRole)
    
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

func validateUpdateRequest(req models.UpdateRequest) fiber.Map {
    errors := make(fiber.Map)
    
    if req.Name != "" && len(req.Name) < 3 {
        errors["name"] = "Name must be at least 3 characters"
    }
    
    if req.Phone != "" {
        if !isValidPhone(req.Phone) {
            errors["phone"] = "Invalid phone number format"
        }
    }
    
    if req.Username != "" && !isAlphanumeric(req.Username) {
        errors["username"] = "Username must be alphanumeric"
    }
    
    if req.Password != "" && len(req.Password) < 8 {
        errors["password"] = "Password must be at least 8 characters"
    }
    
    if len(errors) > 0 {
        return fiber.Map{"errors": errors}
    }
    return nil
}

func buildUpdateQuery(req models.UpdateRequest, hashedPassword string, editedBy, targetID int, requestRole models.UserRole) (string, []interface{}) {
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
    
    if req.Role != "" && requestRole == models.RoleAdmin { // Gunakan requesterRole
        query.WriteString(fmt.Sprintf("role = $%d, ", counter))
        args = append(args, req.Role)
        counter++
    }
    
    if req.Status != nil && requestRole == models.RoleAdmin { // Gunakan requesterRole
        query.WriteString(fmt.Sprintf("status = $%d, ", counter))
        args = append(args, *req.Status)
        counter++
    }
    
    // Update edited_by dan edited_at
    query.WriteString(fmt.Sprintf(
        "edited_by = $%d ",
        counter,
    ))
    args = append(args, editedBy)
    counter++
    
    query.WriteString(fmt.Sprintf("WHERE id = $%d", counter))
    args = append(args, targetID)
    
    return query.String(), args
}

func isAlphanumeric(s string) bool {
    for _, r := range s {
        if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
            return false
        }
    }
    return true
}

func isValidPhone(phone string) bool {
    // Implementasi validasi nomor telepon sesuai kebutuhan
    return strings.HasPrefix(phone, "+") && len(phone) > 8
}

// GetUsers godoc
// @Summary      Get all users
// @Description  Get list of users with pagination
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        page    query     int      false  "Page number"     default(1)
// @Param        limit   query     int      false  "Items per page"  default(10)
// @Param        role    query     string   false  "Filter by role"
// @Param        status  query     bool     false  "Filter by status"
// @Security     ApiKeyAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /users [get]
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
    // Authorization - hanya admin yang bisa akses
    requesterRole := c.Locals("userRole").(models.UserRole)
    if requesterRole != models.RoleAdmin {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Admin access required",
        })
    }

    // Parse query parameters
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "10"))
    role := c.Query("role")
    status := c.Query("status")

    // Validasi input
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 10
    }
    offset := (page - 1) * limit

    // Build query
    query := `SELECT 
                id, name, phone, username, role, status, 
                created_at, created_by, edited_at, edited_by 
              FROM users 
              WHERE deleted_at IS NULL`
    args := []interface{}{}
    paramCounter := 1

    // Filter role
    if role != "" {
        query += fmt.Sprintf(" AND role = $%d", paramCounter)
        args = append(args, role)
        paramCounter++
    }

    // Filter status
    if status != "" {
        query += fmt.Sprintf(" AND status = $%d", paramCounter)
        args = append(args, status == "true")
        paramCounter++
    }

    // Add pagination
    query += fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", paramCounter, paramCounter+1)
    args = append(args, limit, offset)

    // Eksekusi query
    rows, err := h.db.Query(context.Background(), query, args...)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch users",
        })
    }
    defer rows.Close()

    var users []models.UserResponse
    for rows.Next() {
        var user models.UserResponse
        err := rows.Scan(
            &user.ID,
            &user.Name,
            &user.Phone,
            &user.Username,
            &user.Role,
            &user.Status,
            &user.CreatedAt,
            &user.CreatedBy,
            &user.EditedAt,
            &user.EditedBy,
        )
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to parse user data",
            })
        }
        users = append(users, user)
    }

    // Get total count
    countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
    countArgs := []interface{}{}
    paramCounter = 1

    if role != "" {
        countQuery += fmt.Sprintf(" AND role = $%d", paramCounter)
        countArgs = append(countArgs, role)
        paramCounter++
    }

    if status != "" {
        countQuery += fmt.Sprintf(" AND status = $%d", paramCounter)
        countArgs = append(countArgs, status == "true")
        paramCounter++
    }

    var total int
    err = h.db.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to get total users",
        })
    }

    return c.JSON(fiber.Map{
        "data": users,
        "meta": fiber.Map{
            "page":       page,
            "limit":      limit,
            "total":      total,
            "totalPages": int(math.Ceil(float64(total) / float64(limit))),
        },
    })
}

// DeleteUser godoc
// @Summary      Delete a user (soft delete)
// @Description  Mark user as deleted by setting deleted_at timestamp
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Security     ApiKeyAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /users/{id} [delete]
func (h * UserHandler) DeleteUser(c * fiber.Ctx) error {
    // Dapatkan ID user target
    userID := c.Params("id")
    targetID, err := strconv.Atoi(userID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid user ID format",
        })
    }

    // Dapatkan ID user yang melakukan request
    adminID := c.Locals("userID").(int)
    adminRole := c.Locals("userRole").(models.UserRole)

    // Authorization: hanya admin yang bisa menghapus user
    if adminRole != models.RoleAdmin {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Admin access required",
        })
    }

    // Cegah admin menghapus dirinya sendiri
    if targetID == adminID {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Admin cannot delete their own account",
        })
    }

    // Soft delete user
    query := `
        UPDATE users 
        SET deleted_at = $1, deleted_by = $2 
        WHERE id = $3 AND deleted_at IS NULL
    `

    result, err := h.db.Exec(context.Background(), query, time.Now().UTC(), adminID, targetID)

    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to delete user",
        })
    }

    if result.RowsAffected() == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found or already deleted",
        })
    }

    return c.JSON(fiber.Map{
        "message": "User deleted successfully",
    })
}

// RegisterUser godoc
// @Summary      Register new user
// @Description  Create user account for public registration
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.RegisterRequest  true  "Registration data"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /register [post]
func (h *UserHandler) RegisterUser(c *fiber.Ctx) error {
    var req models.RegisterRequest
    
    // Parse request body
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // Validasi input khusus registrasi
    if validationErr := validateRegistrationInput(req); validationErr != nil {
        return c.Status(fiber.StatusBadRequest).JSON(validationErr)
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to process password",
        })
    }

    // Set default role untuk registrasi publik
    if req.Role == "" {
        req.Role = string(models.RoleAdmin)
    }

    // Eksekusi query
    query := `
        INSERT INTO users (
            name, 
            phone, 
            username, 
            password, 
            role
        ) VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `

    var userID int
    err = h.db.QueryRow(context.Background(), query,
        req.Name,
        req.Phone,
        req.Username,
        string(hashedPassword),
        req.Role,
    ).Scan(&userID)

    if err != nil {
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
        "message": "User registered successfully",
    })
}

func validateRegistrationInput(req models.RegisterRequest) fiber.Map {
    errors := make(map[string]string)
    
    // Validasi Nama
    req.Name = strings.TrimSpace(req.Name)
    if req.Name == "" {
        errors["name"] = "Nama harus diisi"
    } else if len(req.Name) < 3 {
        errors["name"] = "Nama minimal 3 karakter"
    } else if len(req.Name) > 100 {
        errors["name"] = "Nama maksimal 100 karakter"
    }

    // Validasi Nomor Telepon
    req.Phone = strings.TrimSpace(req.Phone)
    if req.Phone == "" {
        errors["phone"] = "Nomor telepon harus diisi"
    } else {
        // Cek apakah numeric
        if _, err := strconv.Atoi(req.Phone); err != nil {
            errors["phone"] = "Nomor telepon harus angka"
        } else if len(req.Phone) < 10 {
            errors["phone"] = "Nomor telepon minimal 10 digit"
        } else if len(req.Phone) > 15 {
            errors["phone"] = "Nomor telepon maksimal 15 digit"
        }
    }

    // Validasi Username
    req.Username = strings.TrimSpace(req.Username)
    if req.Username == "" {
        errors["username"] = "Username harus diisi"
    } else if len(req.Username) < 5 {
        errors["username"] = "Username minimal 5 karakter"
    } else if len(req.Username) > 50 {
        errors["username"] = "Username maksimal 50 karakter"
    } else {
        // Cek format username (hanya huruf, angka, dan underscore)
        matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, req.Username)
        if !matched {
            errors["username"] = "Username hanya boleh mengandung huruf, angka, dan underscore"
        }
    }

    // Validasi Password
    req.Password = strings.TrimSpace(req.Password)
    if req.Password == "" {
        errors["password"] = "Password harus diisi"
    } else if len(req.Password) < 8 {
        errors["password"] = "Password minimal 8 karakter"
    } else if len(req.Password) > 72 {
        errors["password"] = "Password maksimal 72 karakter"
    } else {
        // Cek kompleksitas password
        var (
            hasUpper  = regexp.MustCompile(`[A-Z]`).MatchString(req.Password)
            hasLower  = regexp.MustCompile(`[a-z]`).MatchString(req.Password)
            hasNumber = regexp.MustCompile(`[0-9]`).MatchString(req.Password)
        )
        
        if !hasUpper {
            errors["password"] = "Password harus mengandung minimal 1 huruf besar"
        }
        if !hasLower {
            errors["password"] = "Password harus mengandung minimal 1 huruf kecil"
        }
        if !hasNumber {
            errors["password"] = "Password harus mengandung minimal 1 angka"
        }
    }

    if len(errors) > 0 {
        return fiber.Map{
            "error":   "Validasi gagal",
            "details": errors,
        }
    }
    return nil
}
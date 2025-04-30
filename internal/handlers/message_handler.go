package handlers

import (
	"backend-go/internal/models"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MessageHandler handles message-related operations
type MessageHandler struct {
    db        *pgxpool.Pool
}

func NewMessagesHandler(db *pgxpool.Pool) *MessageHandler {
    return &MessageHandler{db: db}
}

// CreateMessage godoc
// @Summary      Create new message
// @Description  Add new message to the system
// @Tags         messages
// @Accept       multipart/form-data
// @Param        name        formData  string  true  "Sender name"
// @Param        company     formData  string  false "Company name"
// @Param        product_id  formData  int     false "Related product ID"
// @Param        address     formData  string  false "Physical address"
// @Param        description formData  string  true  "Message content"
// @Security     ApiKeyAuth
// @Success      201  {object}  models.Message
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /messages [post]
func (h *MessageHandler) CreateMessage(c *fiber.Ctx) error {
    // Dapatkan user yang membuat
    userID := c.Locals("userID").(int)
    
    // Parse form data
    var req models.MessageCreateRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid form data",
        })
    }

    // Validasi manual
    var validationErrors []string
    
    // Validasi name
    req.Name = strings.TrimSpace(req.Name)
    if req.Name == "" {
        validationErrors = append(validationErrors, "name is required")
    } else if len(req.Name) > 100 {
        validationErrors = append(validationErrors, "name max length is 100 characters")
    }

    // Validasi company
    req.Company = strings.TrimSpace(req.Company)
    if len(req.Company) > 100 {
        validationErrors = append(validationErrors, "company max length is 100 characters")
    }

    // Validasi description
    req.Description = strings.TrimSpace(req.Description)
    if req.Description == "" {
        validationErrors = append(validationErrors, "description is required")
    }

    if len(validationErrors) > 0 {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Validation failed",
            "details": validationErrors,
        })
    }

    // Validasi product_id jika ada
    if req.ProductID != nil {
        var exists bool
        err := h.db.QueryRow(
            context.Background(),
            "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)",
            *req.ProductID,
        ).Scan(&exists)
        
        if err != nil || !exists {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid product ID",
            })
        }
    }

    // Insert ke database
    query := `
        INSERT INTO messages (
            name,
            company,
            id_product,
            address,
            description,
            created_by
        ) VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at
    `

    var message models.Message
    err := h.db.QueryRow(context.Background(), query,
        req.Name,
        req.Company,
        req.ProductID,
        req.Address,
        req.Description,
        userID,
    ).Scan(&message.ID, &message.CreatedAt)

    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create message: " + err.Error(),
        })
    }

    // Isi response
    message.Name = req.Name
    message.Company = req.Company
    message.ProductID = req.ProductID
    message.Address = req.Address
    message.Description = req.Description
    message.CreatedBy = userID

    return c.Status(fiber.StatusCreated).JSON(message)
}

// UpdateMessage godoc
// @Summary      Update message
// @Description  Update existing message data
// @Tags         messages
// @Accept       multipart/form-data
// @Produce      json
// @Param        id           path      int     true  "Message ID"
// @Param        name         formData  string  false "Sender name"
// @Param        company      formData  string  false "Company name"
// @Param        product_id   formData  int     false "Related product ID"
// @Param        address      formData  string  false "Physical address"
// @Param        description  formData  string  false "Message content"
// @Security     ApiKeyAuth
// @Success      200  {object}  models.Message
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /messages/{id} [put]
func (h *MessageHandler) UpdateMessage(c *fiber.Ctx) error {
    userID := c.Locals("userID").(int)
    
    // Parse ID
    messageID := c.Params("id")
    id, err := strconv.Atoi(messageID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid message ID format",
        })
    }

    // Parse form data
    var req models.MessageUpdateRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid form data",
        })
    }

    // Validasi manual
    var validationErrors []string

    if req.Name != nil {
        *req.Name = strings.TrimSpace(*req.Name)
        if *req.Name == "" {
            validationErrors = append(validationErrors, "name cannot be empty")
        } else if len(*req.Name) > 100 {
            validationErrors = append(validationErrors, "name max length is 100 characters")
        }
    }

    if req.Company != nil {
        *req.Company = strings.TrimSpace(*req.Company)
        if len(*req.Company) > 100 {
            validationErrors = append(validationErrors, "company max length is 100 characters")
        }
    }

    if req.Description != nil {
        *req.Description = strings.TrimSpace(*req.Description)
        if *req.Description == "" {
            validationErrors = append(validationErrors, "description cannot be empty")
        }
    }

    if len(validationErrors) > 0 {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Validation failed",
            "details": validationErrors,
        })
    }

    // Validasi product_id jika ada
    if req.ProductID != nil {
        var exists bool
        err := h.db.QueryRow(
            context.Background(),
            "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)",
            *req.ProductID,
        ).Scan(&exists)
        
        if err != nil || !exists {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid product ID",
            })
        }
    }

    // Build dynamic query
    query := `
        UPDATE messages SET
            name = COALESCE(NULLIF($1, ''), name),
            company = COALESCE(NULLIF($2, ''), company),
            id_product = COALESCE($3, id_product),
            address = COALESCE(NULLIF($4, ''), address),
            description = COALESCE(NULLIF($5, ''), description),
            edited_by = $6
        WHERE id = $7
        RETURNING *
    `

    args := []interface{}{
        req.Name,
        req.Company,
        req.ProductID,
        req.Address,
        req.Description,
        userID,
        id,
    }

    var message models.Message
    err = h.db.QueryRow(
        context.Background(),
        query,
        args...,
    ).Scan(
        &message.ID,
        &message.Name,
        &message.Company,
        &message.ProductID,
        &message.Address,
        &message.Description,
        &message.CreatedAt,
        &message.CreatedBy,
        &message.EditedAt,
        &message.EditedBy,
        &message.DeletedAt,
        &message.DeletedBy,
    )

    if err != nil {
        if err == pgx.ErrNoRows {
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
                "error": "Message not found",
            })
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to update message: " + err.Error(),
        })
    }

    return c.JSON(message)
}

// DeleteMessage godoc
// @Summary      Delete message
// @Description  Soft delete a message by marking it as deleted
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Message ID"
// @Security     ApiKeyAuth
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /messages/{id} [delete]
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
    // Dapatkan user yang melakukan delete
    userID := c.Locals("userID").(int)
    
    // Parse ID
    messageID := c.Params("id")
    id, err := strconv.Atoi(messageID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid message ID format",
        })
    }

    // Lakukan soft delete
    query := `
        UPDATE messages 
        SET 
            deleted_at = $1,
            deleted_by = $2
        WHERE 
            id = $3 
            AND deleted_at IS NULL
        RETURNING id, deleted_at
    `
    
    var (
        deletedID int
        deletedAt time.Time
    )
    
    err = h.db.QueryRow(
        c.Context(),
        query,
        time.Now(),
        userID,
        id,
    ).Scan(&deletedID, &deletedAt)

    if err != nil {
        if err == pgx.ErrNoRows {
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
                "error": "Message not found or already deleted",
            })
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to delete message: " + err.Error(),
        })
    }

    return c.SendStatus(fiber.StatusNoContent)
}

// GetMessages godoc
// @Summary      Get all messages
// @Description  Retrieve all active messages with optional product info
// @Tags         messages
// @Produce      json
// @Param        page      query   int     false  "Page number"
// @Param        limit     query   int     false  "Items per page"
// @Param        product_id query  int     false  "Filter by product ID"
// @Success      200  {array}  models.MessageWithProduct
// @Failure      500  {object}  map[string]string
// @Router       /messages [get]
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
    // Parse query parameters
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "10"))
    productID, _ := strconv.Atoi(c.Query("product_id"))
    
    offset := (page - 1) * limit

    // Build query
    query := `
        SELECT 
            m.id,
            m.name,
            m.company,
            m.id_product,
            m.address,
            m.description,
            m.created_at,
            m.created_by,
            m.edited_at,
            p.title as product_name,
            p.image as product_image
        FROM messages m
        LEFT JOIN products p ON m.id_product = p.id
        WHERE m.deleted_at IS NULL
    `
    
    args := []interface{}{}
    argCounter := 1
    
    if productID > 0 {
        query += fmt.Sprintf(" AND m.id_product = $%d", argCounter)
        args = append(args, productID)
        argCounter++
    }
    
    query += " ORDER BY m.created_at DESC"
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCounter, argCounter+1)
    args = append(args, limit, offset)

    rows, err := h.db.Query(c.Context(), query, args...)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch messages: " + err.Error(),
        })
    }
    defer rows.Close()

    var messages []models.MessageWithProduct
    for rows.Next() {
        var msg models.MessageWithProduct
        err := rows.Scan(
            &msg.ID,
            &msg.Name,
            &msg.Company,
            &msg.ProductID,
            &msg.Address,
            &msg.Description,
            &msg.CreatedAt,
            &msg.CreatedBy,
            &msg.EditedAt,
            &msg.ProductName,
            &msg.ProductImage,
        )
        
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to parse messages: " + err.Error(),
            })
        }
        messages = append(messages, msg)
    }

    if messages == nil {
        return c.JSON([]interface{}{})
    }

    return c.JSON(messages)
}
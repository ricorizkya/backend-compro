package handlers

import (
	"backend-go/internal/models"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CarouselHandler struct {
	db *pgxpool.Pool
}

func NewCarouselHandler(db *pgxpool.Pool) *CarouselHandler {
    return &CarouselHandler{db: db}
}

// CreateCarousel godoc
// @Summary      Create new carousel
// @Description  Add new carousel item
// @Tags         carousel
// @Accept       multipart/form-data
// @Produce      json
// @Param        image       formData  file    true  "Carousel image"
// @Param        title       formData  string  true  "Carousel title"
// @Param        description formData  string  false "Carousel description"
// @Param        status      formData  bool    false "Carousel status"
// @Security     ApiKeyAuth
// @Success      201  {object}  models.Carousel
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /carousel [post]
func (h * CarouselHandler) CreateCarousel(c * fiber.Ctx) error {
	// Dapatkan user yang membuat
	userID := c.Locals("userID").(int)

	// Parse form data
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image is required",
		})
	}

	var req models.CarouselCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid form data",
		})
	}

	// Validasi input
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Title is required",
		})
	}

	// Simpan gambar
	uploadPath := "uploads/carousel/"
	if err := os.MkdirAll(uploadPath, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), req.Title, ext)
	filePath := filepath.Join(uploadPath, filename)

	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save image",
		})
	}

	// Simpan data carousel ke database
	query := `
	INSERT INTO carousel (
            image, 
            title, 
            description, 
            status, 
            created_by
        ) VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at
	`

	var carousel models.Carousel
	err = h.db.QueryRow(context.Background(), query,
        "uploads/carousel/"+filename,
        req.Title,
        req.Description,
        req.Status,
        userID,
    ).Scan(&carousel.ID, &carousel.CreatedAt)

	if err != nil {
        // Hapus file yang sudah diupload jika gagal insert
        os.Remove(filePath)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create carousel",
        })
    }

	carousel.Image = "uploads/carousel/" + filename
    carousel.Title = req.Title
    carousel.Description = req.Description
    carousel.Status = req.Status
	carousel.CreatedBy = &userID

    return c.Status(fiber.StatusCreated).JSON(carousel)
}

// UpdateCarousel godoc
// @Summary      Update carousel item
// @Description  Update existing carousel data
// @Tags         carousel
// @Accept       multipart/form-data
// @Produce      json
// @Param        id          path      int     true  "Carousel ID"
// @Param        image       formData  file    false "New carousel image"
// @Param        title       formData  string  false "Carousel title"
// @Param        description formData  string  false "Carousel description"
// @Param        status      formData  bool    false "Carousel status"
// @Security     ApiKeyAuth
// @Success      200  {object}  models.Carousel
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /carousel/{id} [put]
func (h *CarouselHandler) UpdateCarousel(c *fiber.Ctx) error {
    carouselID := c.Params("id")
    id, err := strconv.Atoi(carouselID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid carousel ID",
        })
    }

    // Dapatkan user yang melakukan update
    userID := c.Locals("userID").(int)
    
    // Cek apakah carousel ada
    var existingImage string
    err = h.db.QueryRow(context.Background(),
        "SELECT image FROM carousel WHERE id = $1 AND deleted_at IS NULL",
        id,
    ).Scan(&existingImage)
    
    if err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Carousel not found",
        })
    }

    // Parse form data secara manual
    title := c.FormValue("title")
    description := c.FormValue("description")
    statusStr := c.FormValue("status")
    
    // Konversi status ke boolean
    var status *bool
    if statusStr != "" {
        statusVal, err := strconv.ParseBool(statusStr)
        if err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid status value",
            })
        }
        status = &statusVal
    }

    req := models.CarouselUpdateRequest{
        Title:       title,
        Description: description,
        Status:      status,
    }

    // Handle image upload
    file, _ := c.FormFile("image")
    var newImagePath string
    
    if file != nil {
        // Upload new image
        uploadDir := "uploads/carousel/"
        if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to create upload directory",
            })
        }
        
        ext := filepath.Ext(file.Filename)
        filename := fmt.Sprintf("%d-%s%s", 
            time.Now().UnixNano(), 
            strings.ReplaceAll(req.Title, " ", "_"),
            ext,
        )
        
        // Jika title kosong
        if req.Title == "" {
            filename = fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
        }
        
        filePath := filepath.Join(uploadDir, filename)
        
        if err := c.SaveFile(file, filePath); err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to save image",
            })
        }
        newImagePath = "uploads/carousel/" + filename
        
        // Delete old image
        go func(oldImage string) {
            if oldImage != "" {
                os.Remove("." + oldImage)
            }
        }(existingImage)
    }

    // Build dynamic query
    query := `UPDATE carousel SET
                image = COALESCE(NULLIF($1, ''), image),
                title = COALESCE(NULLIF($2, ''), title),
                description = COALESCE(NULLIF($3, ''), description),
                status = COALESCE($4, status),
                edited_by = $5
              WHERE id = $6
              RETURNING *`

    args := []interface{}{
        newImagePath,
        req.Title,
        req.Description,
        req.Status,
        userID,
        id,
    }

    var carousel models.Carousel
    err = h.db.QueryRow(context.Background(), query, args...).Scan(
        &carousel.ID,
        &carousel.Image,
        &carousel.Title,
        &carousel.Description,
        &carousel.Status,
        &carousel.CreatedAt,
        &carousel.CreatedBy,
        &carousel.EditedAt,
        &carousel.EditedBy,
        &carousel.DeletedAt,
        &carousel.DeletedBy,
    )

    if err != nil {
        if newImagePath != "" {
            os.Remove("." + newImagePath)
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to update carousel",
        })
    }

    return c.JSON(carousel)
}

// DeleteCarousel godoc
// @Summary      Delete carousel item (soft delete)
// @Description  Mark carousel as deleted and remove associated image
// @Tags         carousel
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Carousel ID"
// @Security     ApiKeyAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /carousel/{id} [delete]
func (h *CarouselHandler) DeleteCarousel(c *fiber.Ctx) error {
    carouselID := c.Params("id")
    id, err := strconv.Atoi(carouselID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid carousel ID format",
        })
    }

    // Dapatkan admin yang melakukan delete
    adminID := c.Locals("userID").(int)
    adminRole := c.Locals("userRole").(models.UserRole)

    if adminRole != models.RoleAdmin {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Admin access required",
        })
    }

    // Dapatkan path gambar dan validasi keberadaan
    var imagePath string
    err = h.db.QueryRow(context.Background(),
        `SELECT image FROM carousel 
         WHERE id = $1 AND deleted_at IS NULL`,
        id,
    ).Scan(&imagePath)

    if err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Carousel not found or already deleted",
        })
    }

    // Soft delete di database
    query := `
        UPDATE carousel 
        SET deleted_at = $1, deleted_by = $2 
        WHERE id = $3
    `
    result, err := h.db.Exec(
        context.Background(),
        query,
        time.Now().UTC(),
        adminID,
        id,
    )

    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to delete carousel",
        })
    }

    if result.RowsAffected() == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Carousel not found",
        })
    }

    // Hapus file gambar
    if imagePath != "" {
        go func(path string) {
            fullPath := path // karena path disimpan sebagai "/uploads/..."
            if err := os.Remove(fullPath); err != nil {
                log.Printf("Failed to delete image: %s. Error: %v", path, err)
            }
        }(imagePath)
    }

    return c.JSON(fiber.Map{
        "message": "Carousel deleted successfully",
    })
}

// GetCarousels godoc
// @Summary      Get all carousel items
// @Description  Get list of carousels with optional filters
// @Tags         carousel
// @Accept       json
// @Produce      json
// @Param        page    query     int     false  "Page number"     default(1)
// @Param        limit   query     int     false  "Items per page"  default(10)
// @Param        status  query     bool    false  "Filter by status"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /carousel [get]
func (h *CarouselHandler) GetCarousels(c *fiber.Ctx) error {
    // Parse query parameters
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "10"))
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
                id, image, title, description, status, created_at 
              FROM carousel 
              WHERE deleted_at IS NULL`
    args := []interface{}{}
    paramCounter := 1

    // Filter status
    if status != "" {
        statusBool, err := strconv.ParseBool(status)
        if err == nil {
            query += fmt.Sprintf(" AND status = $%d", paramCounter)
            args = append(args, statusBool)
            paramCounter++
        }
    }

    // Add pagination
    query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", paramCounter, paramCounter+1)
    args = append(args, limit, offset)

    // Eksekusi query
    rows, err := h.db.Query(context.Background(), query, args...)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch carousels",
        })
    }
    defer rows.Close()

    var carousels []models.CarouselResponse
    for rows.Next() {
        var carousel models.CarouselResponse
        err := rows.Scan(
            &carousel.ID,
            &carousel.Image,
            &carousel.Title,
            &carousel.Description,
            &carousel.Status,
            &carousel.CreatedAt,
        )
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to parse carousel data",
            })
        }
        carousels = append(carousels, carousel)
    }

    // Get total count
    countQuery := `SELECT COUNT(*) FROM carousel WHERE deleted_at IS NULL`
    countArgs := []interface{}{}
    paramCounter = 1

    if status != "" {
        statusBool, _ := strconv.ParseBool(status)
        countQuery += fmt.Sprintf(" AND status = $%d", paramCounter)
        countArgs = append(countArgs, statusBool)
    }

    var total int
    err = h.db.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to get total carousels",
        })
    }

    return c.JSON(fiber.Map{
        "data": carousels,
        "meta": fiber.Map{
            "page":       page,
            "limit":      limit,
            "total":      total,
            "totalPages": int(math.Ceil(float64(total) / float64(limit))),
        },
    })
}
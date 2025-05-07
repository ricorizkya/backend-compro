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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PortfolioHandler struct {
	db *pgxpool.Pool
}

func NewPortfolioHandler(db *pgxpool.Pool) *PortfolioHandler {
	return &PortfolioHandler{db: db}
}

// CreatePortfolioImage godoc
// @Summary      Add new portfolio image
// @Description  Upload and create new portfolio image
// @Tags         portfolio
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Portfolio image"
// @Security     ApiKeyAuth
// @Success      201  {object}  models.PortfolioImage
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portfolio [post]
func (h *PortfolioHandler) CreatePortfolioImage(c *fiber.Ctx) error {
	// Dapatkan user yang membuat
	userID := c.Locals("userID").(int)

	// Handle image upload
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image is required",
		})
	}

	// Validasi tipe file
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedTypes[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file type. Allowed: JPG, JPEG, PNG, WEBP",
		})
	}

	// Simpan gambar
	currentDir, _ := os.Getwd()
	uploadDir := filepath.Join(currentDir, "uploads/portfolio/images")
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	filename := fmt.Sprintf("%d-%s",
		time.Now().UnixNano(),
		filepath.Base(file.Filename),
	)
	filePath := filepath.Join(uploadDir, filename)

	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save image",
		})
	}

	// Simpan ke database
	query := `
        INSERT INTO portfolio_images (image, created_by)
        VALUES ($1, $2)
        RETURNING id, created_at
    `

	var portfolioImage models.PortfolioImage
	err = h.db.QueryRow(context.Background(), query,
		"uploads/portfolio/images/"+filename,
		userID,
	).Scan(&portfolioImage.ID, &portfolioImage.CreatedAt)

	if err != nil {
		// Hapus file yang sudah diupload jika gagal insert
		os.Remove(filePath)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create portfolio image: " + err.Error(),
		})
	}

	// Isi response
	portfolioImage.Image = "uploads/portfolio/images/" + filename
	portfolioImage.CreatedBy = userID

	return c.Status(fiber.StatusCreated).JSON(portfolioImage)
}

// UpdatePortfolioImage godoc
// @Summary      Update portfolio image
// @Description  Replace existing portfolio image with new one
// @Tags         portfolio
// @Accept       multipart/form-data
// @Produce      json
// @Param        id     path      int   true  "Portfolio Image ID"
// @Param        image  formData  file  true  "New portfolio image"
// @Security     ApiKeyAuth
// @Success      200  {object}  models.PortfolioImage
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portfolio/{id} [put]
func (h *PortfolioHandler) UpdatePortfolioImage(c *fiber.Ctx) error {
	// Dapatkan user yang melakukan update
	userID := c.Locals("userID").(int)

	// Parse ID
	imageID := c.Params("id")
	id, err := strconv.Atoi(imageID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid image ID format",
		})
	}

	// Handle image upload
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image is required",
		})
	}

	// Validasi tipe file
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedTypes[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file type. Allowed: JPG, JPEG, PNG, WEBP",
		})
	}

	// Dapatkan path gambar lama
	var oldImagePath string
	err = h.db.QueryRow(
		context.Background(),
		"SELECT image FROM portfolio_images WHERE id = $1 AND deleted_at IS NULL",
		id,
	).Scan(&oldImagePath)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Portfolio image not found",
		})
	}

	// Simpan gambar baru
	uploadDir := "uploads/portfolio/images"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	filename := fmt.Sprintf("%d-%s",
		time.Now().UnixNano(),
		filepath.Base(file.Filename),
	)
	filePath := filepath.Join(uploadDir, filename)

	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save new image",
		})
	}

	// Update database
	query := `
        UPDATE portfolio_images 
        SET 
            image = $1,
            edited_by = $2
        WHERE id = $3
        RETURNING id, image, created_at, edited_at
    `

	var updatedImage models.PortfolioImage
	err = h.db.QueryRow(
		context.Background(),
		query,
		"uploads/portfolio/images/"+filename,
		userID,
		id,
	).Scan(
		&updatedImage.ID,
		&updatedImage.Image,
		&updatedImage.CreatedAt,
		&updatedImage.EditedAt,
	)

	if err != nil {
		// Hapus gambar baru jika gagal update
		os.Remove(filePath)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update portfolio image: " + err.Error(),
		})
	}

	// Hapus gambar lama
	go func(oldPath string) {
		if oldPath != "" {
			fullPath := "" + oldPath // karena path disimpan sebagai "/uploads/..."
			if err := os.Remove(fullPath); err != nil {
				log.Printf("Gagal menghapus gambar lama: %s. Error: %v", oldPath, err)
			}
		}
	}(oldImagePath)

	updatedImage.CreatedBy = userID
	return c.JSON(updatedImage)
}

// DeletePortfolioImage godoc
// @Summary      Delete portfolio image (soft delete)
// @Description  Mark portfolio image as deleted and remove associated image file
// @Tags         portfolio
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Portfolio Image ID"
// @Security     ApiKeyAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portfolio/{id} [delete]
func (h *PortfolioHandler) DeletePortfolioImage(c *fiber.Ctx) error {
	// Dapatkan admin yang melakukan delete
	adminID := c.Locals("userID").(int)
	adminRole := c.Locals("userRole").(models.UserRole)

	if adminRole != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Parse ID
	imageID := c.Params("id")
	id, err := strconv.Atoi(imageID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid image ID format",
		})
	}

	// Dapatkan path gambar
	var imagePath string
	err = h.db.QueryRow(
		context.Background(),
		`SELECT image FROM portfolio_images 
         WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&imagePath)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Portfolio image not found",
		})
	}

	// Soft delete di database
	query := `
        UPDATE portfolio_images 
        SET 
            deleted_at = $1,
            deleted_by = $2
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
			"error": "Failed to delete portfolio image",
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Portfolio image not found",
		})
	}

	// Hapus file gambar
	if imagePath != "" {
		go func(path string) {
			fullPath := "" + path // karena path disimpan sebagai "/uploads/..."
			if err := os.Remove(fullPath); err != nil {
				log.Printf("Gagal menghapus gambar portfolio: %s. Error: %v", path, err)
			}
		}(imagePath)
	}

	return c.JSON(fiber.Map{
		"message": "Portfolio image deleted successfully",
	})
}

// GetPortfolioImages godoc
// @Summary      Get all portfolio images
// @Description  Get list of portfolio images with pagination
// @Tags         portfolio
// @Accept       json
// @Produce      json
// @Param        page    query     int     false  "Page number"     default(1)
// @Param        limit   query     int     false  "Items per page"  default(10)
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /portfolio [get]
func (h *PortfolioHandler) GetPortfolioImages(c *fiber.Ctx) error {
	// Parse query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	// Validasi input
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Query untuk mendapatkan data
	query := `SELECT 
                id, image, created_at, created_by 
              FROM portfolio_images 
              WHERE deleted_at IS NULL
              ORDER BY created_at DESC 
              LIMIT $1 OFFSET $2`

	rows, err := h.db.Query(context.Background(), query, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch portfolio images",
		})
	}
	defer rows.Close()

	var images []models.PortfolioImageResponse
	for rows.Next() {
		var img models.PortfolioImageResponse
		err := rows.Scan(
			&img.ID,
			&img.Image,
			&img.CreatedAt,
			&img.CreatedBy,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to parse image data",
			})
		}
		images = append(images, img)
	}

	// Query untuk total data
	var total int
	err = h.db.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM portfolio_images WHERE deleted_at IS NULL",
	).Scan(&total)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get total images",
		})
	}

	return c.JSON(fiber.Map{
		"data": images,
		"meta": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}

// GetPortfolioImagesByID godoc
// @Summary      Get PortfolioImages by ID
// @Description  Retrieve PortfolioImages details by PortfolioImages ID
// @Tags         PortfolioImages
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "PortfolioImages ID"
// @Security     ApiKeyAuth
// @Success      200  {object}  models.PortfolioImagesResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portofolio/image/{id} [get]
func (h *PortfolioHandler) GetPortfolioImageByID(c *fiber.Ctx) error {
	// Parse ID dari parameter URL
	PorfolioID := c.Params("id")
	id, err := strconv.Atoi(PorfolioID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid Portfolio Image ID format",
		})
	}

	// Query ke database
	query := `
        SELECT 
            id, image, created_at, created_by
        FROM portfolio_images
        WHERE id = $1 AND deleted_at IS NULL
    `

	var portfolio_images models.PortfolioImageResponse
	err = h.db.QueryRow(context.Background(), query, id).Scan(
		&portfolio_images.ID,
		&portfolio_images.Image,
		&portfolio_images.CreatedAt,
		&portfolio_images.CreatedBy,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Portfolio Image not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch Portfolio Image: " + err.Error(),
		})
	}

	return c.JSON(portfolio_images)
}

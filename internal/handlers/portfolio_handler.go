package handlers

import (
	"backend-go/internal/models"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// CreatePortfolioReview godoc
// @Summary      Create new portfolio review
// @Description  Add new portfolio review with optional product association
// @Tags         portfolio
// @Accept       multipart/form-data
// @Produce      json
// @Param        image        formData  file    false "Review image"
// @Param        product_id   formData  int     false "Associated product ID"
// @Param        title        formData  string  true  "Review title"
// @Param        description  formData  string  true  "Review description"
// @Param        date         formData  string  true  "Review date (YYYY-MM-DD)"
// @Security     ApiKeyAuth
// @Success      201  {object}  models.PortfolioReview
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portfolio/reviews [post]
func (h *PortfolioHandler) CreatePortfolioReview(c *fiber.Ctx) error {
	// Dapatkan user yang membuat
	userID := c.Locals("userID").(int)

	// Parse form data
	var req models.PortfolioReviewCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid form data",
		})
	}

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid date format. Use YYYY-MM-DD",
		})
	}

	// Handle image upload
	file, _ := c.FormFile("image")
	var imagePath string

	if file != nil {
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
		uploadDir := filepath.Join(currentDir, "uploads/portfolio/reviews")
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create upload directory",
			})
		}

		filename := fmt.Sprintf("%d-%s%s",
			time.Now().UnixNano(),
			strings.ReplaceAll(req.Title, " ", "-"),
			ext,
		)
		filePath := filepath.Join(uploadDir, filename)

		if err := c.SaveFile(file, filePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save image",
			})
		}
		imagePath = "uploads/portfolio/reviews/" + filename
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
        INSERT INTO portfolio_review (
            id_product,
            title,
            description,
            image,
            date,
            created_by
        ) VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at
    `

	var review models.PortfolioReview
	err = h.db.QueryRow(context.Background(), query,
		req.ProductID,
		req.Title,
		req.Description,
		imagePath,
		date,
		userID,
	).Scan(&review.ID, &review.CreatedAt)

	if err != nil {
		// Hapus gambar jika gagal insert
		if imagePath != "" {
			os.Remove("." + imagePath)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create portfolio review: " + err.Error(),
		})
	}

	// Isi response
	review.ProductID = req.ProductID
	review.Title = req.Title
	review.Description = req.Description
	review.Image = imagePath
	review.Date = date
	review.CreatedBy = userID

	return c.Status(fiber.StatusCreated).JSON(review)
}

// UpdatePortfolioReview godoc
// @Summary      Update portfolio review
// @Description  Update existing portfolio review data
// @Tags         portfolio
// @Accept       multipart/form-data
// @Produce      json
// @Param        id           path      int     true  "Review ID"
// @Param        image        formData  file    false "New review image"
// @Param        product_id   formData  int     false "Associated product ID"
// @Param        title        formData  string  false "Review title"
// @Param        description  formData  string  false "Review description"
// @Param        date         formData  string  false "Review date (YYYY-MM-DD)"
// @Security     ApiKeyAuth
// @Success      200  {object}  models.PortfolioReview
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portfolio/reviews/{id} [put]
func (h *PortfolioHandler) UpdatePortfolioReview(c *fiber.Ctx) error {
	// Dapatkan user yang melakukan update
	userID := c.Locals("userID").(int)

	// Parse ID
	reviewID := c.Params("id")
	id, err := strconv.Atoi(reviewID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid review ID format",
		})
	}

	// Cek apakah review ada
	var existingImage string
	var existingProductID *int
	err = h.db.QueryRow(
		context.Background(),
		`SELECT image, id_product FROM portfolio_review 
         WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&existingImage, &existingProductID)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Portfolio review not found",
		})
	}

	// Parse form data
	var req models.PortfolioReviewUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid form data",
		})
	}

	// Handle image upload
	file, _ := c.FormFile("image")
	var newImagePath string

	if file != nil {
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

		// Simpan gambar baru
		currentDir, _ := os.Getwd()
		uploadDir := filepath.Join(currentDir, "uploads/portfolio/reviews")
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create upload directory",
			})
		}

		filename := fmt.Sprintf("%d-%s%s",
			time.Now().UnixNano(),
			strings.ReplaceAll(req.Title, " ", "-"),
			ext,
		)
		filePath := filepath.Join(uploadDir, filename)

		if err := c.SaveFile(file, filePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save new image",
			})
		}
		newImagePath = "uploads/portfolio/reviews/" + filename

		// Hapus gambar lama
		go func(oldPath string) {
			if oldPath != "" {
				os.Remove("" + oldPath)
			}
		}(existingImage)
	}

	// Parse dan validasi date
	var date time.Time
	if req.Date != "" {
		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid date format. Use YYYY-MM-DD",
			})
		}
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
	query := `UPDATE portfolio_review SET
                id_product = COALESCE(NULLIF($1, 0), id_product),
                title = COALESCE(NULLIF($2, ''), title),
                description = COALESCE(NULLIF($3, ''), description),
                image = COALESCE(NULLIF($4, ''), image),
                date = COALESCE($5, date),
                edited_by = $6
              WHERE id = $7
              RETURNING *`

	args := []interface{}{
		req.ProductID,
		req.Title,
		req.Description,
		newImagePath,
		date,
		userID,
		id,
	}

	var review models.PortfolioReview
	err = h.db.QueryRow(
		context.Background(),
		query,
		args...,
	).Scan(
		&review.ID,
		&review.ProductID,
		&review.Title,
		&review.Description,
		&review.Image,
		&review.Date,
		&review.CreatedAt,
		&review.CreatedBy,
		&review.EditedAt,
		&review.EditedBy,
		&review.DeletedAt,
		&review.DeletedBy,
	)

	if err != nil {
		if newImagePath != "" {
			os.Remove("." + newImagePath)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update portfolio review: " + err.Error(),
		})
	}

	return c.JSON(review)
}

// DeletePortfolioReview godoc
// @Summary      Delete portfolio review
// @Description  Soft delete a portfolio review by marking it as deleted
// @Tags         portfolio
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Review ID"
// @Security     ApiKeyAuth
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portfolio/reviews/{id} [delete]
func (h *PortfolioHandler) DeletePortfolioReview(c *fiber.Ctx) error {
	// Dapatkan user yang melakukan delete
	userID := c.Locals("userID").(int)

	// Parse ID
	reviewID := c.Params("id")
	id, err := strconv.Atoi(reviewID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid review ID format",
		})
	}

	// Lakukan soft delete
	query := `
        UPDATE portfolio_review 
        SET deleted_at = $1, 
            deleted_by = $2 
        WHERE id = $3 
            AND deleted_at IS NULL
    `

	result, err := h.db.Exec(
		context.Background(),
		query,
		time.Now(),
		userID,
		id,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete portfolio review: " + err.Error(),
		})
	}

	// Cek apakah data benar-benar terupdate
	if rowsAffected := result.RowsAffected(); rowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Portfolio review not found or already deleted",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetPortfolioReviews godoc
// @Summary      Get all portfolio reviews
// @Description  Retrieve all active portfolio reviews with optional product info
// @Tags         portfolio
// @Produce      json
// @Success      200  {array}  handlers.PortfolioReviewWithProduct
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portfolio/reviews [get]
func (h *PortfolioHandler) GetPortfolioReviews(c *fiber.Ctx) error {
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

	query := `
        SELECT 
            pr.id,
            pr.id_product,
            pr.title,
            pr.description,
            pr.image,
            pr.date,
            pr.created_at,
            pr.created_by,
            pr.edited_at,
            pr.edited_by,
            p.title as product_name,
            p.image as product_image
        FROM portfolio_review pr
        LEFT JOIN products p ON pr.id_product = p.id
        WHERE pr.deleted_at IS NULL
        ORDER BY pr.date DESC
        LIMIT $1 OFFSET $2
    `

	rows, err := h.db.Query(context.Background(), query, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch portfolio reviews: " + err.Error(),
		})
	}
	defer rows.Close()

	var reviews []models.PortfolioReviewWithProduct
	for rows.Next() {
		var review models.PortfolioReviewWithProduct
		err := rows.Scan(
			&review.ID,
			&review.ProductID,
			&review.Title,
			&review.Description,
			&review.Image,
			&review.Date,
			&review.CreatedAt,
			&review.CreatedBy,
			&review.EditedAt,
			&review.EditedBy,
			&review.ProductName,
			&review.ProductImage,
		)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to parse portfolio reviews: " + err.Error(),
			})
		}
		reviews = append(reviews, review)
	}

	if len(reviews) == 0 {
		return c.Status(fiber.StatusOK).JSON([]interface{}{})
	}

	// Query untuk total data
	var total int
	err = h.db.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM portfolio_review WHERE deleted_at IS NULL",
	).Scan(&total)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get total data",
		})
	}

	return c.JSON(fiber.Map{
		"data": reviews,
		"meta": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}

// GetPortfolioReviewByID godoc
// @Summary      Get portfolio review by ID
// @Description  Retrieve a single portfolio review with product details
// @Tags         portfolio
// @Produce      json
// @Param        id   path      int  true  "Portfolio Review ID"
// @Success      200  {object}  models.PortfolioReviewDetail
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /portfolio/reviews/{id} [get]
func (h *PortfolioHandler) GetPortfolioReviewByID(c *fiber.Ctx) error {
	// Parse ID dari parameter URL
	reviewID := c.Params("id")
	id, err := strconv.Atoi(reviewID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid review ID format",
		})
	}

	query := `
        SELECT 
            pr.id,
            pr.id_product,
            pr.title,
            pr.description,
            pr.image,
            pr.date,
            pr.created_at,
            pr.created_by,
            pr.edited_at,
            pr.edited_by,
            p.title as product_name,
            p.image as product_image
        FROM portfolio_review pr
        LEFT JOIN products p ON pr.id_product = p.id
        WHERE pr.id = $1 AND pr.deleted_at IS NULL
    `

	var review models.PortfolioReviewWithProduct
	err = h.db.QueryRow(context.Background(), query, id).Scan(
		&review.ID,
		&review.ProductID,
		&review.Title,
		&review.Description,
		&review.Image,
		&review.Date,
		&review.CreatedAt,
		&review.CreatedBy,
		&review.EditedAt,
		&review.EditedBy,
		&review.ProductName,
		&review.ProductImage,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Portfolio review not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch portfolio review: " + err.Error(),
		})
	}

	return c.JSON(review)
}

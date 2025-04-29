package handlers

import (
	"backend-go/internal/models"
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	uploadPath := "../uploads/carousel/"
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
        "../uploads/carousel/"+filename,
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

	carousel.Image = "../uploads/carousel/" + filename
    carousel.Title = req.Title
    carousel.Description = req.Description
    carousel.Status = req.Status
	carousel.CreatedBy = &userID

    return c.Status(fiber.StatusCreated).JSON(carousel)
}
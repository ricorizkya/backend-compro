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
	"github.com/shopspring/decimal"
)

type ProductHandler struct {
    db *pgxpool.Pool
}

func NewProductHandler(db *pgxpool.Pool) *ProductHandler {
    return &ProductHandler{db: db}
}

// CreateProduct godoc
// @Summary      Create new product
// @Description  Add new product item
// @Tags         products
// @Accept       multipart/form-data
// @Produce      json
// @Param        image        formData  file    true  "Product image"
// @Param        title        formData  string  true  "Product title"
// @Param        description  formData  string  false "Product description"
// @Param        type_product formData  string  true  "Product type (physical/digital/service)"
// @Param        price        formData  string  true  "Product price (format: 100.00)"
// @Param        status       formData  bool    false "Product status"
// @Security     ApiKeyAuth
// @Success      201  {object}  models.Product
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /products [post]
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
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

    // Parse form data
    var req models.ProductCreateRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid form data",
        })
    }

    // Konversi price ke decimal
    price, err := decimal.NewFromString(req.Price)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid price format",
        })
    }

    // Validasi price >= 0
    if price.LessThan(decimal.Zero) {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Price cannot be negative",
        })
    }

    // Simpan gambar
    uploadDir := "./uploads/products"
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

    // Simpan ke database
    query := `
        INSERT INTO products (
            image,
            title,
            description,
            type_product,
            price,
            status,
            created_by
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at
    `

    var product models.Product
    err = h.db.QueryRow(context.Background(), query,
        "/uploads/products/"+filename,
        req.Title,
        req.Description,
        req.TypeProduct,
        price,
        req.Status,
        userID,
    ).Scan(&product.ID, &product.CreatedAt)

    if err != nil {
        // Hapus file yang sudah diupload jika gagal insert
        os.Remove(filePath)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create product: " + err.Error(),
        })
    }

    // Isi response
    product.Image = "/uploads/products/" + filename
    product.Title = req.Title
    product.Description = req.Description
    product.TypeProduct = req.TypeProduct
    product.Price, _ = price.Float64()
    product.Status = req.Status
    product.CreatedBy = userID

    return c.Status(fiber.StatusCreated).JSON(product)
}

// UpdateProduct godoc
// @Summary      Update product
// @Description  Update existing product data
// @Tags         products
// @Accept       multipart/form-data
// @Produce      json
// @Param        id           path      int     true  "Product ID"
// @Param        image        formData  file    false "New product image"
// @Param        title        formData  string  false "Product title"
// @Param        description  formData  string  false "Product description"
// @Param        type_product formData  string  false "Product type (physical/digital/service)"
// @Param        price        formData  string  false "Product price (format: 100.00)"
// @Param        status       formData  bool    false "Product status"
// @Security     ApiKeyAuth
// @Success      200  {object}  models.Product
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
    productID := c.Params("id")
    id, err := strconv.Atoi(productID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid product ID format",
        })
    }

    // Dapatkan user yang melakukan update
    userID := c.Locals("userID").(int)
    
    // Cek apakah product ada
    var existingImage string
    err = h.db.QueryRow(context.Background(),
        `SELECT image FROM products 
         WHERE id = $1 AND deleted_at IS NULL`,
        id,
    ).Scan(&existingImage)

    if err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Product not found",
        })
    }

    // Parse form data
    var req models.ProductUpdateRequest
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

        // Upload new image
        uploadDir := "./uploads/products"
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
        newImagePath = "/uploads/products/" + filename

        // Delete old image
        go func(oldImage string) {
            if oldImage != "" {
                os.Remove("." + oldImage)
            }
        }(existingImage)
    }

    // Konversi price
    var price decimal.Decimal
    if req.Price != "" {
        price, err = decimal.NewFromString(req.Price)
        if err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid price format",
            })
        }
        if price.LessThan(decimal.Zero) {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Price cannot be negative",
            })
        }
    }

    // Build dynamic query
    query := `UPDATE products SET
				image = COALESCE(NULLIF($1, ''), image),
				title = COALESCE(NULLIF($2, ''), title),
				description = COALESCE(NULLIF($3, ''), description),
				type_product = CASE 
					WHEN $4::text = '' THEN type_product 
					ELSE $4::product_type 
				END,
				price = COALESCE(NULLIF($5, 0), price),
				status = COALESCE($6, status),
				edited_by = $7
			WHERE id = $8
			RETURNING *`

	args := []interface{}{
		newImagePath,
		req.Title,
		req.Description,
		req.TypeProduct, // Pastikan ini string kosong jika tidak diupdate
		price,
		req.Status,
		userID,
		id,
	}

    var product models.Product
    var priceDB decimal.Decimal
    err = h.db.QueryRow(context.Background(), query, args...).Scan(
        &product.ID,
        &product.Image,
        &product.Title,
        &product.Description,
        &product.TypeProduct,
        &priceDB,
        &product.Status,
        &product.CreatedAt,
        &product.CreatedBy,
        &product.EditedAt,
        &product.EditedBy,
        &product.DeletedAt,
        &product.DeletedBy,
    )

    if err != nil {
        if newImagePath != "" {
            os.Remove("." + newImagePath)
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to update product: " + err.Error(),
        })
    }

    // Konversi decimal ke float untuk response
    product.Price, _ = priceDB.Float64()
    
    return c.JSON(product)
}

// DeleteProduct godoc
// @Summary      Delete product (soft delete)
// @Description  Mark product as deleted and remove associated image
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Product ID"
// @Security     ApiKeyAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
    productID := c.Params("id")
    id, err := strconv.Atoi(productID)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid product ID format",
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
        `SELECT image FROM products 
         WHERE id = $1 AND deleted_at IS NULL`,
        id,
    ).Scan(&imagePath)

    if err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Product not found or already deleted",
        })
    }

    // Soft delete di database
    query := `
        UPDATE products 
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
            "error": "Failed to delete product",
        })
    }

    if result.RowsAffected() == 0 {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Product not found",
        })
    }

    // Hapus file gambar
    if imagePath != "" {
        go func(path string) {
            fullPath := "." + path // karena path disimpan sebagai "/uploads/..."
            if err := os.Remove(fullPath); err != nil {
                log.Printf("Failed to delete product image: %s. Error: %v", path, err)
            }
        }(imagePath)
    }

    return c.JSON(fiber.Map{
        "message": "Product deleted successfully",
    })
}

// GetProducts godoc
// @Summary      Get all products
// @Description  Get list of products with pagination and filters
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        page     query     int     false  "Page number"     default(1)
// @Param        limit    query     int     false  "Items per page"  default(10)
// @Param        status   query     bool    false  "Filter by status"
// @Param        type     query     string  false  "Filter by product type"
// @Param        minPrice query     number  false  "Minimum price"
// @Param        maxPrice query     number  false  "Maximum price"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /products [get]
func (h *ProductHandler) GetProducts(c *fiber.Ctx) error {
    // Parse query parameters
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "10"))
    status := c.Query("status")
    productType := c.Query("type")
    minPrice := c.Query("minPrice")
    maxPrice := c.Query("maxPrice")

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
                id, image, title, description, 
                type_product, price, status, created_at 
              FROM products 
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

    // Filter type
    if productType != "" {
        query += fmt.Sprintf(" AND type_product = $%d", paramCounter)
        args = append(args, productType)
        paramCounter++
    }

    // Filter harga
    if minPrice != "" {
        query += fmt.Sprintf(" AND price >= $%d", paramCounter)
        args = append(args, minPrice)
        paramCounter++
    }
    if maxPrice != "" {
        query += fmt.Sprintf(" AND price <= $%d", paramCounter)
        args = append(args, maxPrice)
        paramCounter++
    }

    // Add pagination
    query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", paramCounter, paramCounter+1)
    args = append(args, limit, offset)

    // Eksekusi query
    rows, err := h.db.Query(context.Background(), query, args...)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch products",
        })
    }
    defer rows.Close()

    var products []models.ProductResponse
    for rows.Next() {
        var product models.ProductResponse
        var price decimal.Decimal
        
        err := rows.Scan(
            &product.ID,
            &product.Image,
            &product.Title,
            &product.Description,
            &product.TypeProduct,
            &price,
            &product.Status,
            &product.CreatedAt,
        )
        
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to parse product data",
            })
        }
        
        product.Price, _ = price.Float64()
        products = append(products, product)
    }

    // Get total count
    countQuery := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`
    countArgs := []interface{}{}
    paramCounter = 1

    if status != "" {
        statusBool, _ := strconv.ParseBool(status)
        countQuery += fmt.Sprintf(" AND status = $%d", paramCounter)
        countArgs = append(countArgs, statusBool)
        paramCounter++
    }
    
    if productType != "" {
        countQuery += fmt.Sprintf(" AND type_product = $%d", paramCounter)
        countArgs = append(countArgs, productType)
        paramCounter++
    }
    
    if minPrice != "" {
        countQuery += fmt.Sprintf(" AND price >= $%d", paramCounter)
        countArgs = append(countArgs, minPrice)
        paramCounter++
    }
    
    if maxPrice != "" {
        countQuery += fmt.Sprintf(" AND price <= $%d", paramCounter)
        countArgs = append(countArgs, maxPrice)
        paramCounter++
    }

    var total int
    err = h.db.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to get total products",
        })
    }

    return c.JSON(fiber.Map{
        "data": products,
        "meta": fiber.Map{
            "page":       page,
            "limit":      limit,
            "total":      total,
            "totalPages": int(math.Ceil(float64(total) / float64(limit))),
        },
    })
}
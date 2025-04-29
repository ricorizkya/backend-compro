package models

import (
	"time"
)

// Buat custom type untuk enum product_type
type ProductType string

const (
    ProductTypePhysical ProductType = "physical"
    ProductTypeDigital  ProductType = "digital"
    ProductTypeService  ProductType = "service"
)

type Product struct {
    ID           int          `json:"id"`
    Image        string       `json:"image" validate:"required,url"`
    Title        string       `json:"title" validate:"required,max=100"`
    Description  string       `json:"description,omitempty"`
    TypeProduct  ProductType  `json:"type_product" validate:"required,oneof=physical digital service"`
    Price        float64      `json:"price" validate:"required,min=0"`
    Status       bool         `json:"status"`
    CreatedAt    time.Time    `json:"created_at"`
    CreatedBy    int          `json:"created_by"`
    EditedAt     *time.Time   `json:"edited_at,omitempty"`
    EditedBy     *int         `json:"edited_by,omitempty"`
    DeletedAt    *time.Time   `json:"deleted_at,omitempty"`
    DeletedBy    *int         `json:"deleted_by,omitempty"`
}

type ProductCreateRequest struct {
    Title        string      `form:"title" validate:"required,max=100"`
    Description  string      `form:"description,omitempty"`
    TypeProduct  ProductType `form:"type_product" validate:"required,oneof=physical digital service"`
    Price        string      `form:"price" validate:"required,decimal=2"`
    Status       bool        `form:"status"`
}

type ProductUpdateRequest struct {
    Title        string       `form:"title,omitempty" validate:"max=100"`
    Description  string       `form:"description,omitempty"`
    TypeProduct  ProductType  `form:"type_product,omitempty" validate:"omitempty,oneof=physical digital service"`
    Price        string       `form:"price,omitempty" validate:"omitempty,decimal=2"`
    Status       *bool        `form:"status,omitempty"`
}

type ProductResponse struct {
    ID           int           `json:"id"`
    Image        string        `json:"image"`
    Title        string        `json:"title"`
    Description  string        `json:"description,omitempty"`
    TypeProduct  ProductType   `json:"type_product"`
    Price        float64       `json:"price"`
    Status       bool          `json:"status"`
    CreatedAt    time.Time     `json:"created_at"`
}
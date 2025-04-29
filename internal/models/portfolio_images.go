package models

import (
	"time"
)

type PortfolioImage struct {
    ID         int        `json:"id"`
    Image      string     `json:"image" validate:"required"`
    CreatedAt  time.Time  `json:"created_at"`
    CreatedBy  int        `json:"created_by"`
    EditedAt   *time.Time `json:"edited_at,omitempty"`
    EditedBy   *int       `json:"edited_by,omitempty"`
    DeletedAt  *time.Time `json:"deleted_at,omitempty"`
    DeletedBy  *int       `json:"deleted_by,omitempty"`
}

type PortfolioImageCreateRequest struct {
    // Tidak perlu field tambahan karena hanya menerima file gambar
}

type PortfolioImageResponse struct {
    ID         int       `json:"id"`
    Image      string    `json:"image"`
    CreatedAt  time.Time `json:"created_at"`
    CreatedBy  int       `json:"created_by"`
}
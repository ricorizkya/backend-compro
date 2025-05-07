package models

import "time"

type PortfolioReview struct {
	ID          int        `json:"id"`
	ProductID   *int       `json:"product_id,omitempty"`
	Title       string     `json:"title" validate:"required,max=100"`
	Description string     `json:"description" validate:"required"`
	Image       string     `json:"image,omitempty"`
	Date        time.Time  `json:"date" validate:"required"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   int        `json:"created_by"`
	EditedAt    *time.Time `json:"edited_at,omitempty"`
	EditedBy    *int       `json:"edited_by,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	DeletedBy   *int       `json:"deleted_by,omitempty"`
}

type PortfolioReviewCreateRequest struct {
	ProductID   *int   `form:"product_id,omitempty"`
	Title       string `form:"title" validate:"required,max=100"`
	Description string `form:"description" validate:"required"`
	Date        string `form:"date" validate:"required,datetime=2006-01-02"`
}

type PortfolioReviewUpdateRequest struct {
	ProductID   *int   `form:"product_id,omitempty"`
	Title       string `form:"title,omitempty" validate:"max=100"`
	Description string `form:"description,omitempty"`
	Date        string `form:"date,omitempty" validate:"omitempty,datetime=2006-01-02"`
}

type PortfolioReviewWithProduct struct {
	PortfolioReview
	ProductName  *string `json:"product_name,omitempty"`
	ProductImage *string `json:"product_image,omitempty"`
}

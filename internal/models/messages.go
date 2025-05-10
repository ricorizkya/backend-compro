package models

import "time"

type Message struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Company      string     `json:"company,omitempty"`
	Address      string     `json:"address,omitempty"`
	Description  string     `json:"description"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    int        `json:"created_by"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	EditedBy     *int       `json:"edited_by,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	DeletedBy    *int       `json:"deleted_by,omitempty"`
	ProductID    *int       `json:"product_id,omitempty"`
	DateSchedule *time.Time `json:"date_schedule"`
	Phone        string     `json:"phone"`
}

type MessageCreateRequest struct {
	Name         string     `form:"name"`
	Company      string     `form:"company"`
	ProductID    *int       `form:"product_id"`
	Address      string     `form:"address"`
	Description  string     `form:"description"`
	DateSchedule *time.Time `json:"date_schedule"`
	Phone        string     `json:"phone"`
}

type MessageUpdateRequest struct {
	Name         *string    `form:"name"`
	Company      *string    `form:"company"`
	ProductID    *int       `form:"product_id"`
	Address      *string    `form:"address"`
	Description  *string    `form:"description"`
	DateSchedule *time.Time `json:"date_schedule,omitempty"`
	Phone        string     `json:"phone"`
}

type MessageWithProduct struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Company      string     `json:"company,omitempty"`
	ProductID    *int       `json:"product_id,omitempty"`
	Address      string     `json:"address,omitempty"`
	Description  string     `json:"description"`
	DateSchedule *time.Time `json:"date_schedule"`
	Phone        string     `json:"phone"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    int        `json:"created_by"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	ProductName  *string    `json:"product_name,omitempty"`
	ProductImage *string    `json:"product_image,omitempty"`
}

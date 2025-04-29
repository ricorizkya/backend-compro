package models

import (
	"time"
)

type Carousel struct {
	ID        	int       `json:"id"`
	Image	 	string    `json:"image" validate:"required,url"`
	Title	 	string    `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
	Status		bool      `json:"status"`
	CreatedAt 	time.Time `json:"created_at"`
	CreatedBy 	*int      `json:"created_by"`
	EditedAt  	*time.Time `json:"edited_at"`
	EditedBy  	*int      `json:"edited_by"`
	DeletedAt 	*time.Time `json:"deleted_at"`
	DeletedBy 	*int      `json:"deleted_by"`
}

type CarouselCreateRequest struct {
	Image       string `json:"image" validate:"required,url"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
	Status	  	bool   `json:"status" validate:"required"`
}

type CarouselUpdateRequest struct {
    Title       string `form:"title,omitempty" validate:"max=100"`
    Description string `form:"description,omitempty"`
    Status      *bool  `form:"status,omitempty"`
}

type CarouselResponse struct {
    ID          int        `json:"id"`
    Image       string     `json:"image"`
    Title       string     `json:"title"`
    Description string     `json:"description,omitempty"`
    Status      bool       `json:"status"`
    CreatedAt   time.Time  `json:"created_at"`
}

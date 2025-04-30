package models

import (
	"time"
)

// UserRole tipe enum untuk role user
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleStaff  UserRole = "staff"
	RoleUser   UserRole = "user"
)

// User struktur data untuk tabel users
type User struct {
	ID         int        `json:"id"`
	Name       string     `json:"name" validate:"required,min=3"`
	Phone      string     `json:"phone" validate:"required,e164"`
	Username   string     `json:"username" validate:"required,alphanum"`
	Password   string     `json:"password" validate:"required,min=8"`
	Role       UserRole   `json:"role" validate:"omitempty,oneof=admin staff user"`
	Status     bool       `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	CreatedBy  *int       `json:"created_by"`
	EditedAt   *time.Time `json:"edited_at"`
	EditedBy   *int       `json:"edited_by"`
	DeletedAt  *time.Time `json:"deleted_at"`
	DeletedBy  *int       `json:"deleted_by"`
}

// CreateRequest struktur untuk input create user
type CreateRequest struct {
    Name     string   `json:"name" validate:"required,min=3"`
    Phone    string   `json:"phone" validate:"required,e164"`
    Username string   `json:"username" validate:"required,alphanum"`
    Password string   `json:"password" validate:"required,min=8"`
    Role     UserRole `json:"role,omitempty"`
}

// UpdateRequest struktur untuk input update user
type UpdateRequest struct {
    Name     string   `json:"name,omitempty"`
    Phone    string   `json:"phone,omitempty"`
    Username string   `json:"username,omitempty"`
    Password string   `json:"password,omitempty"`
    Role     UserRole `json:"role,omitempty"`
    Status   *bool    `json:"status,omitempty"`
}

// UserResponse struktur untuk output user
type UserResponse struct {
    ID        int        `json:"id"`
    Name      string     `json:"name"`
    Phone     string     `json:"phone"`
    Username  string     `json:"username"`
    Role      UserRole   `json:"role"`
    Status    bool       `json:"status"`
    CreatedAt time.Time  `json:"created_at"`
    CreatedBy *int       `json:"created_by"`
    EditedAt  *time.Time `json:"edited_at"`
    EditedBy  *int       `json:"edited_by"`
}

type RegisterRequest struct {
    Name     string `json:"name" validate:"required,min=3,max=100"`
    Phone    string `json:"phone" validate:"required,numeric,min=10,max=15"`
    Username string `json:"username" validate:"required,min=5,max=50"`
    Password string `json:"password" validate:"required,min=8,max=72"`
    Role    string `json:"role" validate:"required,oneof=admin staff user"`
}
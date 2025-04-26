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
    Name      string   `json:"name" validate:"required,min=3"`
    Phone     string   `json:"phone" validate:"required,e164"`
    Username  string   `json:"username" validate:"required,alphanum"`
    Password  string   `json:"password" validate:"required,min=8"`
    Role      UserRole `json:"role" validate:"omitempty,oneof=admin staff user"`
    CreatedBy *int      `json:"created_by"`
    Status   *bool    `json:"status,omitempty"`
}

type UpdateRequest struct {
    Name     string   `json:"name,omitempty"`
    Phone    string   `json:"phone,omitempty"`
    Username string   `json:"username,omitempty"`
    Password string   `json:"password,omitempty"`
    Role     UserRole `json:"role,omitempty"`
    Status   *bool    `json:"status,omitempty"` // Pointer untuk handle null
}
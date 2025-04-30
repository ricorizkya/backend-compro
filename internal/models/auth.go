package models

import "github.com/golang-jwt/jwt/v5"

type LoginRequest struct {
    Username string `json:"username" validate:"required"`
    Password string `json:"password" validate:"required,min=8"`
}

type Claims struct {
    UserID int      `json:"user_id"`
    Role   UserRole `json:"role"`
    jwt.RegisteredClaims
}

type UserLoginResponse struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Name     string `json:"name"`   
    Role     string `json:"role"`
}
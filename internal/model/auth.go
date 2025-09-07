package model

import (
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/database"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type JWTClaims struct {
	UserID    primitive.ObjectID  `json:"userId"`
	CompanyID *primitive.ObjectID `json:"companyId,omitempty"`
	jwt.RegisteredClaims
}

// Authentication middleware struct
type AuthContext struct {
	UserID    primitive.ObjectID
	CompanyID primitive.ObjectID
	User      *database.User
}
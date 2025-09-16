package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"renotech.com.my/internal/database"
)

type UserRegistrationModal struct {
	User    UserRegistrationData    `json:"user" binding:"required"`
	Company CompanyRegistrationData `json:"company" binding:"required"`
}

type UserRegistrationData struct {
	Username       string `json:"username" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	Contact        string `json:"contact" binding:"required"`
	ProfilePicture string `json:"profilePicture"`
}

type CompanyRegistrationData struct {
	Name                string `json:"name" binding:"required"`
	ClientDisplayName   string `json:"clientDisplayName"`
	SupplierDisplayName string `json:"supplierDisplayName"`
	Address             string `json:"address"`
	Website             string `json:"website"`
	Logo                string `json:"logo"`
	Contact             string `json:"contact"`
	PaymentDetail       string `json:"paymentDetail"`
}

type UserAddModal struct {
	Username       string `json:"username" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	Contact        string `json:"contact" binding:"required"`
	ProfilePicture string `json:"profilePicture"`
	Comment        string `json:"comment"`
}

type UserLoginModal struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// User service structs
type UserCreateResponse struct {
	Token string        `json:"token"`
	User  database.User `json:"user"`
}

// User CRUD request/response models
type UserListRequest struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	Sort      bson.M `json:"sort"`
	Search    string `json:"search"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Contact   string `json:"contact"`
	Company   string `json:"company"`
	Type      string `json:"type"`
	IsEnabled *bool  `json:"isEnabled"`
}

type UserListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}

// Password reset request/response models
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

type ResetPasswordRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// Change password request model (for authenticated users changing their password)
type ChangePasswordRequest struct {
	OldPassword     string `json:"oldPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

type ChangePasswordResponse struct {
	Message string `json:"message"`
}

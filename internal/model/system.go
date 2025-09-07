package model

import (
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
)

type SystemContext struct {
	MongoDB   *mongo.Database
	Logger    *zap.Logger
	User      database.User
	RequestID string
}

// AppError represents a structured application error
type AppError struct {
	Code    enum.ErrorCode `json:"code"`
	Message string         `json:"message"`
	Details interface{}    `json:"details,omitempty"`
}

type ResponseError struct {
	Status string    `json:"status"`
	Error  *AppError `json:"error"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

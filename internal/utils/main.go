package utils

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/logs"
)

// SystemError creates a new application error
func SystemError(code enum.ErrorCode, message string, details interface{}) *model.AppError {
	return &model.AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// SendErrorResponse sends a structured error response
func SendErrorResponse(c *gin.Context, err interface{}) {
	var appErr *model.AppError
	var statusCode int

	// Handle different error types
	switch e := err.(type) {
	case *model.AppError:
		appErr = e
	case error:
		// Don't expose internal errors to clients
		appErr = SystemError(enum.ErrorCodeInternal, "An internal error occurred", nil)
	case string:
		appErr = SystemError(enum.ErrorCodeBadRequest, e, nil)
	default:
		appErr = SystemError(enum.ErrorCodeInternal, "An unexpected error occurred", nil)
	}

	// Map error codes to HTTP status codes
	switch appErr.Code {
	case enum.ErrorCodeValidation, enum.ErrorCodeBadRequest:
		statusCode = http.StatusBadRequest
	case enum.ErrorCodeNotFound:
		statusCode = http.StatusNotFound
	case enum.ErrorCodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case enum.ErrorCodeTooLarge:
		statusCode = http.StatusRequestEntityTooLarge
	default:
		statusCode = http.StatusInternalServerError
	}

	errResponse := model.ResponseError{
		Status: "error",
		Error:  appErr,
	}

	c.JSON(statusCode, errResponse)
}

// SendSuccessResponse sends a structured success response
func SendSuccessResponse(c *gin.Context, data interface{}, options ...interface{}) {
	response := gin.H{
		"status": "success",
		"data":   data,
	}

	// Handle optional parameters
	for _, option := range options {
		switch v := option.(type) {
		case int64:
			response["total"] = v
		case string:
			response["message"] = v
		case *string:
			if v != nil {
				response["message"] = *v
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// SendSuccessMessageResponse sends a success response with just a message
func SendSuccessMessageResponse(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": message,
	})
}

func RemoveExtension(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

func SystemContextBaseInit() *model.SystemContext {
	logger := logs.LoggerGet()
	mongoDB := MongoGet()

	return &model.SystemContext{
		Logger:    logger,
		MongoDB:   mongoDB,
		RequestID: GenerateRequestID(),
	}
}

// SystemContextWithRequestID creates a system context with a specific request ID
func SystemContextWithRequestID(requestID string) *model.SystemContext {
	logger := logs.LoggerGet()
	mongoDB := MongoGet()

	return &model.SystemContext{
		Logger:    logger,
		MongoDB:   mongoDB,
		RequestID: requestID,
	}
}

func ConvertToFullPath(input string) string {
	projectRoot, _ := filepath.Abs(filepath.Join())
	fullPath := filepath.Join(projectRoot, input)
	return fullPath
}

// GenerateRequestID generates a unique request ID for tracing
func GenerateRequestID() string {
	// Simple implementation - in production, consider using UUID
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// ValidateObjectID validates and converts string ID to ObjectID
func ValidateObjectID(idStr string) (primitive.ObjectID, error) {
	if idStr == "" {
		return primitive.NilObjectID, SystemError(enum.ErrorCodeValidation, "ID is required", nil)
	}

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return primitive.NilObjectID, SystemError(enum.ErrorCodeValidation, "Invalid ID format", map[string]interface{}{"id": idStr})
	}

	return id, nil
}

// ValidateListParameters validates and sanitizes list query parameters
func ValidateListParameters(pageStr, limitStr string) (int, int, error) {
	page := 1
	limit := 10

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p >= 1 {
			page = p
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l >= 1 && l <= 100 {
			limit = l
		}
	}

	return page, limit, nil
}

// FormatPriceString converts interface{} value to formatted price string with specified decimal places
func FormatPriceString(value interface{}, decimals int) string {
	var floatVal float64
	var err error

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return ""
		}
		floatVal, err = strconv.ParseFloat(v, 64)
	case float64:
		floatVal = v
	case float32:
		floatVal = float64(v)
	case int:
		floatVal = float64(v)
	case int64:
		floatVal = float64(v)
	default:
		// If we can't convert, return the original value as string
		return fmt.Sprintf("%v", value)
	}

	if err != nil {
		// If parsing failed, return original value as string
		return fmt.Sprintf("%v", value)
	}

	// Format with specified decimal places
	formatStr := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(formatStr, floatVal)
}

// sanitizeFilename removes dangerous characters from filename
func SanitizeFilename(filename string) string {
	// Remove path separators and other dangerous characters
	dangerousChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename
	for _, char := range dangerousChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}
	return sanitized
}

// getEnvString gets string environment variable with default
func GetEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets integer environment variable with default
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetSystemContextFromGin retrieves SystemContext from Gin context with fallback
func GetSystemContextFromGin(c *gin.Context) *model.SystemContext {
	// First try to get enriched SystemContext from Gin context (set by JWT middleware)
	if systemContext, exists := c.Get("SystemContext"); exists {
		return systemContext.(*model.SystemContext)
	}
	
	// Fallback: create new SystemContext with RequestID if not found
	requestID, _ := c.Get("RequestID")
	if requestID == nil {
		return SystemContextBaseInit()
	}
	
	return SystemContextWithRequestID(requestID.(string))
}

// ValidateFilePath validates that a file path exists in the system
func ValidateFilePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil 
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return SystemError(
			enum.ErrorCodeValidation,
			"File path does not exist",
			map[string]interface{}{"path": path},
		)
	}

	return nil
}

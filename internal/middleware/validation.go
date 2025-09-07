package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/utils"
)

// RequestValidationMiddleware validates common request properties
func RequestValidationMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Validate Content-Type for POST, PUT, PATCH requests
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")

			// For file uploads, allow multipart/form-data
			if strings.Contains(c.Request.URL.Path, "/upload") {
				if !strings.Contains(contentType, "multipart/form-data") {
					c.JSON(http.StatusBadRequest, gin.H{
						"status": "error",
						"error": utils.SystemError(
							enum.ErrorCodeValidation,
							"Content-Type must be multipart/form-data for file uploads",
							nil,
						),
					})
					c.Abort()
					return
				}
			} else {
				// For other endpoints, expect JSON
				if !strings.Contains(contentType, "application/json") && contentType != "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"status": "error",
						"error": utils.SystemError(
							enum.ErrorCodeValidation,
							"Content-Type must be application/json",
							nil,
						),
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	})
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Check if request ID is already provided in headers
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = utils.GenerateRequestID()
		}

		// Add request ID to context and response headers
		c.Set("RequestID", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	})
}

// JSONResponseMiddleware ensures consistent JSON response format
func JSONResponseMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Set JSON content type for API responses
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Content-Type", "application/json")
		}

		c.Next()
	})
}

// PathValidationMiddleware validates request paths for security
func PathValidationMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Check for common attack patterns
		dangerousPatterns := []string{
			"../",
			"..\\",
			"/etc/",
			"/proc/",
			"/sys/",
			"\\windows\\",
			"\\system32\\",
		}

		for _, pattern := range dangerousPatterns {
			if strings.Contains(strings.ToLower(path), pattern) {
				c.JSON(http.StatusBadRequest, gin.H{
					"status": "error",
					"error": utils.SystemError(
						enum.ErrorCodeValidation,
						"Invalid request path",
						nil,
					),
				})
				c.Abort()
				return
			}
		}

		c.Next()
	})
}

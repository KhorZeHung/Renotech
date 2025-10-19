package middleware

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware implements rate limiting per IP
func RateLimitMiddleware() gin.HandlerFunc {
	// Get rate limit configuration from environment
	requests := getEnvInt("RATE_LIMIT_REQUESTS", 100)
	window := getEnvInt("RATE_LIMIT_WINDOW", 60)

	// Create a rate limiter that allows 'requests' requests per 'window' seconds
	limiter := rate.NewLimiter(rate.Every(time.Duration(window)*time.Second/time.Duration(requests)), requests)

	return gin.HandlerFunc(func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "Rate limit exceeded",
			})
			c.Abort()
			return
		}
		c.Next()
	})
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Or specific origin
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})
}

// ValidateFileUpload validates file uploads for security
func ValidateFileUpload() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Only apply to file upload endpoints
		if c.Request.Method != "POST" || !strings.Contains(c.Request.URL.Path, "/upload") {
			c.Next()
			return
		}

		// Check content type
		contentType := c.GetHeader("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid content type for file upload",
			})
			c.Abort()
			return
		}

		// Check file size before processing
		maxSize := int64(getEnvInt("MAX_FILE_SIZE", 52428800)) // 50MB default
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"status":  "error",
				"message": "File size exceeds maximum allowed size",
			})
			c.Abort()
			return
		}

		c.Next()
	})
}

// ValidateFilePath prevents path traversal attacks
func ValidateFilePath(filePath string) bool {
	// Clean the path
	cleanPath := filepath.Clean(filePath)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return false
	}

	return true
}

// AllowedFileTypes defines the whitelist of allowed file extensions
var AllowedFileTypes = map[string]bool{
	// Image formats
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
	".svg":  true,
	".ico":  true,
	".tiff": true,
	".tif":  true,
	".heic": true,
	".heif": true,

	// Video formats
	".mp4":  true,
	".mov":  true,
	".avi":  true,
	".mkv":  true,
	".webm": true,
	".mpeg": true,
	".mpg":  true,
	".wmv":  true,
	".flv":  true,
	".m4v":  true,
	".3gp":  true,

	// Audio formats
	".mp3":  true,
	".wav":  true,
	".ogg":  true,
	".flac": true,
	".aac":  true,
	".m4a":  true,
	".wma":  true,
	".aiff": true,
	".opus": true,

	// Document formats
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".ppt":  true,
	".pptx": true,
	".txt":  true,
	".csv":  true,
	".rtf":  true,
	".odt":  true,
	".ods":  true,
	".odp":  true,

	// Archive formats
	".zip": true,
	".rar": true,
	".7z":  true,
	".tar": true,
	".gz":  true,

	// Web formats
	".html": true,
	".css":  true,
	".js":   true,
	".json": true,
	".xml":  true,
}

// IsFileTypeAllowed checks if the file extension is in the whitelist
func IsFileTypeAllowed(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return AllowedFileTypes[ext]
}

// getEnvInt gets integer environment variable with default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

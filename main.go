package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"renotech.com.my/internal/controller"
	"renotech.com.my/internal/middleware"
	"renotech.com.my/internal/utils"
	"renotech.com.my/logs"
)

func init() {
	// // Load environment variables
	// if err := godotenv.Load(); err != nil {
	// 	log.Printf("Warning: .env file not found: %v", err)
	// }

	// init logger
	logs.LoggingSetup()

	// init mongo db
	err := utils.MongoInit()
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB: %v", err)
	}

	// Verify MongoDB connection
	if err := utils.MongoHealthCheck(); err != nil {
		log.Fatalf("MongoDB health check failed: %v", err)
	}

	log.Println("Application initialized successfully")
}

func main() {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware stack
	setupMiddleware(router)

	// Setup API routes
	setupRoutes(router)

	// Get server configuration
	host := getEnvString("SERVER_HOST", "localhost")
	port := getEnvString("SERVER_PORT", "8000")
	addr := fmt.Sprintf("%s:%s", host, port)

	// Create HTTP server
	srv := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Clean up resources
	utils.MongoCleanUp()
	log.Println("Server stopped")
}

// setupMiddleware configures all middleware
func setupMiddleware(router *gin.Engine) {
	// Recovery middleware
	router.Use(gin.Recovery())

	// Custom logging middleware
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	// Security middleware
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RateLimitMiddleware())

	// Validation middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.RequestValidationMiddleware())
	router.Use(middleware.PathValidationMiddleware())
	router.Use(middleware.JSONResponseMiddleware())

	// File upload security
	router.Use(middleware.ValidateFileUpload())
}

// setupRoutes configures all API routes
func setupRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/health", healthCheckHandler)

	// API routes
	controller.SystemAPIInit(router)
	controller.AuthAPIInit(router)
	controller.MediaAPIInit(router)
	controller.QuotationTemplateAPIInit(router)
	controller.DocumentTemplateAPIInit(router)
	controller.CompanyAPIInit(router)
	controller.UserAPIInit(router)
	controller.SupplierAPIInit(router)
	controller.MaterialAPIInit(router)
	controller.FolderAPIInit(router)
	controller.QuotationAPIInit(router)
}

// healthCheckHandler provides a health check endpoint
func healthCheckHandler(c *gin.Context) {
	// Check MongoDB health
	if err := utils.MongoHealthCheck(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// getEnvString gets string environment variable with default
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

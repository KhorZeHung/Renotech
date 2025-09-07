package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/middleware"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/service"
	"renotech.com.my/internal/utils"
)

func mediaFileUploadHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("File upload started", zap.String("endpoint", "/api/v1/media/upload"))
	defer systemContext.Logger.Info("File upload completed")

	// Get the uploaded file from the form data
	file, err := c.FormFile("file")
	if err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"No file provided or invalid file",
			nil,
		))
		return
	}

	// Get module from form data (required)
	module := c.PostForm("module")
	if strings.TrimSpace(module) == "" {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Module is required",
			nil,
		))
		return
	}

	// Validate file type
	if !middleware.IsFileTypeAllowed(file.Filename) {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"File type not allowed",
			map[string]interface{}{"filename": file.Filename},
		))
		return
	}

	// Check file size against configured limit
	maxSize := int64(utils.GetEnvInt("MAX_FILE_SIZE", 5242880)) // 5MB default
	if file.Size > maxSize {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeTooLarge,
			fmt.Sprintf("File size (%d bytes) exceeds maximum allowed size (%d bytes)", file.Size, maxSize),
			nil,
		))
		return
	}

	// Check if upload directory exists, create with proper permissions
	uploadDir := utils.GetEnvString("UPLOAD_DIR", "./assets/client")
	err = os.MkdirAll(uploadDir, 0755) // More restrictive permissions
	if err != nil {
		systemContext.Logger.Error("Failed to create upload directory", zap.Error(err))
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeInternal,
			"Failed to prepare upload directory",
			nil,
		))
		return
	}

	// Generate secure filename
	fileName := fmt.Sprintf("%d_%s", time.Now().UnixMilli(), utils.SanitizeFilename(file.Filename))
	filePath := filepath.Join(uploadDir, fileName)
	// Convert backslashes to forward slashes for cross-platform compatibility
	filePath = strings.ReplaceAll(filePath, "\\", "/")

	// Remove ./ prefix for database storage
	filePath = strings.TrimPrefix(filePath, "./")

	// Validate the final path for security
	if !middleware.ValidateFilePath(filePath) {
		systemContext.Logger.Error("Invalid file path detected", zap.String("path", filePath))
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid file path",
			nil,
		))
		return
	}

	// Save the file to the defined path
	if err = c.SaveUploadedFile(file, filePath); err != nil {
		systemContext.Logger.Error("Failed to save uploaded file", zap.Error(err))
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeInternal,
			"Failed to save file",
			nil,
		))
		return
	}

	input := database.Media{
		Name:      fileName,
		Extension: filepath.Ext(file.Filename),
		Path:      filePath,
		FileName:  file.Filename,
		Module:    module,
		Company:   *systemContext.User.Company,
		CreatedBy: *systemContext.User.ID,
	}

	result, createErr := service.MediaCreate(&input, systemContext)
	if createErr != nil {
		systemContext.Logger.Error("Failed to create media record", zap.Error(createErr))
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeInternal,
			"Failed to create media record",
			nil,
		))
		return
	}

	systemContext.Logger.Info("File uploaded successfully",
		zap.String("filename", fileName),
		zap.String("originalName", file.Filename),
		zap.Int64("size", file.Size),
	)

	// Send response with uploaded file info
	utils.SendSuccessResponse(c, result)
}
func mediaDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Media deletion started", zap.String("endpoint", "/api/v1/media/delete"))
	defer systemContext.Logger.Info("Media deletion completed")

	input := c.Param("id")
	if input == "" {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Media ID is required",
			nil,
		))
		return
	}

	id, err := primitive.ObjectIDFromHex(input)
	if err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid media ID format",
			map[string]interface{}{"id": input},
		))
		return
	}

	err = service.MediaDelete(id, systemContext)
	if err != nil {
		systemContext.Logger.Error("Failed to delete media", zap.Error(err), zap.String("mediaId", id.Hex()))
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeInternal,
			"Failed to delete media",
			nil,
		))
		return
	}

	systemContext.Logger.Info("Media deleted successfully", zap.String("mediaId", id.Hex()))
	utils.SendSuccessMessageResponse(c, "Media deleted successfully")
}

func mediaListHandler(c *gin.Context) {
	var input model.MediaListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.MediaList(input)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func mediaFileFileServerHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	// Get the requested file path from the URL
	filePath := c.Param("filePath")
	if filePath == "" {
		c.JSON(404, gin.H{"error": "File path is required"})
		return
	}

	// Validate the file path for security
	if !middleware.ValidateFilePath(filePath) {
		c.JSON(400, gin.H{"error": "Invalid file path"})
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	fullPath := filepath.Join("./assets", filePath)

	// Normalize path separators
	fullPath = strings.ReplaceAll(fullPath, "\\", "/")
	// Check if file exists and is within assets directory
	cleanPath := filepath.Clean(fullPath)
	cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")
	if !strings.HasPrefix(cleanPath, "./assets") && !strings.HasPrefix(cleanPath, "assets") {
		c.JSON(400, gin.H{"error": "Access denied"})
		return
	}

	// Validate company ownership of the media file
	if err := service.MediaValidateAccess(fullPath, systemContext); err != nil {
		c.JSON(403, gin.H{"error": "Access denied - file not found or unauthorized"})
		return
	}

	// Set Content-Type header based on file extension
	switch ext {
	case ".png":
		c.Header("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		c.Header("Content-Type", "image/jpeg")
	case ".gif":
		c.Header("Content-Type", "image/gif")
	case ".svg":
		c.Header("Content-Type", "image/svg+xml") // SVG format
	case ".mp4":
		c.Header("Content-Type", "video/mp4")
	case ".webm":
		c.Header("Content-Type", "video/webm")
	case ".avi":
		c.Header("Content-Type", "video/x-msvideo")
	case ".pdf":
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
		c.Header("Content-Type", "application/pdf")
	case ".doc", ".docx":
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	case ".xls", ".xlsx":
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	case ".json":
		c.Header("Content-Type", "application/json") // JSON format
	case ".html":
		c.Header("Content-Type", "text/html; charset=utf-8") // HTML format
	case ".css":
		c.Header("Content-Type", "text/css") // CSS format
	case ".js":
		c.Header("Content-Type", "application/javascript") // JavaScript format
	default:
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
		c.Header("Content-Type", "application/octet-stream")
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}

	// Serve the file
	c.File(fullPath)
}

func MediaAPIInit(r *gin.Engine) {
	// Protected endpoints (require JWT auth)
	protectedGroup := r.Group("/api/v1/media")
	protectedGroup.Use(middleware.JWTAuthMiddleware())
	{
		protectedGroup.POST("/upload", mediaFileUploadHandler)
		protectedGroup.DELETE("/delete/:id", mediaDeleteHandler)
	}

	// Unprotected endpoints
	r.POST("/api/v1/media/list", mediaListHandler)

	// Secured file serving endpoint
	assetsGroup := r.Group("/assets")
	assetsGroup.Use(middleware.JWTAuthMiddleware())
	{
		assetsGroup.GET("/*filePath", mediaFileFileServerHandler)
	}
}

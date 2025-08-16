package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/service"
	"renotech.com.my/internal/utils"
	"strings"
	"time"
)

func mediaFileUploadHandler(c *gin.Context) {
	_, err := c.FormFile("file")

	systemContext := utils.SystemContextBaseInit()
	systemContext.Logger.Info("mediaFileUploadHandler", zap.Any("message", "start"))
	defer systemContext.Logger.Info("mediaFileUploadHandler", zap.Any("message", "end"))

	if err != nil {
		utils.SendErrorResponse(c, err.Error())
		return
	}

	// Get the uploaded file from the form data
	file, fileErr := c.FormFile("file") // "file" is the name of the input field in the HTML form
	if fileErr != nil {
		utils.SendErrorResponse(c, fileErr.Error())
		return
	}

	if file.Size > 50*1024*1024 {
		utils.SendErrorResponse(c, "file too large")
		return
	}

	// Check if the logs directory exists, and create it if it doesn't
	err = os.MkdirAll("./assets/client", os.ModePerm)

	if err != nil {
		fmt.Println(err.Error())
		zap.L().Fatal("Failed to create logs directory", zap.Error(err))
	}

	fileName := fmt.Sprintf("%v_%v", time.Now().UnixMilli(), file.Filename)

	filePath := filepath.Join("assets/client", fileName)

	// Save the file to the defined path
	if err = c.SaveUploadedFile(file, filePath); err != nil {
		utils.SendErrorResponse(c, "failed to save file")
		return
	}

	input := database.Media{
		Name:      fileName,
		Extension: filepath.Ext(file.Filename),
		Path:      filePath,
		FileName:  file.Filename,
	}

	result, createErr := service.MediaCreate(&input, systemContext)

	if createErr != nil {
		utils.SendErrorResponse(c, createErr.Error())
		return
	}

	// Send response with paths of uploaded files
	utils.SendSuccessResponse(c, result)
}
func mediaDeleteHandler(c *gin.Context) {
	systemContext := utils.SystemContextBaseInit()
	systemContext.Logger.Info("mediaDeleteHandler", zap.Any("message", "start"))

	defer systemContext.Logger.Info("mediaDeleteHandler", zap.Any("message", "end"))

	input := c.Param("id")

	id, ok := primitive.ObjectIDFromHex(input)

	if ok != nil {
		utils.SendErrorResponse(c, "invalid id")
	}

	err := service.MediaDelete(id, systemContext)

	if err != nil {
		utils.SendErrorResponse(c, err.Error())
		return
	}

	utils.SendSuccessResponse(c, "success")
}
func mediaFileFileServerHandler(c *gin.Context) {
	// Get the requested file path from the URL
	filePath := c.Param("filePath") // filePath will match everything after /assets/
	ext := strings.ToLower(filepath.Ext(filePath))

	// Serve files from the ./assets directory
	fullPath := "./assets/" + filePath

	// Set Content-Type header based on file extension
	switch ext {
	case ".png":
		c.Header("Content-Type", "image/png")
		break
	case ".jpg", ".jpeg":
		c.Header("Content-Type", "image/jpeg")
		break
	case ".gif":
		c.Header("Content-Type", "image/gif")
		break
	case ".svg":
		c.Header("Content-Type", "image/svg+xml") // SVG format
		break
	case ".mp4":
		c.Header("Content-Type", "video/mp4")
		break
	case ".webm":
		c.Header("Content-Type", "video/webm")
		break
	case ".avi":
		c.Header("Content-Type", "video/x-msvideo")
		break
	case ".pdf":
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
		c.Header("Content-Type", "application/pdf")
		break
	case ".doc", ".docx":
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		break
	case ".xls", ".xlsx":
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		break
	case ".json":
		c.Header("Content-Type", "application/json") // JSON format
		break
	case ".html":
		c.Header("Content-Type", "text/html; charset=utf-8") // HTML format
		break
	case ".css":
		c.Header("Content-Type", "text/css") // CSS format
		break
	case ".js":
		c.Header("Content-Type", "application/javascript") // JavaScript format
	default:
		c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
		c.Header("Content-Type", "application/octet-stream")
		break
	}

	// Serve the file if it exists
	c.File(fullPath)
}

func MediaAPIInit(r *gin.Engine) {
	r.POST("/api/v1/media/upload", mediaFileUploadHandler)
	r.DELETE("/api/v1/media/delete/:id", mediaDeleteHandler)
	r.GET("/assets/*filePath", mediaFileFileServerHandler)
}

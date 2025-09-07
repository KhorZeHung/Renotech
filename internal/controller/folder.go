package controller

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/middleware"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/service"
	"renotech.com.my/internal/utils"
)

func folderCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Folder creation started", zap.String("endpoint", "/api/v1/folder"))
	defer ctx.Logger.Info("Folder creation completed")

	var input database.Folder
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.FolderCreate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Folder creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Folder creation successful",
		zap.String("folderID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func folderGetHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	folderID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.FolderGetByID(folderID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func folderListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.FolderListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.FolderList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func folderUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Folder update started", zap.String("endpoint", "/api/v1/folder"))
	defer systemContext.Logger.Info("Folder update completed")

	var input database.Folder
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.FolderUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Folder update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Folder update successful",
		zap.String("folderID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func folderDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Folder deletion started", zap.String("endpoint", "/api/v1/folder/:id"))
	defer systemContext.Logger.Info("Folder deletion completed")

	folderID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	_, err = service.FolderDelete(folderID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Folder deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Folder deletion successful",
		zap.String("folderID", folderID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Folder deleted successfully")
}

func FolderAPIInit(r *gin.Engine) {
	// Folder routes - Protected with tenant auth middleware
	folderGroup := r.Group("/api/v1/folder")
	folderGroup.Use(middleware.JWTAuthMiddleware())
	{
		folderGroup.POST("", folderCreateHandler)
		folderGroup.GET("/:id", folderGetHandler)
		folderGroup.POST("/list", folderListHandler)
		folderGroup.PUT("", folderUpdateHandler)
		folderGroup.DELETE("/:id", folderDeleteHandler)
	}
}
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

func projectCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Project creation started", zap.String("endpoint", "/api/v1/project"))
	defer ctx.Logger.Info("Project creation completed")

	var input database.Project
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.ProjectCreate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Project creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Project creation successful",
		zap.String("projectID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func projectConvertFromQuotationHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Project conversion started", zap.String("endpoint", "/api/v1/project/convert"))
	defer ctx.Logger.Info("Project conversion completed")

	var input model.ProjectConvertRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.ProjectConvertFromQuotation(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Project conversion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Project conversion successful",
		zap.String("projectID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func projectGetHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	projectID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.ProjectGetByID(projectID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func projectListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.ProjectListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.ProjectList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func projectUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Project update started", zap.String("endpoint", "/api/v1/project"))
	defer systemContext.Logger.Info("Project update completed")

	var input model.ProjectUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.ProjectUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Project update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Project update successful",
		zap.String("projectID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func projectDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Project deletion started", zap.String("endpoint", "/api/v1/project/:id"))
	defer systemContext.Logger.Info("Project deletion completed")

	projectID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	_, err = service.ProjectDelete(projectID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Project deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Project deletion successful",
		zap.String("projectID", projectID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Project deleted successfully")
}

func projectGetProcurementStatusHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	projectID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.ProjectGetProcurementStatus(projectID, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func ProjectAPIInit(router *gin.Engine) {
	projectGroup := router.Group("/api/v1/project")
	projectGroup.Use(middleware.JWTAuthMiddleware())
	{
		projectGroup.POST("", projectCreateHandler)
		projectGroup.POST("/convert", projectConvertFromQuotationHandler)
		projectGroup.GET("/:id", projectGetHandler)
		projectGroup.GET("/:id/procurement-status", projectGetProcurementStatusHandler)
		projectGroup.POST("/list", projectListHandler)
		projectGroup.PUT("", projectUpdateHandler)
		projectGroup.DELETE("/:id", projectDeleteHandler)
	}
}

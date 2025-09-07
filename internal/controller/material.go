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

// Tenant handlers
func materialCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Material creation started", zap.String("endpoint", "/api/v1/material"))
	defer ctx.Logger.Info("Material creation completed")

	var input database.Material
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.MaterialTenantCreate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Material creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Material creation successful",
		zap.String("materialID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func materialGetHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	materialID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.MaterialTenantGetByID(materialID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func materialListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.MaterialListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.MaterialTenantList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func materialUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Material update started", zap.String("endpoint", "/api/v1/material"))
	defer systemContext.Logger.Info("Material update completed")

	var input database.Material
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// Validate that ID is provided in payload
	if input.ID == nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Material ID is required in request payload",
			nil,
		))
		return
	}

	result, err := service.MaterialTenantUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Material update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Material update successful",
		zap.String("materialID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func materialDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Material deletion started", zap.String("endpoint", "/api/v1/material/:id"))
	defer systemContext.Logger.Info("Material deletion completed")

	materialID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	err = service.MaterialDelete(materialID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Material deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Material deletion successful",
		zap.String("materialID", materialID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Material deleted successfully")
}

func MaterialAPIInit(r *gin.Engine) {
	// Tenant routes (for company users to manage their own company's materials) - Protected
	tenantGroup := r.Group("/api/v1/material")
	tenantGroup.Use(middleware.JWTAuthMiddleware())
	{
		tenantGroup.POST("", materialCreateHandler)
		tenantGroup.GET("/:id", materialGetHandler)
		tenantGroup.POST("/list", materialListHandler)
		tenantGroup.PUT("", materialUpdateHandler)
		tenantGroup.DELETE("/:id", materialDeleteHandler)
	}
}
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
func supplierCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Supplier creation started", zap.String("endpoint", "/api/v1/supplier"))
	defer ctx.Logger.Info("Supplier creation completed")

	var input database.Supplier
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.SupplierTenantCreate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Supplier creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Supplier creation successful",
		zap.String("supplierID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func supplierGetHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	supplierID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.SupplierTenantGetByID(supplierID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func supplierListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.SupplierListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.SupplierTenantList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func supplierUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Supplier update started", zap.String("endpoint", "/api/v1/supplier/:id"))
	defer systemContext.Logger.Info("Supplier update completed")

	var input database.Supplier
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.SupplierTenantUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Supplier update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Supplier update successful",
		zap.String("supplierID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func supplierDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Supplier deletion started", zap.String("endpoint", "/api/v1/supplier/:id"))
	defer systemContext.Logger.Info("Supplier deletion completed")

	supplierID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	err = service.SupplierDelete(supplierID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Supplier deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Supplier deletion successful",
		zap.String("supplierID", supplierID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Supplier deleted successfully")
}

func SupplierAPIInit(r *gin.Engine) {
	// Tenant routes (for company users to manage their own company's suppliers) - Protected
	tenantGroup := r.Group("/api/v1/supplier")
	tenantGroup.Use(middleware.JWTAuthMiddleware())
	{
		tenantGroup.POST("", supplierCreateHandler)
		tenantGroup.GET("/:id", supplierGetHandler)
		tenantGroup.POST("/list", supplierListHandler)
		tenantGroup.PUT("", supplierUpdateHandler)
		tenantGroup.DELETE("/:id", supplierDeleteHandler)
	}
}

package controller

import (
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

// Tenant handlers
func companyCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Company creation started", zap.String("endpoint", "/api/v1/company"))
	defer ctx.Logger.Info("Company creation completed")

	var input database.Company
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.CompanyTenantCreate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Company creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Company creation successful",
		zap.String("companyID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func companyGetHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	result, err := service.CompanyTenantGet(ctx)
	if err != nil {
		ctx.Logger.Error("Company creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func companyTenantUpdateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Company tenant update started", zap.String("endpoint", "/api/v1/company"))
	defer ctx.Logger.Info("Company tenant update completed")

	var input database.Company
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// Set the company ID from the user context
	input.ID = ctx.User.Company

	result, err := service.CompanyTenantUpdate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Company tenant update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Company tenant update successful",
		zap.String("companyID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func companyDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Company deletion started", zap.String("endpoint", "/api/v1/company/:id"))
	defer systemContext.Logger.Info("Company deletion completed")

	var input primitive.ObjectID
	input = *systemContext.User.Company

	_, err := service.CompanyDelete(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Company deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Company deletion successful",
		zap.String("companyID", input.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Company deleted successfully")
}

// Admin handlers
func companyAdminCreateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Company admin creation started", zap.String("endpoint", "/api/v1/admin/company"))
	defer systemContext.Logger.Info("Company admin creation completed")

	var input database.Company
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.CompanyAdminCreate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Company admin creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Company admin creation successful",
		zap.String("companyID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func companyAdminListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.CompanyListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.CompanyAdminList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func companyAdminDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Company admin deletion started", zap.String("endpoint", "/api/v1/admin/company/:id"))
	defer systemContext.Logger.Info("Company admin deletion completed")

	var input primitive.ObjectID
	input = *systemContext.User.Company

	_, err := service.CompanyDelete(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Company admin deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Company admin deletion successful",
		zap.String("companyID", input.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Company deleted successfully")
}

func companyAdminUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Company admin update started", zap.String("endpoint", "/api/v1/company/:id"))
	defer systemContext.Logger.Info("Company admin update completed")

	var input database.Company
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.CompanyAdminUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Company admin update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Company admin update successful",
		zap.String("companyID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func CompanyAPIInit(r *gin.Engine) {
	// Tenant routes (for company users to manage their own company) - Protected
	tenantGroup := r.Group("/api/v1/company")
	tenantGroup.Use(middleware.JWTAuthMiddleware())
	{
		tenantGroup.POST("", companyCreateHandler)
		tenantGroup.GET("", companyGetHandler)
		tenantGroup.PUT("", companyTenantUpdateHandler)
		tenantGroup.DELETE("", companyDeleteHandler)
	}

	// Admin routes (for system admins to manage all companies) - Protected
	adminGroup := r.Group("/api/v1/admin/company")
	adminGroup.Use(middleware.JWTAuthMiddleware())
	{
		adminGroup.POST("", companyAdminCreateHandler)
		adminGroup.POST("/list", companyAdminListHandler)
		adminGroup.PUT("", companyAdminUpdateHandler)
		adminGroup.DELETE("/:id", companyAdminDeleteHandler)
	}
}

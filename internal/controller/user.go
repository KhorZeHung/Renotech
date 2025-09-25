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

// Tenant endpoints
func userCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("User creation started", zap.String("endpoint", "/api/v1/user"))
	defer ctx.Logger.Info("User creation completed")

	var input database.User
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.UserTenantCreate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("User creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("User creation successful",
		zap.String("userID", result.User.ID.Hex()),
		zap.String("username", result.User.Username),
	)

	utils.SendSuccessResponse(c, result)
}

func userGetByIDHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	userID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.UserTenantGetByID(userID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func userGetHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	userID := ctx.User.ID

	result, err := service.UserTenantGetByID(*userID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func userListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.UserListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.UserTenantList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func userUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("User update started", zap.String("endpoint", "/api/v1/user/:id"))
	defer systemContext.Logger.Info("User update completed")

	var input database.User
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.UserTenantUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("User update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("User update successful",
		zap.String("userID", result.ID.Hex()),
		zap.String("username", result.Username),
	)

	utils.SendSuccessResponse(c, result)
}

func userDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("User deletion started", zap.String("endpoint", "/api/v1/user/:id"))
	defer systemContext.Logger.Info("User deletion completed")

	userID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	_, err = service.UserDelete(userID, systemContext)
	if err != nil {
		systemContext.Logger.Error("User deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("User deletion successful",
		zap.String("userID", userID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "User deleted successfully")
}

// Admin endpoints
func userAdminCreateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("User admin creation started", zap.String("endpoint", "/api/v1/admin/user"))
	defer systemContext.Logger.Info("User admin creation completed")

	var input database.User
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.UserAdminCreate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("User admin creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("User admin creation successful",
		zap.String("userID", result.ID.Hex()),
		zap.String("username", result.Username),
	)

	utils.SendSuccessResponse(c, result)
}

func userAdminListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.UserListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.UserAdminList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func userAdminUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("User admin update started", zap.String("endpoint", "/api/v1/admin/user/:id"))
	defer systemContext.Logger.Info("User admin update completed")

	var input database.User
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.UserAdminUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("User admin update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("User admin update successful",
		zap.String("userID", result.ID.Hex()),
		zap.String("username", result.Username),
	)

	utils.SendSuccessResponse(c, result)
}

func userAdminDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("User admin deletion started", zap.String("endpoint", "/api/v1/admin/user/:id"))
	defer systemContext.Logger.Info("User admin deletion completed")

	userID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}
	_, err = service.UserDelete(userID, systemContext)
	if err != nil {
		systemContext.Logger.Error("User admin deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("User admin deletion successful",
		zap.String("userID", userID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "User deleted successfully")
}

func userChangePasswordHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("User password change started", zap.String("endpoint", "/api/v1/user/change-password"))
	defer ctx.Logger.Info("User password change completed")

	var input model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// Get user ID from context (set by JWT middleware)
	userID := ctx.User.ID

	result, err := service.UserChangePassword(&input, *userID, ctx)
	if err != nil {
		ctx.Logger.Error("User password change failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("User password change successful",
		zap.String("userID", userID.Hex()),
	)

	utils.SendSuccessResponse(c, result)
}

func UserAPIInit(r *gin.Engine) {
	// Tenant routes (for company users to manage users in their company) - Protected
	tenantGroup := r.Group("/api/v1/user")
	tenantGroup.Use(middleware.JWTAuthMiddleware())
	{
		tenantGroup.GET("", userGetHandler)
		tenantGroup.GET("/:id", userGetHandler)
		tenantGroup.POST("/list", userListHandler)
		tenantGroup.PUT("", userUpdateHandler)
		tenantGroup.DELETE("/:id", userDeleteHandler)
		tenantGroup.POST("/change-password", userChangePasswordHandler)
	}

	r.POST("/api/v1/user", userCreateHandler)

	// Admin routes (for system admins to manage all users) - Protected
	adminGroup := r.Group("/api/v1/admin/user")
	adminGroup.Use(middleware.JWTAuthMiddleware())
	{
		adminGroup.POST("", userAdminCreateHandler)
		adminGroup.POST("/list", userAdminListHandler)
		adminGroup.PUT("", userAdminUpdateHandler)
		adminGroup.DELETE("/:id", userAdminDeleteHandler)
	}
}

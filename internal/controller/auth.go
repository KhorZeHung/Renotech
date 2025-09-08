package controller

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/service"
	"renotech.com.my/internal/utils"
)

func loginHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("User login started", zap.String("endpoint", "/api/v1/auth/login"))
	defer systemContext.Logger.Info("User login completed")

	var input model.LoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.AuthLogin(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("User login failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("User login successful",
		zap.String("email", input.Email),
	)

	utils.SendSuccessResponse(c, result)
}

func forgotPasswordHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Forgot password started", zap.String("endpoint", "/api/v1/auth/forgot-password"))
	defer systemContext.Logger.Info("Forgot password completed")

	var input model.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.UserForgotPassword(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Forgot password failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Forgot password email sent",
		zap.String("email", input.Email),
	)

	utils.SendSuccessResponse(c, result)
}

func resetPasswordHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Reset password started", zap.String("endpoint", "/api/v1/auth/reset-password"))
	defer systemContext.Logger.Info("Reset password completed")

	var input model.ResetPasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.UserResetPassword(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Reset password failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Reset password successful",
		zap.String("email", input.Email),
	)

	utils.SendSuccessResponse(c, result)
}

func AuthAPIInit(r *gin.Engine) {
	r.POST("/api/v1/auth/login", loginHandler)
	r.POST("/api/v1/auth/forgot-password", forgotPasswordHandler)
	r.POST("/api/v1/auth/reset-password", resetPasswordHandler)
}
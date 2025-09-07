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

func quotationCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Quotation creation started", zap.String("endpoint", "/api/v1/quotation"))
	defer ctx.Logger.Info("Quotation creation completed")

	var input database.Quotation
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.QuotationCreate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Quotation creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Quotation creation successful",
		zap.String("quotationID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func quotationGetHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	quotationID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.QuotationGetByID(quotationID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func quotationListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.QuotationListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.QuotationList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func quotationUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Quotation update started", zap.String("endpoint", "/api/v1/quotation"))
	defer systemContext.Logger.Info("Quotation update completed")

	var input database.Quotation
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.QuotationUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Quotation update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Quotation update successful",
		zap.String("quotationID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func quotationDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Quotation deletion started", zap.String("endpoint", "/api/v1/quotation/:id"))
	defer systemContext.Logger.Info("Quotation deletion completed")

	quotationID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	_, err = service.QuotationDelete(quotationID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Quotation deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Quotation deletion successful",
		zap.String("quotationID", quotationID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Quotation deleted successfully")
}

func quotationToggleStarHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Quotation star toggle started", zap.String("endpoint", "/api/v1/quotation/:id/star"))
	defer systemContext.Logger.Info("Quotation star toggle completed")

	quotationID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	var input model.QuotationToggleStarRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.QuotationToggleStar(quotationID, input.IsStared, systemContext)
	if err != nil {
		systemContext.Logger.Error("Quotation star toggle failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Quotation star toggle successful",
		zap.String("quotationID", quotationID.Hex()),
		zap.Bool("isStared", result.IsStared),
	)

	utils.SendSuccessResponse(c, result)
}

func QuotationAPIInit(r *gin.Engine) {
	// Quotation routes - Protected with tenant auth middleware
	quotationGroup := r.Group("/api/v1/quotation")
	quotationGroup.Use(middleware.JWTAuthMiddleware())
	{
		quotationGroup.POST("", quotationCreateHandler)
		quotationGroup.GET("/:id", quotationGetHandler)
		quotationGroup.POST("/list", quotationListHandler)
		quotationGroup.PUT("", quotationUpdateHandler)
		quotationGroup.DELETE("/:id", quotationDeleteHandler)
		quotationGroup.PATCH("/:id/star", quotationToggleStarHandler)
	}
}
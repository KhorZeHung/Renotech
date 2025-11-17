package controller

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/middleware"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/service"
	"renotech.com.my/internal/utils"
)

func orderInitHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Order initialization started", zap.String("endpoint", "/api/v1/order/init"))
	defer ctx.Logger.Info("Order initialization completed")

	var input model.OrderInitRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.OrderInit(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Order initialization failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Order initialization successful",
		zap.Int("totalOrders", result.Summary.TotalOrders),
		zap.Int("supplierCount", result.Summary.SupplierCount),
	)

	utils.SendSuccessResponse(c, result)
}

func orderCreateStandaloneHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Standalone order creation started", zap.String("endpoint", "/api/v1/order/standalone"))
	defer ctx.Logger.Info("Standalone order creation completed")

	var input model.OrderStandaloneRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.OrderCreateStandalone(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Standalone order creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Standalone order creation successful",
		zap.String("orderId", result.ID.Hex()),
		zap.String("poNumber", result.PONumber),
	)

	utils.SendSuccessResponse(c, result)
}

func orderCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Order creation started", zap.String("endpoint", "/api/v1/order"))
	defer ctx.Logger.Info("Order creation completed")

	var input model.OrderCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.OrderCreate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Order creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Order creation successful",
		zap.String("orderId", result.ID.Hex()),
		zap.String("poNumber", result.PONumber),
		zap.String("orderType", string(result.OrderType)),
	)

	utils.SendSuccessResponse(c, result)
}

func orderUpdateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Order update started", zap.String("endpoint", "/api/v1/order/:id"))
	defer ctx.Logger.Info("Order update completed")

	orderID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	var input model.OrderUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// Ensure ID from URL matches ID in request body
	input.ID = orderID

	result, err := service.OrderUpdate(&input, ctx)
	if err != nil {
		ctx.Logger.Error("Order update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Order update successful",
		zap.String("orderId", result.ID.Hex()),
		zap.String("poNumber", result.PONumber),
	)

	utils.SendSuccessResponse(c, result)
}

func orderDeleteHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Order deletion started", zap.String("endpoint", "/api/v1/order/:id"))
	defer ctx.Logger.Info("Order deletion completed")

	orderID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	err = service.OrderDelete(orderID, ctx)
	if err != nil {
		ctx.Logger.Error("Order deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Order deletion successful",
		zap.String("orderId", orderID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Order deleted successfully")
}

func orderUpdateStatusHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Order status update started", zap.String("endpoint", "/api/v1/order/:id/status"))
	defer ctx.Logger.Info("Order status update completed")

	orderID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	var input model.OrderStatusUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.OrderUpdateStatus(orderID, input.Status, ctx)
	if err != nil {
		ctx.Logger.Error("Order status update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	ctx.Logger.Info("Order status update successful",
		zap.String("orderId", result.ID.Hex()),
		zap.String("newStatus", string(result.Status)),
	)

	utils.SendSuccessResponse(c, result)
}

func orderListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.OrderListRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.OrderList(input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func orderGetProjectSummaryHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	projectID, err := utils.ValidateObjectID(c.Param("projectId"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.OrderGetProjectSummary(projectID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func OrderAPIInit(router *gin.Engine) {
	orderGroup := router.Group("/api/v1/order")
	orderGroup.Use(middleware.JWTAuthMiddleware())
	{
		orderGroup.POST("/init", orderInitHandler)
		orderGroup.POST("/standalone", orderCreateStandaloneHandler)
		orderGroup.POST("", orderCreateHandler)
		orderGroup.PUT("/:id", orderUpdateHandler)
		orderGroup.DELETE("/:id", orderDeleteHandler)
		orderGroup.PATCH("/:id/status", orderUpdateStatusHandler)
		orderGroup.POST("/list", orderListHandler)
		orderGroup.GET("/project/:projectId/summary", orderGetProjectSummaryHandler)
	}
}

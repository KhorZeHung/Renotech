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
	ctx.Logger.Info("Order initialization started", zap.String("endpoint", "/api/v1/orders/init"))
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
		zap.String("projectID", input.ProjectID.Hex()),
		zap.Int("ordersCreated", result.Summary.TotalOrders),
		zap.Float64("totalValue", result.Summary.TotalValue),
	)

	utils.SendSuccessResponse(c, result)
}

func orderCreateHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)
	ctx.Logger.Info("Order creation started", zap.String("endpoint", "/api/v1/orders"))
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
		zap.String("orderID", result.ID.Hex()),
		zap.String("poNumber", result.PONumber),
		zap.String("supplier", result.Supplier.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func orderGetHandler(c *gin.Context) {
	ctx := utils.GetSystemContextFromGin(c)

	orderID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.OrderGetByID(orderID, ctx)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

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

func orderUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Order update started", zap.String("endpoint", "/api/v1/orders"))
	defer systemContext.Logger.Info("Order update completed")

	var input model.OrderUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.OrderUpdate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Order update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Order update successful",
		zap.String("orderID", result.ID.Hex()),
		zap.String("poNumber", result.PONumber),
	)

	utils.SendSuccessResponse(c, result)
}

func orderDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Order deletion started", zap.String("endpoint", "/api/v1/orders/:id"))
	defer systemContext.Logger.Info("Order deletion completed")

	orderID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	_, err = service.OrderDelete(orderID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Order deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Order deletion successful",
		zap.String("orderID", orderID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Order deleted successfully")
}

func orderStatusUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Order status update started", zap.String("endpoint", "/api/v1/orders/:id/status"))
	defer systemContext.Logger.Info("Order status update completed")

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

	result, err := service.OrderUpdateStatus(orderID, input.Status, systemContext)
	if err != nil {
		systemContext.Logger.Error("Order status update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Order status update successful",
		zap.String("orderID", orderID.Hex()),
		zap.String("newStatus", string(result.Status)),
	)

	utils.SendSuccessResponse(c, result)
}

func orderDuplicateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Order duplication started", zap.String("endpoint", "/api/v1/orders/:id/duplicate"))
	defer systemContext.Logger.Info("Order duplication completed")

	orderID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.OrderDuplicate(orderID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Order duplication failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Order duplication successful",
		zap.String("originalOrderID", orderID.Hex()),
		zap.String("newOrderID", result.ID.Hex()),
		zap.String("newPONumber", result.PONumber),
	)

	utils.SendSuccessResponse(c, result)
}

func OrderAPIInit(r *gin.Engine) {
	// Order routes - Protected with tenant auth middleware
	orderGroup := r.Group("/api/v1/orders")
	orderGroup.Use(middleware.JWTAuthMiddleware())
	{
		orderGroup.POST("/init", orderInitHandler)                    // Generate orders from project
		orderGroup.POST("", orderCreateHandler)                      // Create new order
		orderGroup.GET("/:id", orderGetHandler)                      // Get order by ID
		orderGroup.POST("/list", orderListHandler)                   // List orders (with filters)
		orderGroup.PUT("", orderUpdateHandler)                       // Update order
		orderGroup.DELETE("/:id", orderDeleteHandler)                // Delete order
		orderGroup.PATCH("/:id/status", orderStatusUpdateHandler)    // Update order status
		orderGroup.POST("/:id/duplicate", orderDuplicateHandler)     // Duplicate order
	}
}
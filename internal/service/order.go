package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/utils"
)

// orderInit initializes draft orders from project materials grouped by supplier
func OrderInit(input *model.OrderInitRequest, systemContext *model.SystemContext) (*model.OrderInitResponse, error) {
	// Validate project exists and belongs to user's company
	project, err := ProjectGetByID(input.ProjectID, systemContext)
	if err != nil {
		return nil, err
	}

	// If project has no materials, return empty response
	if len(project.AreaMaterials) == 0 {
		return &model.OrderInitResponse{
			Orders: []database.Order{},
			Summary: struct {
				TotalOrders   int                                   `json:"totalOrders"`
				SupplierCount int                                   `json:"supplierCount"`
				TotalValue    float64                               `json:"totalValue"`
				BySupplier    map[string]model.OrderSupplierSummary `json:"bySupplier"`
			}{
				TotalOrders:   0,
				SupplierCount: 0,
				TotalValue:    0,
				BySupplier:    make(map[string]model.OrderSupplierSummary),
			},
		}, nil
	}

	now := time.Now()

	// Group materials by supplier
	materialCollection := systemContext.MongoDB.Collection("material")
	supplierGroups := make(map[string]*supplierOrderGroup)

	for areaIndex, areaMaterial := range project.AreaMaterials {
		for _, materialDetail := range areaMaterial.Materials {
			// Skip if no material reference
			if materialDetail.Material == nil {
				continue
			}

			// Fetch material from database to get supplier info
			var material database.Material
			err := materialCollection.FindOne(context.Background(), bson.M{
				"_id":       materialDetail.Material,
				"isDeleted": false,
			}).Decode(&material)

			if err != nil {
				continue
			}

			// Get supplier ID (use material name as fallback if no supplier)
			supplierKey := "no-supplier"
			if material.Company != primitive.NilObjectID {
				supplierKey = material.Company.Hex()
			}

			// Initialize supplier group if not exists
			if _, exists := supplierGroups[supplierKey]; !exists {
				supplierGroups[supplierKey] = &supplierOrderGroup{
					SupplierID:   &material.Company,
					SupplierName: "Unknown Supplier",
					Items:        []database.OrderItem{},
				}
			}

			// Add item to supplier group
			quantity := materialDetail.Quantity
			unitPrice := material.CostPerUnit // Use cost per unit for procurement
			totalPrice := quantity * unitPrice

			orderItem := database.OrderItem{
				Material:    materialDetail.Material,
				ProjectArea: &areaIndex,
				Name:        materialDetail.Name,
				Description: materialDetail.Description,
				Brand:       materialDetail.Brand,
				Unit:        materialDetail.Unit,
				Quantity:    quantity,
				UnitPrice:   unitPrice,
				TotalPrice:  totalPrice,
				Remark:      materialDetail.Remark,
			}

			supplierGroups[supplierKey].Items = append(supplierGroups[supplierKey].Items, orderItem)
		}
	}

	// Create draft orders for each supplier
	orders := []database.Order{}
	totalValue := 0.0
	bySupplierSummary := make(map[string]model.OrderSupplierSummary)

	for supplierKey, group := range supplierGroups {
		if len(group.Items) == 0 {
			continue
		}

		// Calculate totals
		subtotal, taxAmount, totalCharge := calculateOrderTotals(group.Items, 0) // Default 0% tax

		// Create draft order
		order := database.Order{
			Project:   &input.ProjectID,
			Company:   systemContext.User.Company,
			OrderType: enum.OrderTypeProjectMaterial,
			Supplier: database.OrderSupplier{
				ID:   group.SupplierID,
				Name: group.SupplierName,
			},
			OrderDate:        &now,
			ExpectedDelivery: nil,
			Items:            group.Items,
			SubTotal:         subtotal,
			TaxRate:          0,
			TaxAmount:        taxAmount,
			TotalCharge:      totalCharge,
			Status:           enum.OrderStatusDraft,
			Priority:         enum.OrderPriorityMedium,
			IsDeleted:        false,
		}

		orders = append(orders, order)
		totalValue += totalCharge

		// Build supplier summary
		bySupplierSummary[supplierKey] = model.OrderSupplierSummary{
			SupplierName: group.SupplierName,
			ItemCount:    len(group.Items),
			TotalValue:   totalCharge,
		}
	}

	response := &model.OrderInitResponse{
		Orders: orders,
		Summary: struct {
			TotalOrders   int                                   `json:"totalOrders"`
			SupplierCount int                                   `json:"supplierCount"`
			TotalValue    float64                               `json:"totalValue"`
			BySupplier    map[string]model.OrderSupplierSummary `json:"bySupplier"`
		}{
			TotalOrders:   len(orders),
			SupplierCount: len(supplierGroups),
			TotalValue:    totalValue,
			BySupplier:    bySupplierSummary,
		},
	}

	return response, nil
}

// orderCreateStandalone creates a standalone order
func OrderCreateStandalone(input *model.OrderStandaloneRequest, systemContext *model.SystemContext) (*database.Order, error) {
	// Validate project exists
	_, err := ProjectGetByID(input.Project, systemContext)
	if err != nil {
		return nil, err
	}

	// Calculate totals
	subtotal, taxAmount, totalCharge := calculateOrderTotals(input.Items, input.TaxRate)

	// Generate PO number
	poNumber, err := generatePONumber(systemContext)
	if err != nil {
		return nil, err
	}

	// Create action log
	actionLogs := []database.SystemActionLog{
		{
			Description: fmt.Sprintf("Standalone order created by %s", systemContext.User.Username),
			Time:        time.Now(),
			ByName:      systemContext.User.Username,
			ById:        systemContext.User.ID,
		},
	}

	// Set default priority if not provided
	priority := input.Priority
	if priority == "" {
		priority = enum.OrderPriorityMedium
	}

	// Create order
	order := &database.Order{
		Project:          &input.Project,
		Company:          systemContext.User.Company,
		OrderType:        enum.OrderTypeStandalone,
		Supplier:         input.Supplier,
		PONumber:         poNumber,
		OrderDate:        &input.OrderDate,
		ExpectedDelivery: &input.ExpectedDelivery,
		DeliveryAddress:  input.DeliveryAddress,
		DeliveryContact:  input.DeliveryContact,
		DeliveryPhone:    input.DeliveryPhone,
		DeliveryRemark:   input.DeliveryRemark,
		TermConditions:   input.TermConditions,
		Items:            input.Items,
		SubTotal:         subtotal,
		TaxRate:          input.TaxRate,
		TaxAmount:        taxAmount,
		TotalCharge:      totalCharge,
		Status:           enum.OrderStatusDraft,
		Priority:         priority,
		Remark:           input.Remark,
		InternalNotes:    input.InternalNotes,
		ActionLogs:       actionLogs,
		CreatedAt:        time.Now(),
		CreatedBy:        *systemContext.User.ID,
		UpdatedAt:        time.Now(),
		UpdatedBy:        systemContext.User.ID,
		IsDeleted:        false,
	}

	// Insert to database
	collection := systemContext.MongoDB.Collection("order")
	result, err := collection.InsertOne(context.Background(), order)
	if err != nil {
		systemContext.Logger.Error("service.OrderCreateStandalone", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create standalone order", nil)
	}

	orderID := result.InsertedID.(primitive.ObjectID)
	order.ID = &orderID

	// NOTE: Standalone orders do NOT update Project.MaterialSummary

	return order, nil
}

// OrderCreate creates an order (both project-material and standalone types)
func OrderCreate(input *model.OrderCreateRequest, systemContext *model.SystemContext) (*database.Order, error) {
	// Validate project exists
	_, err := ProjectGetByID(input.Project, systemContext)
	if err != nil {
		return nil, err
	}

	// Validate order type specific requirements
	if input.OrderType == enum.OrderTypeProjectMaterial {
		// Project-material orders can optionally specify source area index
		// No strict validation needed as it's for tracking purposes
	}

	// Calculate totals
	subtotal, taxAmount, totalCharge := calculateOrderTotals(input.Items, input.TaxRate)

	// Generate PO number
	poNumber, err := generatePONumber(systemContext)
	if err != nil {
		return nil, err
	}

	// Create action log
	actionLogs := []database.SystemActionLog{
		{
			Description: fmt.Sprintf("Order created by %s", systemContext.User.Username),
			Time:        time.Now(),
			ByName:      systemContext.User.Username,
			ById:        systemContext.User.ID,
		},
	}

	// Set default priority if not provided
	priority := input.Priority
	if priority == "" {
		priority = enum.OrderPriorityMedium
	}

	// Create order
	order := &database.Order{
		Project:          &input.Project,
		Company:          systemContext.User.Company,
		OrderType:        input.OrderType,
		Supplier:         input.Supplier,
		PONumber:         poNumber,
		OrderDate:        input.OrderDate,
		ExpectedDelivery: input.ExpectedDelivery,
		DeliveryAddress:  input.DeliveryAddress,
		DeliveryContact:  input.DeliveryContact,
		DeliveryPhone:    input.DeliveryPhone,
		DeliveryRemark:   input.DeliveryRemark,
		TermConditions:   input.TermConditions,
		Items:            input.Items,
		SubTotal:         subtotal,
		TaxRate:          input.TaxRate,
		TaxAmount:        taxAmount,
		TotalCharge:      totalCharge,
		Status:           enum.OrderStatusDraft,
		Priority:         priority,
		Remark:           input.Remark,
		InternalNotes:    input.InternalNotes,
		ActionLogs:       actionLogs,
		CreatedAt:        time.Now(),
		CreatedBy:        *systemContext.User.ID,
		UpdatedAt:        time.Now(),
		UpdatedBy:        systemContext.User.ID,
		IsDeleted:        false,
	}

	// Insert to database
	collection := systemContext.MongoDB.Collection("order")
	result, err := collection.InsertOne(context.Background(), order)
	if err != nil {
		systemContext.Logger.Error("service.OrderCreate", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create order", nil)
	}

	orderID := result.InsertedID.(primitive.ObjectID)
	order.ID = &orderID

	// Update project material summary if this is a project-material order
	if input.OrderType == enum.OrderTypeProjectMaterial {
		err = updateProjectMaterialSummary(input.Project, input.Items, systemContext)
		if err != nil {
			// Log error but don't fail the order creation
			systemContext.Logger.Error("service.OrderCreate - Failed to update material summary",
				zap.Error(err),
				zap.String("projectId", input.Project.Hex()),
			)
		}
	}

	return order, nil
}

// OrderUpdate updates an existing order
func OrderUpdate(input *model.OrderUpdateRequest, systemContext *model.SystemContext) (*database.Order, error) {
	collection := systemContext.MongoDB.Collection("order")

	// Check if order exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var existingOrder database.Order
	err := collection.FindOne(context.Background(), filter).Decode(&existingOrder)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Order not found", nil)
	}

	// Calculate new totals
	subtotal, taxAmount, totalCharge := calculateOrderTotals(input.Items, input.TaxRate)

	// Create action log
	actionLog := database.SystemActionLog{
		Description: fmt.Sprintf("Order updated by %s", systemContext.User.Username),
		Time:        time.Now(),
		ByName:      systemContext.User.Username,
		ById:        systemContext.User.ID,
	}

	updatedActionLogs := append(existingOrder.ActionLogs, actionLog)

	// Build update
	update := bson.M{
		"$set": bson.M{
			"supplier":         input.Supplier,
			"orderDate":        input.OrderDate,
			"expectedDelivery": input.ExpectedDelivery,
			"deliveryAddress":  input.DeliveryAddress,
			"deliveryContact":  input.DeliveryContact,
			"deliveryPhone":    input.DeliveryPhone,
			"deliveryRemark":   input.DeliveryRemark,
			"termConditions":   input.TermConditions,
			"items":            input.Items,
			"subTotal":         subtotal,
			"taxRate":          input.TaxRate,
			"taxAmount":        taxAmount,
			"totalCharge":      totalCharge,
			"priority":         input.Priority,
			"remark":           input.Remark,
			"internalNotes":    input.InternalNotes,
			"actionLogs":       updatedActionLogs,
			"updatedAt":        time.Now(),
			"updatedBy":        systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		systemContext.Logger.Error("service.OrderUpdate", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update order", nil)
	}

	// Fetch updated order
	var updatedOrder database.Order
	err = collection.FindOne(context.Background(), filter).Decode(&updatedOrder)
	if err != nil {
		systemContext.Logger.Error("service.OrderUpdate - Failed to fetch updated order", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated order", nil)
	}

	// Recalculate material summary if this is a project-material order
	if existingOrder.OrderType == enum.OrderTypeProjectMaterial && existingOrder.Project != nil {
		err = recalculateProjectMaterialSummary(*existingOrder.Project, systemContext)
		if err != nil {
			systemContext.Logger.Error("service.OrderUpdate - Failed to recalculate material summary",
				zap.Error(err),
				zap.String("projectId", existingOrder.Project.Hex()),
			)
		}
	}

	return &updatedOrder, nil
}

// OrderDelete soft deletes an order
func OrderDelete(orderID primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("order")

	// Check if order exists
	filter := bson.M{
		"_id":       orderID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var order database.Order
	err := collection.FindOne(context.Background(), filter).Decode(&order)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeNotFound, "Order not found", nil)
	}

	// Create deletion action log
	deletionLog := database.SystemActionLog{
		Description: fmt.Sprintf("Order deleted by %s", systemContext.User.Username),
		Time:        time.Now(),
		ByName:      systemContext.User.Username,
		ById:        systemContext.User.ID,
	}

	updatedActionLogs := append(order.ActionLogs, deletionLog)

	// Soft delete
	update := bson.M{
		"$set": bson.M{
			"isDeleted":  true,
			"actionLogs": updatedActionLogs,
			"updatedAt":  time.Now(),
			"updatedBy":  systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		systemContext.Logger.Error("service.OrderDelete", zap.Error(err))
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to delete order", nil)
	}

	// Recalculate material summary if this is a project-material order
	if order.OrderType == enum.OrderTypeProjectMaterial && order.Project != nil {
		err = recalculateProjectMaterialSummary(*order.Project, systemContext)
		if err != nil {
			systemContext.Logger.Error("service.OrderDelete - Failed to recalculate material summary",
				zap.Error(err),
				zap.String("projectId", order.Project.Hex()),
			)
		}
	}

	return nil
}

// OrderUpdateStatus updates the status of an order
func OrderUpdateStatus(orderID primitive.ObjectID, status enum.OrderStatus, systemContext *model.SystemContext) (*database.Order, error) {
	collection := systemContext.MongoDB.Collection("order")

	// Check if order exists
	filter := bson.M{
		"_id":       orderID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var order database.Order
	err := collection.FindOne(context.Background(), filter).Decode(&order)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Order not found", nil)
	}

	// Create action log for status change
	actionLog := database.SystemActionLog{
		Description: fmt.Sprintf("Order status changed from %s to %s by %s", order.Status, status, systemContext.User.Username),
		Time:        time.Now(),
		ByName:      systemContext.User.Username,
		ById:        systemContext.User.ID,
	}

	updatedActionLogs := append(order.ActionLogs, actionLog)

	// Update status
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"actionLogs": updatedActionLogs,
			"updatedAt":  time.Now(),
			"updatedBy":  systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		systemContext.Logger.Error("service.OrderUpdateStatus", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update order status", nil)
	}

	// Fetch updated order
	err = collection.FindOne(context.Background(), filter).Decode(&order)
	if err != nil {
		systemContext.Logger.Error("service.OrderUpdateStatus - Failed to fetch updated order", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated order", nil)
	}

	return &order, nil
}

// OrderList returns a paginated list of orders with filters
func OrderList(input model.OrderListRequest, systemContext *model.SystemContext) (*model.OrderListResponse, error) {
	if systemContext.User.Company == nil {
		return &model.OrderListResponse{
			Data:       []bson.M{},
			Page:       1,
			Limit:      10,
			Total:      0,
			TotalPages: 0,
		}, nil
	}

	collection := systemContext.MongoDB.Collection("order")

	// Build base filter
	filter := bson.M{
		"isDeleted": false,
		"company":   systemContext.User.Company,
	}

	// Add filters
	if input.Project != nil {
		filter["project"] = input.Project
	}

	if input.OrderType != nil {
		filter["orderType"] = *input.OrderType
	}

	if input.SupplierID != nil {
		filter["supplier._id"] = input.SupplierID
	}

	if strings.TrimSpace(input.SupplierName) != "" {
		filter["supplier.name"] = primitive.Regex{Pattern: input.SupplierName, Options: "i"}
	}

	if input.Status != nil {
		filter["status"] = *input.Status
	}

	if input.Priority != nil {
		filter["priority"] = *input.Priority
	}

	if strings.TrimSpace(input.PONumber) != "" {
		filter["poNumber"] = primitive.Regex{Pattern: input.PONumber, Options: "i"}
	}

	if input.DateFrom != nil {
		filter["orderDate"] = bson.M{"$gte": *input.DateFrom}
	}

	if input.DateTo != nil {
		if filter["orderDate"] != nil {
			filter["orderDate"].(bson.M)["$lte"] = *input.DateTo
		} else {
			filter["orderDate"] = bson.M{"$lte": *input.DateTo}
		}
	}

	// Add global search
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"poNumber": searchRegex},
				{"supplier.name": searchRegex},
				{"remark": searchRegex},
			},
		}

		if len(filter) > 2 {
			filter = bson.M{
				"$and": []bson.M{
					filter,
					searchFilter,
				},
			}
		} else {
			filter["$or"] = searchFilter["$or"]
		}
	}

	return executeOrderList(collection, filter, input, systemContext)
}

// OrderGetProjectSummary returns summary of orders for a project
func OrderGetProjectSummary(projectID primitive.ObjectID, systemContext *model.SystemContext) (*bson.M, error) {
	// Validate project exists
	_, err := ProjectGetByID(projectID, systemContext)
	if err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("order")

	// Get all orders for this project
	filter := bson.M{
		"project":   projectID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.OrderGetProjectSummary", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve orders", nil)
	}
	defer cursor.Close(context.Background())

	var orders []database.Order
	if err = cursor.All(context.Background(), &orders); err != nil {
		systemContext.Logger.Error("service.OrderGetProjectSummary - Failed to decode orders", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode orders", nil)
	}

	// Calculate summaries
	projectMaterialStats := struct {
		Count      int            `json:"count"`
		TotalValue float64        `json:"totalValue"`
		ByStatus   map[string]int `json:"byStatus"`
	}{
		Count:      0,
		TotalValue: 0,
		ByStatus:   make(map[string]int),
	}

	standaloneStats := struct {
		Count      int            `json:"count"`
		TotalValue float64        `json:"totalValue"`
		ByStatus   map[string]int `json:"byStatus"`
	}{
		Count:      0,
		TotalValue: 0,
		ByStatus:   make(map[string]int),
	}

	for _, order := range orders {
		if order.OrderType == enum.OrderTypeProjectMaterial {
			projectMaterialStats.Count++
			projectMaterialStats.TotalValue += order.TotalCharge
			projectMaterialStats.ByStatus[string(order.Status)]++
		} else if order.OrderType == enum.OrderTypeStandalone {
			standaloneStats.Count++
			standaloneStats.TotalValue += order.TotalCharge
			standaloneStats.ByStatus[string(order.Status)]++
		}
	}

	grandTotal := projectMaterialStats.TotalValue + standaloneStats.TotalValue

	summary := bson.M{
		"projectMaterialOrders": projectMaterialStats,
		"standaloneOrders":      standaloneStats,
		"grandTotal":            grandTotal,
	}

	return &summary, nil
}

// Helper functions

type supplierOrderGroup struct {
	SupplierID   *primitive.ObjectID
	SupplierName string
	Items        []database.OrderItem
}

func calculateOrderTotals(items []database.OrderItem, taxRate float64) (subtotal, taxAmount, totalCharge float64) {
	subtotal = 0.0
	for _, item := range items {
		subtotal += item.TotalPrice
	}

	taxAmount = subtotal * (taxRate / 100)
	totalCharge = subtotal + taxAmount

	return subtotal, taxAmount, totalCharge
}

func generatePONumber(systemContext *model.SystemContext) (string, error) {
	collection := systemContext.MongoDB.Collection("order")

	// Get current year and month
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")

	// Find the latest PO number for this month
	filter := bson.M{
		"company":   systemContext.User.Company,
		"isDeleted": false,
		"poNumber":  primitive.Regex{Pattern: fmt.Sprintf("^PO-%s%s-", year, month), Options: ""},
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	var latestOrder database.Order
	err := collection.FindOne(context.Background(), filter, opts).Decode(&latestOrder)

	sequence := 1
	if err == nil && latestOrder.PONumber != "" {
		// Parse sequence from existing PO number
		var parsedSeq int
		_, err := fmt.Sscanf(latestOrder.PONumber, fmt.Sprintf("PO-%s%s-%%d", year, month), &parsedSeq)
		if err == nil {
			sequence = parsedSeq + 1
		}
	}

	poNumber := fmt.Sprintf("PO-%s%s-%04d", year, month, sequence)
	return poNumber, nil
}

func updateProjectMaterialSummary(projectID primitive.ObjectID, orderItems []database.OrderItem, systemContext *model.SystemContext) error {
	// For simplicity, recalculate the entire summary
	return recalculateProjectMaterialSummary(projectID, systemContext)
}

func recalculateProjectMaterialSummary(projectID primitive.ObjectID, systemContext *model.SystemContext) error {
	// Get project
	project, err := ProjectGetByID(projectID, systemContext)
	if err != nil {
		return err
	}

	// Build material requirements map from project
	materialRequirements := make(map[string]*database.ProjectMaterialSummary)

	for _, areaMaterial := range project.AreaMaterials {
		for _, materialDetail := range areaMaterial.Materials {
			if materialDetail.Material == nil {
				continue
			}

			materialKey := materialDetail.Material.Hex()

			if _, exists := materialRequirements[materialKey]; !exists {
				materialRequirements[materialKey] = &database.ProjectMaterialSummary{
					MaterialID:        materialDetail.Material,
					MaterialName:      materialDetail.Name,
					TotalRequired:     0,
					TotalOrdered:      0,
					ProcurementStatus: enum.ProcurementStatusNotOrdered,
					LastOrderDate:     nil,
				}
			}

			materialRequirements[materialKey].TotalRequired += materialDetail.Quantity
		}
	}

	// Get all project-material orders for this project
	orderCollection := systemContext.MongoDB.Collection("order")
	filter := bson.M{
		"project":   projectID,
		"company":   systemContext.User.Company,
		"orderType": enum.OrderTypeProjectMaterial,
		"isDeleted": false,
	}

	cursor, err := orderCollection.Find(context.Background(), filter)
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())

	var orders []database.Order
	if err = cursor.All(context.Background(), &orders); err != nil {
		return err
	}

	// Aggregate ordered quantities
	for _, order := range orders {
		for _, item := range order.Items {
			if item.Material == nil {
				continue
			}

			materialKey := item.Material.Hex()
			if summary, exists := materialRequirements[materialKey]; exists {
				summary.TotalOrdered += item.Quantity

				// Update last order date
				if summary.LastOrderDate == nil || order.OrderDate.After(*summary.LastOrderDate) {
					summary.LastOrderDate = order.OrderDate
				}
			}
		}
	}

	// Calculate procurement status for each material
	summaryList := []database.ProjectMaterialSummary{}
	for _, summary := range materialRequirements {
		if summary.TotalOrdered == 0 {
			summary.ProcurementStatus = enum.ProcurementStatusNotOrdered
		} else if summary.TotalOrdered < summary.TotalRequired {
			summary.ProcurementStatus = enum.ProcurementStatusPartial
		} else if summary.TotalOrdered == summary.TotalRequired {
			summary.ProcurementStatus = enum.ProcurementStatusFullyOrdered
		} else {
			summary.ProcurementStatus = enum.ProcurementStatusOverOrdered
		}

		summaryList = append(summaryList, *summary)
	}

	// Update project with new material summary
	projectCollection := systemContext.MongoDB.Collection("project")
	update := bson.M{
		"$set": bson.M{
			"materialSummary": summaryList,
			"updatedAt":       time.Now(),
			"updatedBy":       systemContext.User.ID,
		},
	}

	_, err = projectCollection.UpdateOne(context.Background(), bson.M{"_id": projectID}, update)
	if err != nil {
		return err
	}

	return nil
}

func executeOrderList(collection *mongo.Collection, filter bson.M, input model.OrderListRequest, systemContext *model.SystemContext) (*model.OrderListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.OrderList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count orders", nil)
	}

	// Set default pagination values
	page := input.Page
	if page <= 0 {
		page = 1
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Calculate pagination
	skip := (page - 1) * limit
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	// Build sort options
	var sortOptions bson.D
	if len(input.Sort) > 0 {
		for key, value := range input.Sort {
			sortOptions = append(sortOptions, bson.E{Key: key, Value: value})
		}
	} else {
		sortOptions = bson.D{{Key: "createdAt", Value: -1}}
	}

	// Create find options
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(sortOptions)

	// Execute query
	cursor, err := collection.Find(context.Background(), filter, findOptions)
	if err != nil {
		systemContext.Logger.Error("service.OrderList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve orders", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var orders []bson.M
	if err = cursor.All(context.Background(), &orders); err != nil {
		systemContext.Logger.Error("service.OrderList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode orders", nil)
	}

	response := &model.OrderListResponse{
		Data:       orders,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

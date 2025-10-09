package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
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

func OrderInit(input *model.OrderInitRequest, systemContext *model.SystemContext) (*model.OrderInitResponse, error) {
	// Get the project
	project, err := ProjectGetByID(input.ProjectID, systemContext)
	if err != nil {
		return nil, err
	}

	// Aggregate materials by supplier
	supplierMaterials := aggregateMaterialsBySupplier(project.AreaMaterials)

	var orders []database.Order
	supplierSummaries := make(map[string]model.OrderSupplierSummary)
	totalValue := 0.0

	// Create orders for each supplier
	for supplierKey, materials := range supplierMaterials {
		// Get supplier information
		var orderSupplier database.OrderSupplier
		if supplierKey == "no-supplier" {
			orderSupplier = database.OrderSupplier{
				Name: "External Supplier",
			}
		} else {
			supplierObjID, _ := primitive.ObjectIDFromHex(supplierKey)
			supplierInfo, err := populateSupplierFromSystem(supplierObjID, systemContext)
			if err != nil {
				// If supplier not found in system, create placeholder
				orderSupplier = database.OrderSupplier{
					Name: "Unknown Supplier",
				}
			} else {
				orderSupplier = *supplierInfo
			}
		}

		// Convert materials to order items
		var orderItems []database.OrderItem
		for _, material := range materials {
			orderItem := database.OrderItem{
				Material:    material.Material,
				Name:        material.Name,
				Description: fmt.Sprintf("From project: %s", project.Description),
				Brand:       material.Brand,
				Unit:        material.Unit,
				Quantity:    material.Quantity,
				UnitPrice:   material.CostPerUnit,
				TotalPrice:  material.TotalCost,
				Remark:      material.Remark,
			}
			orderItems = append(orderItems, orderItem)
		}

		// Calculate totals
		subTotal, taxAmount, totalCharge := calculateOrderTotals(orderItems, 0) // Default 0% tax

		// Generate PO number
		poNumber, err := generateUniquePONumber(systemContext)
		if err != nil {
			return nil, err
		}

		// Create initial action log
		actionLogs := []database.SystemActionLog{
			createOrderActionLog(fmt.Sprintf("Order initialized from project: %s", project.Description), systemContext),
		}

		// Create order
		order := database.Order{
			Project:          &input.ProjectID,
			Company:          systemContext.User.Company,
			Supplier:         orderSupplier,
			PONumber:         poNumber,
			OrderDate:        time.Now(),
			ExpectedDelivery: time.Now().AddDate(0, 0, 30), // Default 30 days
			Items:            orderItems,
			SubTotal:         subTotal,
			TaxRate:          0,
			TaxAmount:        taxAmount,
			TotalCharge:      totalCharge,
			Status:           enum.OrderStatusDraft,
			Priority:         enum.OrderPriorityMedium,
			ActionLogs:       actionLogs,
			CreatedAt:        time.Now(),
			CreatedBy:        *systemContext.User.ID,
			UpdatedAt:        time.Now(),
			UpdatedBy:        systemContext.User.ID,
			IsDeleted:        false,
		}

		orders = append(orders, order)
		totalValue += totalCharge

		// Create supplier summary
		supplierSummaries[supplierKey] = model.OrderSupplierSummary{
			SupplierName: orderSupplier.Name,
			ItemCount:    len(orderItems),
			TotalValue:   totalCharge,
		}
	}

	// Save all orders to database
	if len(orders) > 0 {
		collection := systemContext.MongoDB.Collection("order")
		var docs []interface{}
		for _, order := range orders {
			docs = append(docs, order)
		}

		result, err := collection.InsertMany(context.Background(), docs)
		if err != nil {
			return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create orders", nil)
		}

		// Update orders with inserted IDs
		for i, insertedID := range result.InsertedIDs {
			objectID := insertedID.(primitive.ObjectID)
			orders[i].ID = &objectID
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
			SupplierCount: len(supplierSummaries),
			TotalValue:    totalValue,
			BySupplier:    supplierSummaries,
		},
	}

	return response, nil
}

func OrderCreate(input *model.OrderCreateRequest, systemContext *model.SystemContext) (*database.Order, error) {
	collection := systemContext.MongoDB.Collection("order")

	// Validate supplier if ID is provided
	if input.Supplier.ID != nil {
		supplierInfo, err := populateSupplierFromSystem(*input.Supplier.ID, systemContext)
		if err != nil {
			return nil, err
		}
		// Auto-populate supplier fields if they're empty
		if input.Supplier.Name == "" {
			input.Supplier.Name = supplierInfo.Name
		}
		if input.Supplier.Contact == "" {
			input.Supplier.Contact = supplierInfo.Contact
		}
		if input.Supplier.Email == "" {
			input.Supplier.Email = supplierInfo.Email
		}
		if input.Supplier.Logo == "" {
			input.Supplier.Logo = supplierInfo.Logo
		}
	}

	// Validate supplier name is provided
	if strings.TrimSpace(input.Supplier.Name) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Supplier name is required", nil)
	}

	// Calculate totals
	subTotal, taxAmount, totalCharge := calculateOrderTotals(input.Items, input.TaxRate)

	// Generate PO number
	poNumber, err := generateUniquePONumber(systemContext)
	if err != nil {
		return nil, err
	}

	// Create initial action log
	actionLogs := []database.SystemActionLog{
		createOrderActionLog(fmt.Sprintf("Order created for supplier: %s", input.Supplier.Name), systemContext),
	}

	// Create order object
	order := &database.Order{
		Project:          input.Project,
		Company:          systemContext.User.Company,
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
		SubTotal:         subTotal,
		TaxRate:          input.TaxRate,
		TaxAmount:        taxAmount,
		TotalCharge:      totalCharge,
		Status:           enum.OrderStatusDraft,
		Priority:         input.Priority,
		Remark:           input.Remark,
		InternalNotes:    input.InternalNotes,
		ActionLogs:       actionLogs,
		CreatedAt:        time.Now(),
		CreatedBy:        *systemContext.User.ID,
		UpdatedAt:        time.Now(),
		UpdatedBy:        systemContext.User.ID,
		IsDeleted:        false,
	}

	result, err := collection.InsertOne(context.Background(), order)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create order", nil)
	}

	orderID := result.InsertedID.(primitive.ObjectID)
	order.ID = &orderID

	return order, nil
}

func OrderUpdate(input *model.OrderUpdateRequest, systemContext *model.SystemContext) (*database.Order, error) {
	collection := systemContext.MongoDB.Collection("order")

	// Check if order exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var currentOrder database.Order
	err := collection.FindOne(context.Background(), filter).Decode(&currentOrder)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Order not found", nil)
	}

	// Validate supplier if ID is provided
	if input.Supplier.ID != nil {
		supplierInfo, err := populateSupplierFromSystem(*input.Supplier.ID, systemContext)
		if err != nil {
			return nil, err
		}
		// Auto-populate supplier fields if they're empty
		if input.Supplier.Name == "" {
			input.Supplier.Name = supplierInfo.Name
		}
		if input.Supplier.Contact == "" {
			input.Supplier.Contact = supplierInfo.Contact
		}
		if input.Supplier.Email == "" {
			input.Supplier.Email = supplierInfo.Email
		}
		if input.Supplier.Logo == "" {
			input.Supplier.Logo = supplierInfo.Logo
		}
	}

	// Validate supplier name is provided
	if strings.TrimSpace(input.Supplier.Name) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Supplier name is required", nil)
	}

	// Calculate totals
	subTotal, taxAmount, totalCharge := calculateOrderTotals(input.Items, input.TaxRate)

	// Build action logs for changes
	actionLogs := currentOrder.ActionLogs

	if input.Supplier.Name != currentOrder.Supplier.Name {
		actionLogs = append(actionLogs, createOrderActionLog(fmt.Sprintf("Supplier changed to: %s", input.Supplier.Name), systemContext))
	}
	if !input.OrderDate.Equal(currentOrder.OrderDate) {
		actionLogs = append(actionLogs, createOrderActionLog(fmt.Sprintf("Order date changed to: %s", input.OrderDate.Format("2006-01-02")), systemContext))
	}
	if !input.ExpectedDelivery.Equal(currentOrder.ExpectedDelivery) {
		actionLogs = append(actionLogs, createOrderActionLog(fmt.Sprintf("Expected delivery changed to: %s", input.ExpectedDelivery.Format("2006-01-02")), systemContext))
	}
	if len(input.Items) != len(currentOrder.Items) {
		actionLogs = append(actionLogs, createOrderActionLog("Order items updated", systemContext))
	}

	// Build update object
	updateFields := bson.M{
		"supplier":         input.Supplier,
		"orderDate":        input.OrderDate,
		"expectedDelivery": input.ExpectedDelivery,
		"deliveryAddress":  input.DeliveryAddress,
		"deliveryContact":  input.DeliveryContact,
		"deliveryPhone":    input.DeliveryPhone,
		"deliveryRemark":   input.DeliveryRemark,
		"termConditions":   input.TermConditions,
		"items":            input.Items,
		"subTotal":         subTotal,
		"taxRate":          input.TaxRate,
		"taxAmount":        taxAmount,
		"totalCharge":      totalCharge,
		"priority":         input.Priority,
		"remark":           input.Remark,
		"internalNotes":    input.InternalNotes,
		"actionLogs":       actionLogs,
		"updatedAt":        time.Now(),
		"updatedBy":        systemContext.User.ID,
	}

	update := bson.M{"$set": updateFields}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update order", nil)
	}

	// Return updated order
	var doc database.Order
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated order", nil)
	}

	return &doc, nil
}

func OrderGetByID(orderID primitive.ObjectID, systemContext *model.SystemContext) (*database.Order, error) {
	collection := systemContext.MongoDB.Collection("order")

	filter := bson.M{
		"_id":       orderID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Order
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Order not found", nil)
	}

	return &doc, nil
}

func OrderList(input model.OrderListRequest, systemContext *model.SystemContext) (*model.OrderListResponse, error) {
	// Check if user has a company
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
	filter := bson.M{"isDeleted": false, "company": systemContext.User.Company}

	// Add field-specific filters
	if input.Project != nil {
		filter["project"] = input.Project
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

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"poNumber": searchRegex},
				{"supplier.name": searchRegex},
				{"remark": searchRegex},
				{"internalNotes": searchRegex},
			},
		}

		// Combine existing filter with search filter
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

func OrderDelete(input primitive.ObjectID, systemContext *model.SystemContext) (*database.Order, error) {
	collection := systemContext.MongoDB.Collection("order")

	// Check if order exists
	filter := bson.M{
		"_id":       input,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Order
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Order not found", nil)
	}

	// Add deletion action log
	actionLogs := doc.ActionLogs
	actionLogs = append(actionLogs, createOrderActionLog("Order deleted", systemContext))

	// Soft delete
	update := bson.M{
		"$set": bson.M{
			"isDeleted":  true,
			"actionLogs": actionLogs,
			"updatedAt":  time.Now(),
			"updatedBy":  systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to delete order", nil)
	}

	return &doc, nil
}

func OrderUpdateStatus(orderID primitive.ObjectID, newStatus enum.OrderStatus, systemContext *model.SystemContext) (*database.Order, error) {
	collection := systemContext.MongoDB.Collection("order")

	// Check if order exists
	filter := bson.M{
		"_id":       orderID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Order
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Order not found", nil)
	}

	// Add status change action log
	actionLogs := doc.ActionLogs
	actionLogs = append(actionLogs, createOrderActionLog(fmt.Sprintf("Status changed from %s to %s", doc.Status, newStatus), systemContext))

	// Update status
	update := bson.M{
		"$set": bson.M{
			"status":     newStatus,
			"actionLogs": actionLogs,
			"updatedAt":  time.Now(),
			"updatedBy":  systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update order status", nil)
	}

	// Return updated order
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated order", nil)
	}

	return &doc, nil
}

func OrderDuplicate(orderID primitive.ObjectID, systemContext *model.SystemContext) (*database.Order, error) {
	// Get the original order
	originalOrder, err := OrderGetByID(orderID, systemContext)
	if err != nil {
		return nil, err
	}

	// Generate new PO number
	poNumber, err := generateUniquePONumber(systemContext)
	if err != nil {
		return nil, err
	}

	// Create action log for duplication
	actionLogs := []database.SystemActionLog{
		createOrderActionLog(fmt.Sprintf("Order duplicated from %s", originalOrder.PONumber), systemContext),
	}

	// Create new order based on original
	duplicatedOrder := &database.Order{
		Project:          originalOrder.Project,
		Company:          originalOrder.Company,
		Supplier:         originalOrder.Supplier,
		PONumber:         poNumber,
		OrderDate:        time.Now(),                   // Set to current date
		ExpectedDelivery: time.Now().AddDate(0, 0, 30), // Default 30 days from now
		DeliveryAddress:  originalOrder.DeliveryAddress,
		DeliveryContact:  originalOrder.DeliveryContact,
		DeliveryPhone:    originalOrder.DeliveryPhone,
		DeliveryRemark:   originalOrder.DeliveryRemark,
		TermConditions:   originalOrder.TermConditions,
		Items:            originalOrder.Items, // Copy all items
		SubTotal:         originalOrder.SubTotal,
		TaxRate:          originalOrder.TaxRate,
		TaxAmount:        originalOrder.TaxAmount,
		TotalCharge:      originalOrder.TotalCharge,
		Status:           enum.OrderStatusDraft, // Reset to draft
		Priority:         originalOrder.Priority,
		Remark:           originalOrder.Remark,
		InternalNotes:    originalOrder.InternalNotes,
		ActionLogs:       actionLogs,
		CreatedAt:        time.Now(),
		CreatedBy:        *systemContext.User.ID,
		UpdatedAt:        time.Now(),
		UpdatedBy:        systemContext.User.ID,
		IsDeleted:        false,
	}

	// Save to database
	collection := systemContext.MongoDB.Collection("order")
	result, err := collection.InsertOne(context.Background(), duplicatedOrder)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to duplicate order", nil)
	}

	newOrderID := result.InsertedID.(primitive.ObjectID)
	duplicatedOrder.ID = &newOrderID

	return duplicatedOrder, nil
}

// Helper functions
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
		limit = 100 // Maximum limit
	}

	// Calculate pagination
	skip := (page - 1) * limit
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	// Build sort options - use bson.D to preserve order for multiple sort fields
	var sortOptions bson.D
	if len(input.Sort) > 0 {
		// Convert bson.M to bson.D to preserve field order
		for key, value := range input.Sort {
			sortOptions = append(sortOptions, bson.E{Key: key, Value: value})
		}
	} else {
		// Default sort by createdAt descending (newest first)
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

func createOrderActionLog(description string, systemContext *model.SystemContext) database.SystemActionLog {
	return database.SystemActionLog{
		Description: description,
		Time:        time.Now(),
		ByName:      systemContext.User.Username,
		ById:        systemContext.User.ID,
	}
}

func generateUniquePONumber(systemContext *model.SystemContext) (string, error) {
	collection := systemContext.MongoDB.Collection("order")
	year := time.Now().Year()

	// Find the highest PO number for current year
	filter := bson.M{
		"company":   systemContext.User.Company,
		"isDeleted": false,
		"poNumber": bson.M{
			"$regex": fmt.Sprintf("^PO-%d-", year),
		},
	}

	opts := options.Find().SetSort(bson.M{"poNumber": -1}).SetLimit(1)
	cursor, err := collection.Find(context.Background(), filter, opts)
	if err != nil {
		return "", utils.SystemError(enum.ErrorCodeInternal, "Failed to generate PO number", nil)
	}
	defer cursor.Close(context.Background())

	var lastOrder database.Order
	nextNumber := 1

	if cursor.Next(context.Background()) {
		if err := cursor.Decode(&lastOrder); err == nil {
			// Extract number from PO-YYYY-XXX format
			parts := strings.Split(lastOrder.PONumber, "-")
			if len(parts) == 3 {
				if num, err := strconv.Atoi(parts[2]); err == nil {
					nextNumber = num + 1
				}
			}
		}
	}

	return fmt.Sprintf("PO-%d-%03d", year, nextNumber), nil
}

func populateSupplierFromSystem(supplierID primitive.ObjectID, systemContext *model.SystemContext) (*database.OrderSupplier, error) {
	supplierCollection := systemContext.MongoDB.Collection("supplier")

	filter := bson.M{
		"_id":       supplierID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var supplier database.Supplier
	err := supplierCollection.FindOne(context.Background(), filter).Decode(&supplier)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Supplier not found", nil)
	}

	// Get the primary address (first one if available)
	var address database.SystemAddress
	if len(supplier.OfficeAddress) > 0 {
		address = supplier.OfficeAddress[0]
	}

	return &database.OrderSupplier{
		ID:      supplier.ID,
		Name:    supplier.Name,
		Contact: supplier.Contact,
		Email:   supplier.Email,
		Logo:    supplier.Logo,
		Address: address,
	}, nil
}

func calculateOrderTotals(items []database.OrderItem, taxRate float64) (float64, float64, float64) {
	var subTotal float64

	// Calculate individual item totals and sum them up
	for i := range items {
		items[i].TotalPrice = items[i].UnitPrice * items[i].Quantity
		subTotal += items[i].TotalPrice
	}

	// Calculate tax
	taxAmount := subTotal * (taxRate / 100)
	totalCharge := subTotal + taxAmount

	return subTotal, taxAmount, totalCharge
}

func aggregateMaterialsBySupplier(areaMaterials []database.SystemAreaMaterial) map[string][]database.SystemAreaMaterialDetail {
	supplierMaterials := make(map[string][]database.SystemAreaMaterialDetail)

	for _, areaMaterial := range areaMaterials {
		for _, materialDetail := range areaMaterial.Materials {
			// Process main material
			addMaterialToSupplierMap(supplierMaterials, materialDetail, areaMaterial.Area.Name)

			// Process template materials recursively
			addTemplateMaterials(supplierMaterials, materialDetail.Template, areaMaterial.Area.Name)
		}
	}

	return supplierMaterials
}

func addMaterialToSupplierMap(supplierMap map[string][]database.SystemAreaMaterialDetail, material database.SystemAreaMaterialDetail, areaName string) {
	if material.Supplier == nil {
		// Handle materials without supplier
		supplierKey := "no-supplier"
		supplierMap[supplierKey] = append(supplierMap[supplierKey], material)
	} else {
		supplierKey := material.Supplier.Hex()
		supplierMap[supplierKey] = append(supplierMap[supplierKey], material)
	}
}

func addTemplateMaterials(supplierMap map[string][]database.SystemAreaMaterialDetail, templateMaterials []database.SystemAreaMaterialDetail, areaName string) {
	for _, templateMaterial := range templateMaterials {
		addMaterialToSupplierMap(supplierMap, templateMaterial, areaName)
		// Recursively add nested templates
		addTemplateMaterials(supplierMap, templateMaterial.Template, areaName)
	}
}

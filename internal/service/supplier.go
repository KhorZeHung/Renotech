package service

import (
	"context"
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

// Tenant services
func supplierTenantCreateValidation(input *database.Supplier, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("supplier")

	// Validate logo path if provided
	if err := utils.ValidateFilePath(input.Logo); err != nil {
		return err
	}

	// Check for duplicate supplier name within the same company
	if strings.TrimSpace(input.Name) != "" {
		filter := bson.M{
			"name":      input.Name,
			"company":   systemContext.User.Company,
			"isDeleted": false,
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate supplier name", nil)
		}

		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Supplier name already exists within your company",
				map[string]interface{}{"name": input.Name},
			)
		}
	}

	// Set company from user context and auto-fill fields
	input.Company = systemContext.User.Company
	input.IsDeleted = false
	input.CreatedAt = time.Now()
	input.CreatedBy = *systemContext.User.ID
	input.UpdatedAt = time.Now()
	input.UpdatedBy = systemContext.User.ID

	return nil
}

func SupplierTenantCreate(input *database.Supplier, systemContext *model.SystemContext) (*database.Supplier, error) {
	// Validate input
	if err := supplierTenantCreateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("supplier")

	result, err := collection.InsertOne(context.Background(), input)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create supplier", nil)
	}

	supplierID := result.InsertedID.(primitive.ObjectID)

	var doc database.Supplier
	err = collection.FindOne(context.Background(), bson.M{"_id": supplierID}).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve supplier", nil)
	}

	return &doc, nil
}

func supplierTenantUpdateValidation(input *database.Supplier, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("supplier")

	// Check if supplier exists and belongs to user's company
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Supplier
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeUnauthorized, "Supplier not found or access denied", nil)
	}

	// Validate logo path if provided
	if err := utils.ValidateFilePath(input.Logo); err != nil {
		return err
	}

	// Check for duplicate supplier name within the same company (excluding current supplier)
	if strings.TrimSpace(input.Name) != "" && input.ID != nil {
		filter := bson.M{
			"name":      input.Name,
			"company":   systemContext.User.Company,
			"isDeleted": false,
			"_id":       bson.M{"$ne": input.ID}, // Exclude current supplier
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate supplier name", nil)
		}

		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Supplier name already exists within your company",
				map[string]interface{}{"name": input.Name},
			)
		}
	}

	return nil
}

func SupplierTenantUpdate(input *database.Supplier, systemContext *model.SystemContext) (*database.Supplier, error) {
	// Validate input
	if err := supplierTenantUpdateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("supplier")

	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	update := bson.M{
		"$set": bson.M{
			"label":            input.Label,
			"name":             input.Name,
			"contact":          input.Contact,
			"email":            input.Email,
			"logo":             input.Logo,
			"tags":             input.Tags,
			"description":      input.Description,
			"officeAddress":    input.OfficeAddress,
			"warehouseAddress": input.WarehouseAddress,
			"updatedAt":        time.Now(),
			"updatedBy":        systemContext.User.ID,
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update supplier", nil)
	}

	var doc database.Supplier
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated supplier", nil)
	}

	return &doc, nil
}

func SupplierTenantGetByID(supplierID primitive.ObjectID, systemContext *model.SystemContext) (*database.Supplier, error) {
	collection := systemContext.MongoDB.Collection("supplier")

	filter := bson.M{
		"_id":       supplierID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Supplier
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Supplier not found", nil)
	}

	return &doc, nil
}

func SupplierTenantList(input model.SupplierListRequest, systemContext *model.SystemContext) (*model.SupplierListResponse, error) {
	collection := systemContext.MongoDB.Collection("supplier")

	// Build base filter - tenant can only see their company's suppliers
	filter := bson.M{
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	// Add field-specific filters
	if strings.TrimSpace(input.Label) != "" {
		filter["label"] = primitive.Regex{Pattern: input.Label, Options: "i"}
	}
	if strings.TrimSpace(input.Name) != "" {
		filter["name"] = primitive.Regex{Pattern: input.Name, Options: "i"}
	}
	if strings.TrimSpace(input.Contact) != "" {
		filter["contact"] = primitive.Regex{Pattern: input.Contact, Options: "i"}
	}
	if strings.TrimSpace(input.Email) != "" {
		filter["email"] = primitive.Regex{Pattern: input.Email, Options: "i"}
	}
	if strings.TrimSpace(input.Description) != "" {
		filter["description"] = primitive.Regex{Pattern: input.Description, Options: "i"}
	}

	// Add tag search filter
	if strings.TrimSpace(input.Tags) != "" {
		filter["tags"] = bson.M{
			"$in": []string{input.Tags},
		}
	}

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"label": searchRegex},
				{"name": searchRegex},
				{"contact": searchRegex},
				{"email": searchRegex},
				{"description": searchRegex},
				{"tags": bson.M{"$in": []primitive.Regex{searchRegex}}},
			},
		}

		// Combine existing filter with search filter
		if len(filter) > 2 { // More than just company and isDeleted
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

	return executeSupplierList(collection, filter, input, systemContext)
}

// Shared service
func SupplierDelete(supplierID primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("supplier")

	// Check if supplier exists and belongs to user's company
	filter := bson.M{
		"_id":       supplierID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Supplier
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeNotFound, "Supplier not found or access denied", nil)
	}

	// Soft delete the supplier
	update := bson.M{
		"$set": bson.M{
			"isDeleted": true,
			"updatedAt": time.Now(),
			"updatedBy": systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to delete supplier", nil)
	}

	return nil
}

// Helper functions
func executeSupplierList(collection *mongo.Collection, filter bson.M, input model.SupplierListRequest, systemContext *model.SystemContext) (*model.SupplierListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.SupplierList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count suppliers", nil)
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

	// Build sort options
	sortOptions := input.Sort
	if len(sortOptions) < 1 {
		sortOptions = bson.M{"createdAt": 1}
	}

	// Create find options
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(sortOptions)

	// Execute query
	cursor, err := collection.Find(context.Background(), filter, findOptions)
	if err != nil {
		systemContext.Logger.Error("service.SupplierList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve suppliers", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var suppliers []bson.M
	if err = cursor.All(context.Background(), &suppliers); err != nil {
		systemContext.Logger.Error("service.SupplierList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode suppliers", nil)
	}

	response := &model.SupplierListResponse{
		Data:       suppliers,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}
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
func materialTenantCreateValidation(input *database.Material, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("material")

	// Validate required fields
	if strings.TrimSpace(input.Name) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Name is required", nil)
	}
	if strings.TrimSpace(input.ClientDisplayName) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Client display name is required", nil)
	}
	if strings.TrimSpace(input.SupplierDisplayName) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Supplier display name is required", nil)
	}
	if input.Type == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Type is required", nil)
	}
	if strings.TrimSpace(input.Unit) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Unit is required", nil)
	}
	if input.CostPerUnit <= 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "Cost per unit must be greater than 0", nil)
	}
	if input.PricePerUnit <= 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "Price per unit must be greater than 0", nil)
	}
	if input.Status == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Status is required", nil)
	}

	// Validate supplier if provided - must belong to user's company
	if input.Supplier != nil {
		supplierCollection := systemContext.MongoDB.Collection("supplier")
		var supplierDoc database.Supplier
		err := supplierCollection.FindOne(context.Background(), bson.M{
			"_id":       input.Supplier,
			"company":   *systemContext.User.Company,
			"isDeleted": false,
		}).Decode(&supplierDoc)

		if err != nil {
			return utils.SystemError(enum.ErrorCodeValidation, "Supplier not found or does not belong to your company", nil)
		}
	}

	// Type-specific validation
	switch input.Type {
	case enum.MaterialTypeProduct, enum.MaterialTypeService:
		// Product and Service cannot have template
		if len(input.Template) > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Product and service materials cannot have template",
				map[string]interface{}{"type": input.Type},
			)
		}
	case enum.MaterialTypeTemplate:
		// Template must have template array
		if len(input.Template) == 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Template materials must have at least one template item",
				nil,
			)
		}

		// Validate each template material
		for i, template := range input.Template {
			var materialDoc database.Material
			err := collection.FindOne(context.Background(), bson.M{
				"_id":       template.Material,
				"company":   *systemContext.User.Company,
				"isDeleted": false,
			}).Decode(&materialDoc)

			if err != nil {
				return utils.SystemError(
					enum.ErrorCodeValidation,
					"Template material not found or does not belong to your company",
					map[string]interface{}{"templateIndex": i, "materialId": template.Material.Hex()},
				)
			}

			// Template material cannot be type "template"
			if materialDoc.Type == enum.MaterialTypeTemplate {
				return utils.SystemError(
					enum.ErrorCodeValidation,
					"Template materials cannot reference other template materials",
					map[string]interface{}{"templateIndex": i, "materialName": materialDoc.Name},
				)
			}

			// Template material must be active
			if materialDoc.Status != enum.MaterialStatusActive {
				return utils.SystemError(
					enum.ErrorCodeValidation,
					"Template materials must reference active materials only",
					map[string]interface{}{"templateIndex": i, "materialName": materialDoc.Name, "status": materialDoc.Status},
				)
			}

			input.Template[i].MaterialDoc = materialDoc
		}
	default:
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid material type",
			map[string]interface{}{"type": input.Type},
		)
	}

	// Check for duplicate material name within the same company and supplier
	filter := bson.M{
		"name":      input.Name,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	// Add supplier to filter if provided
	if input.Supplier != nil {
		filter["supplier"] = input.Supplier
	} else {
		filter["supplier"] = bson.M{"$exists": false}
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate material name", nil)
	}

	if count > 0 {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Material name already exists within your company and supplier scope",
			map[string]interface{}{"name": input.Name},
		)
	}

	// Set company from user context and auto-fill fields
	input.Company = *systemContext.User.Company
	input.IsDeleted = false
	input.CreatedAt = time.Now()
	input.CreatedBy = *systemContext.User.ID
	input.UpdatedAt = time.Now()
	input.UpdatedBy = *systemContext.User.ID

	return nil
}

func MaterialTenantCreate(input *database.Material, systemContext *model.SystemContext) (*database.Material, error) {
	// Validate input
	if err := materialTenantCreateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("material")

	result, err := collection.InsertOne(context.Background(), input)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create material", nil)
	}

	materialID := result.InsertedID.(primitive.ObjectID)

	var doc database.Material
	err = collection.FindOne(context.Background(), bson.M{"_id": materialID}).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve material", nil)
	}

	return &doc, nil
}

// Helper function to check if a material is referenced in other materials' templates
func checkMaterialInTemplates(materialID primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("material")

	// Check if this material is referenced in any template
	filter := bson.M{
		"company":           *systemContext.User.Company,
		"isDeleted":         false,
		"template.material": materialID,
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to check material template references", nil)
	}

	if count > 0 {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Material is currently referenced in other materials' templates and cannot be modified or deleted",
			map[string]interface{}{"materialId": materialID.Hex(), "referencedIn": count},
		)
	}

	return nil
}

func materialTenantUpdateValidation(input *database.Material, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("material")

	// Check if material exists and belongs to user's company
	filter := bson.M{
		"_id":       input.ID,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Material
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeUnauthorized, "Material not found or access denied", nil)
	}

	// If changing status from active to another status, check if material is referenced in templates
	if doc.Status == enum.MaterialStatusActive && input.Status != enum.MaterialStatusActive {
		if err := checkMaterialInTemplates(*input.ID, systemContext); err != nil {
			return err
		}
	}

	// Validate required fields
	if strings.TrimSpace(input.Name) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Name is required", nil)
	}
	if strings.TrimSpace(input.ClientDisplayName) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Client display name is required", nil)
	}
	if strings.TrimSpace(input.SupplierDisplayName) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Supplier display name is required", nil)
	}
	if input.Type == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Type is required", nil)
	}
	if strings.TrimSpace(input.Unit) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Unit is required", nil)
	}
	if input.CostPerUnit <= 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "Cost per unit must be greater than 0", nil)
	}
	if input.PricePerUnit <= 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "Price per unit must be greater than 0", nil)
	}
	if input.Status == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Status is required", nil)
	}

	// Validate supplier if provided - must belong to user's company
	if input.Supplier != nil {
		supplierCollection := systemContext.MongoDB.Collection("supplier")
		var supplierDoc database.Supplier
		err := supplierCollection.FindOne(context.Background(), bson.M{
			"_id":       input.Supplier,
			"company":   *systemContext.User.Company,
			"isDeleted": false,
		}).Decode(&supplierDoc)

		if err != nil {
			return utils.SystemError(enum.ErrorCodeValidation, "Supplier not found or does not belong to your company", nil)
		}
	}

	// Type-specific validation
	switch input.Type {
	case enum.MaterialTypeProduct, enum.MaterialTypeService:
		// Product and Service cannot have template
		if len(input.Template) > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Product and service materials cannot have template",
				map[string]interface{}{"type": input.Type},
			)
		}
	case enum.MaterialTypeTemplate:
		// Template must have template array
		if len(input.Template) == 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Template materials must have at least one template item",
				nil,
			)
		}

		// Validate each template material
		for i, template := range input.Template {
			var materialDoc database.Material
			err := collection.FindOne(context.Background(), bson.M{
				"_id":       template.Material,
				"company":   *systemContext.User.Company,
				"isDeleted": false,
			}).Decode(&materialDoc)

			if err != nil {
				return utils.SystemError(
					enum.ErrorCodeValidation,
					"Template material not found or does not belong to your company",
					map[string]interface{}{"templateIndex": i, "materialId": template.Material.Hex()},
				)
			}

			// Template material cannot be type "template"
			if materialDoc.Type == enum.MaterialTypeTemplate {
				return utils.SystemError(
					enum.ErrorCodeValidation,
					"Template materials cannot reference other template materials",
					map[string]interface{}{"templateIndex": i, "materialName": materialDoc.Name},
				)
			}

			// Template material must be active
			if materialDoc.Status != enum.MaterialStatusActive {
				return utils.SystemError(
					enum.ErrorCodeValidation,
					"Template materials must reference active materials only",
					map[string]interface{}{"templateIndex": i, "materialName": materialDoc.Name, "status": materialDoc.Status},
				)
			}

			input.Template[i].MaterialDoc = materialDoc
		}
	default:
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid material type",
			map[string]interface{}{"type": input.Type},
		)
	}

	// Check for duplicate material name within the same company and supplier (excluding current material)
	nameFilter := bson.M{
		"name":      input.Name,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
		"_id":       bson.M{"$ne": input.ID}, // Exclude current material
	}

	// Add supplier to filter if provided
	if input.Supplier != nil {
		nameFilter["supplier"] = input.Supplier
	} else {
		nameFilter["supplier"] = bson.M{"$exists": false}
	}

	count, err := collection.CountDocuments(context.Background(), nameFilter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate material name", nil)
	}

	if count > 0 {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Material name already exists within your company and supplier scope",
			map[string]interface{}{"name": input.Name},
		)
	}

	return nil
}

func MaterialTenantUpdate(input *database.Material, systemContext *model.SystemContext) (*database.Material, error) {
	// Validate input
	if err := materialTenantUpdateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("material")

	filter := bson.M{
		"_id":       input.ID,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	update := bson.M{
		"$set": bson.M{
			"name":                input.Name,
			"clientDisplayName":   input.ClientDisplayName,
			"supplierDisplayName": input.SupplierDisplayName,
			"template":            input.Template,
			"type":                input.Type,
			"supplier":            input.Supplier,
			"brand":               input.Brand,
			"unit":                input.Unit,
			"costPerUnit":         input.CostPerUnit,
			"pricePerUnit":        input.PricePerUnit,
			"categories":          input.Categories,
			"tags":                input.Tags,
			"media":               input.Media,
			"status":              input.Status,
			"remark":              input.Remark,
			"updatedAt":           time.Now(),
			"updatedBy":           *systemContext.User.ID,
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update material", nil)
	}

	var updatedDoc database.Material
	err = collection.FindOne(context.Background(), filter).Decode(&updatedDoc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated material", nil)
	}

	return &updatedDoc, nil
}

func MaterialTenantGetByID(materialID primitive.ObjectID, systemContext *model.SystemContext) (*database.Material, error) {
	collection := systemContext.MongoDB.Collection("material")

	filter := bson.M{
		"_id":       materialID,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Material
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Material not found", nil)
	}

	return &doc, nil
}

func MaterialTenantList(input model.MaterialListRequest, systemContext *model.SystemContext) (*model.MaterialListResponse, error) {
	// Check if user has a company
	if systemContext.User.Company == nil {
		return &model.MaterialListResponse{
			Data:       []bson.M{},
			Page:       1,
			Limit:      10,
			Total:      0,
			TotalPages: 0,
		}, nil
	}

	collection := systemContext.MongoDB.Collection("material")

	// Build base filter - tenant can only see their company's materials
	filter := bson.M{
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	// Add field-specific filters
	if strings.TrimSpace(input.Name) != "" {
		filter["name"] = primitive.Regex{Pattern: input.Name, Options: "i"}
	}
	if strings.TrimSpace(input.ClientDisplayName) != "" {
		filter["clientDisplayName"] = primitive.Regex{Pattern: input.ClientDisplayName, Options: "i"}
	}
	if strings.TrimSpace(input.SupplierDisplayName) != "" {
		filter["supplierDisplayName"] = primitive.Regex{Pattern: input.SupplierDisplayName, Options: "i"}
	}
	if strings.TrimSpace(input.Type) != "" {
		filter["type"] = input.Type
	}
	if strings.TrimSpace(input.Supplier) != "" {
		if supplierID, err := primitive.ObjectIDFromHex(input.Supplier); err == nil {
			filter["supplier"] = supplierID
		}
	}
	if strings.TrimSpace(input.Brand) != "" {
		filter["brand"] = primitive.Regex{Pattern: input.Brand, Options: "i"}
	}
	if strings.TrimSpace(input.Unit) != "" {
		filter["unit"] = primitive.Regex{Pattern: input.Unit, Options: "i"}
	}
	if strings.TrimSpace(input.Status) != "" {
		filter["status"] = input.Status
	}

	// Add category filter
	if strings.TrimSpace(input.Categories) != "" {
		filter["categories"] = bson.M{"$in": []string{input.Categories}}
	}

	// Add tag filter
	if strings.TrimSpace(input.Tags) != "" {
		filter["tags"] = bson.M{"$in": []string{input.Tags}}
	}

	// Add cost per unit filter with MongoDB operators
	if len(input.CostPerUnit) > 0 {
		filter["costPerUnit"] = input.CostPerUnit
	}

	// Add price per unit filter with MongoDB operators
	if len(input.PricePerUnit) > 0 {
		filter["pricePerUnit"] = input.PricePerUnit
	}

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"name": searchRegex},
				{"clientDisplayName": searchRegex},
				{"supplierDisplayName": searchRegex},
				{"brand": searchRegex},
				{"unit": searchRegex},
				{"remark": searchRegex},
				{"categories": bson.M{"$in": []primitive.Regex{searchRegex}}},
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

	return executeMaterialList(collection, filter, input, systemContext)
}

// Shared service
func MaterialDelete(materialID primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("material")

	// Check if material exists and belongs to user's company
	filter := bson.M{
		"_id":       materialID,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Material
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeNotFound, "Material not found or access denied", nil)
	}

	// Check if material is referenced in other materials' templates before deleting
	if err := checkMaterialInTemplates(materialID, systemContext); err != nil {
		return err
	}

	// Soft delete the material
	update := bson.M{
		"$set": bson.M{
			"isDeleted": true,
			"updatedAt": time.Now(),
			"updatedBy": *systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to delete material", nil)
	}

	return nil
}

// Helper functions
func executeMaterialList(collection *mongo.Collection, filter bson.M, input model.MaterialListRequest, systemContext *model.SystemContext) (*model.MaterialListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.MaterialList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count materials", nil)
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
		// Default sort by createdAt ascending
		sortOptions = bson.D{{Key: "createdAt", Value: 1}}
	}

	// Create find options
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(sortOptions)

	// Execute query
	cursor, err := collection.Find(context.Background(), filter, findOptions)
	if err != nil {
		systemContext.Logger.Error("service.MaterialList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve materials", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var materials []bson.M
	if err = cursor.All(context.Background(), &materials); err != nil {
		systemContext.Logger.Error("service.MaterialList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode materials", nil)
	}

	response := &model.MaterialListResponse{
		Data:       materials,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

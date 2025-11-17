package service

import (
	"context"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
			var materialDoc database.MaterialTemplateDoc
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

			input.Template[i].MaterialDoc = materialDoc
		}
	default:
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid material type",
			map[string]interface{}{"type": input.Type},
		)
	}

	// Check for duplicate material name within the same company
	// Rules: Same name allowed if:
	// 1. Each has different supplier (non-null), OR
	// 2. Only one instance has null supplier
	filter := bson.M{
		"name":      input.Name,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	// If input has a supplier, check if this supplier already has a material with this name
	if input.Supplier != nil {
		filter["supplier"] = input.Supplier
		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate material name", nil)
		}
		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Material name already exists for this supplier",
				map[string]interface{}{"name": input.Name},
			)
		}
	} else {
		// If input has null supplier, check if there's already a material with null supplier and same name
		filter["supplier"] = bson.M{"$exists": false}
		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate material name", nil)
		}
		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Material name with no supplier already exists",
				map[string]interface{}{"name": input.Name},
			)
		}
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

	if err := checkMaterialInTemplates(*input.ID, systemContext); err != nil {
		return err
	}

	// Validate required fields
	if strings.TrimSpace(input.Name) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Name is required", nil)
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
			var materialDoc database.MaterialTemplateDoc
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

			input.Template[i].MaterialDoc = materialDoc
		}
	default:
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid material type",
			map[string]interface{}{"type": input.Type},
		)
	}

	// Check for duplicate material name within the same company (excluding current material)
	// Rules: Same name allowed if:
	// 1. Each has different supplier (non-null), OR
	// 2. Only one instance has null supplier
	nameFilter := bson.M{
		"name":      input.Name,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
		"_id":       bson.M{"$ne": input.ID}, // Exclude current material
	}

	// If input has a supplier, check if this supplier already has a material with this name
	if input.Supplier != nil {
		nameFilter["supplier"] = input.Supplier
		count, err := collection.CountDocuments(context.Background(), nameFilter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate material name", nil)
		}
		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Material name already exists for this supplier",
				map[string]interface{}{"name": input.Name},
			)
		}
	} else {
		// If input has null supplier, check if there's already a material with null supplier and same name
		nameFilter["supplier"] = bson.M{"$exists": false}
		count, err := collection.CountDocuments(context.Background(), nameFilter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate material name", nil)
		}
		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Material name with no supplier already exists",
				map[string]interface{}{"name": input.Name},
			)
		}
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
			"name":            input.Name,
			"quotationConfig": input.QuotationConfig,
			"orderConfig":     input.OrderConfig,
			"template":        input.Template,
			"type":            input.Type,
			"brand":           input.Brand,
			"unit":            input.Unit,
			"costPerUnit":     input.CostPerUnit,
			"pricePerUnit":    input.PricePerUnit,
			"media":           input.Media,
			"status":          input.Status,
			"remark":          input.Remark,
			"updatedAt":       time.Now(),
			"updatedBy":       *systemContext.User.ID,
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

func MaterialTenantGetByID(materialID primitive.ObjectID, systemContext *model.SystemContext) (*bson.M, error) {
	collection := systemContext.MongoDB.Collection("material")

	filter := bson.M{
		"_id":       materialID,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	// Build aggregation pipeline to populate supplier
	pipeline := mongo.Pipeline{
		// Match stage - filter material by ID and company
		{{Key: "$match", Value: filter}},

		// Lookup stage - join supplier collection
		{{Key: "$lookup", Value: bson.M{
			"from":         "supplier",
			"localField":   "supplier",
			"foreignField": "_id",
			"as":           "supplierDoc",
		}}},

		// Unwind stage - convert array to object (preserve null)
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$supplierDoc",
			"preserveNullAndEmptyArrays": true,
		}}},

		// AddFields stage - project only selected supplier fields
		{{Key: "$addFields", Value: bson.M{
			"supplierDoc": bson.M{
				"$cond": bson.A{
					bson.M{"$eq": bson.A{"$supplierDoc", bson.M{}}},
					nil,
					bson.M{
						"_id":         "$supplierDoc._id",
						"name":        "$supplierDoc.name",
						"displayName": "$supplierDoc.displayName",
					},
				},
			},
		}}},
	}

	// Execute aggregation
	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		systemContext.Logger.Error("service.MaterialTenantGetByID", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve material", nil)
	}
	defer cursor.Close(context.Background())

	// Decode result
	var results []bson.M
	if err = cursor.All(context.Background(), &results); err != nil {
		systemContext.Logger.Error("service.MaterialTenantGetByID", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode material", nil)
	}

	// Check if material was found
	if len(results) == 0 {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Material not found", nil)
	}

	return &results[0], nil
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
	if strings.TrimSpace(input.Type) != "" {
		filter["type"] = input.Type
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

func MaterialTenantGetAvailableSuppliers(materialName string, systemContext *model.SystemContext) ([]database.Supplier, error) {
	// Validate input
	if strings.TrimSpace(materialName) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Material name is required", nil)
	}

	// Check if user has a company
	if systemContext.User.Company == nil {
		return []database.Supplier{}, nil
	}

	materialCollection := systemContext.MongoDB.Collection("material")
	supplierCollection := systemContext.MongoDB.Collection("supplier")

	// Find all materials with the given name in the user's company
	filter := bson.M{
		"name":      materialName,
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	cursor, err := materialCollection.Find(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.MaterialTenantGetAvailableSuppliers - Failed to find materials", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve materials", nil)
	}
	defer cursor.Close(context.Background())

	// Extract supplier IDs from materials
	var materials []database.Material
	if err = cursor.All(context.Background(), &materials); err != nil {
		systemContext.Logger.Error("service.MaterialTenantGetAvailableSuppliers - Failed to decode materials", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode materials", nil)
	}

	// Collect supplier IDs that already have this material
	usedSupplierIDs := []primitive.ObjectID{}
	for _, material := range materials {
		if material.Supplier != nil {
			usedSupplierIDs = append(usedSupplierIDs, *material.Supplier)
		}
	}

	// Query suppliers that are NOT in the used supplier list
	supplierFilter := bson.M{
		"company":   *systemContext.User.Company,
		"isDeleted": false,
	}

	// Exclude suppliers that already have this material name
	if len(usedSupplierIDs) > 0 {
		supplierFilter["_id"] = bson.M{"$nin": usedSupplierIDs}
	}

	cursor, err = supplierCollection.Find(context.Background(), supplierFilter)
	if err != nil {
		systemContext.Logger.Error("service.MaterialTenantGetAvailableSuppliers - Failed to find suppliers", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve suppliers", nil)
	}
	defer cursor.Close(context.Background())

	var availableSuppliers []database.Supplier
	if err = cursor.All(context.Background(), &availableSuppliers); err != nil {
		systemContext.Logger.Error("service.MaterialTenantGetAvailableSuppliers - Failed to decode suppliers", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode suppliers", nil)
	}

	return availableSuppliers, nil
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

	// Build aggregation pipeline
	pipeline := mongo.Pipeline{
		// Match stage - filter materials
		{{Key: "$match", Value: filter}},

		// Lookup stage - join supplier collection
		{{Key: "$lookup", Value: bson.M{
			"from":         "supplier",
			"localField":   "supplier",
			"foreignField": "_id",
			"as":           "supplierDoc",
		}}},

		// Unwind stage - convert array to object (preserve null)
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$supplierDoc",
			"preserveNullAndEmptyArrays": true,
		}}},

		// AddFields stage - project only selected supplier fields
		{{Key: "$addFields", Value: bson.M{
			"supplierDoc": bson.M{
				"$cond": bson.A{
					bson.M{"$eq": bson.A{"$supplierDoc", bson.M{}}},
					nil,
					bson.M{
						"_id":         "$supplierDoc._id",
						"name":        "$supplierDoc.name",
						"displayName": "$supplierDoc.displayName",
					},
				},
			},
		}}},

		// Sort stage
		{{Key: "$sort", Value: sortOptions}},

		// Facet stage - handle count and data in single query
		{{Key: "$facet", Value: bson.M{
			"metadata": mongo.Pipeline{
				{{Key: "$count", Value: "total"}},
			},
			"data": mongo.Pipeline{
				{{Key: "$skip", Value: skip}},
				{{Key: "$limit", Value: limit}},
			},
		}}},
	}

	// Execute aggregation
	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		systemContext.Logger.Error("service.MaterialList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve materials", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var results []bson.M
	if err = cursor.All(context.Background(), &results); err != nil {
		systemContext.Logger.Error("service.MaterialList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode materials", nil)
	}

	// Extract data and metadata from facet result
	var materials []bson.M
	var total int64 = 0

	if len(results) > 0 {
		result := results[0]

		// Extract data
		if data, ok := result["data"].(primitive.A); ok {
			for _, item := range data {
				if doc, ok := item.(bson.M); ok {
					materials = append(materials, doc)
				}
			}
		}

		// Extract total count
		if metadata, ok := result["metadata"].(primitive.A); ok && len(metadata) > 0 {
			if metaDoc, ok := metadata[0].(bson.M); ok {
				if count, ok := metaDoc["total"].(int32); ok {
					total = int64(count)
				} else if count, ok := metaDoc["total"].(int64); ok {
					total = count
				}
			}
		}
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	response := &model.MaterialListResponse{
		Data:       materials,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

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

func generateUniqueQuotationName(baseName string, systemContext *model.SystemContext) (string, error) {
	collection := systemContext.MongoDB.Collection("quotation")

	// Try original name first
	filter := bson.M{
		"name":      baseName,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return "", utils.SystemError(enum.ErrorCodeInternal, "Failed to check name uniqueness", nil)
	}

	if count == 0 {
		return baseName, nil
	}

	// If name exists, try with numbers
	for i := 1; i <= 100; i++ {
		newName := fmt.Sprintf("%s (%d)", baseName, i)
		filter["name"] = newName

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return "", utils.SystemError(enum.ErrorCodeInternal, "Failed to check name uniqueness", nil)
		}

		if count == 0 {
			return newName, nil
		}
	}

	return "", utils.SystemError(enum.ErrorCodeValidation, "Unable to generate unique name", nil)
}

func validateAreaMaterials(areaMaterials []database.SystemAreaMaterial, systemContext *model.SystemContext) error {
	materialCollection := systemContext.MongoDB.Collection("material")

	for _, areaMaterial := range areaMaterials {
		for _, materialDetail := range areaMaterial.Materials {
			// Only validate if material ID is provided
			if materialDetail.Material != nil {
				// Check if material exists, is active, not deleted, and belongs to company
				filter := bson.M{
					"_id":       *materialDetail.Material,
					"company":   systemContext.User.Company,
					"status":    enum.MaterialStatusActive,
					"isDeleted": false,
				}

				count, err := materialCollection.CountDocuments(context.Background(), filter)
				if err != nil {
					return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate material", nil)
				}

				if count == 0 {
					return utils.SystemError(
						enum.ErrorCodeValidation,
						"Material not found or not active",
						map[string]interface{}{"materialId": materialDetail.Material.Hex()},
					)
				}
			}
		}
	}

	return nil
}

func calculateQuotationTotals(areaMaterials []database.SystemAreaMaterial, discount database.SystemDiscount) (float64, float64, float64, float64) {
	var totalCost, totalCharge float64

	// First, calculate individual material totals and sum them up
	for i := range areaMaterials {
		for j := range areaMaterials[i].Materials {
			material := &areaMaterials[i].Materials[j]
			
			// Calculate individual material totals
			material.TotalCost = material.CostPerUnit * material.Quantity
			material.TotalPrice = material.PricePerUnit * material.Quantity
			
			// Add to overall totals
			totalCost += material.TotalCost
			totalCharge += material.TotalPrice
		}
	}

	// Calculate discount
	var totalDiscount float64
	switch discount.Type {
	case enum.DiscountTypeRate:
		totalDiscount = totalCharge * (discount.Value / 100)
	case enum.DiscountTypeAmount:
		totalDiscount = discount.Value
	default:
		totalDiscount = 0
	}

	// Calculate net charge (total charge minus discount)
	totalNettCharge := totalCharge - totalDiscount
	if totalNettCharge < 0 {
		totalNettCharge = 0
	}

	return totalCost, totalCharge, totalDiscount, totalNettCharge
}

func quotationCreateValidation(input *database.Quotation, systemContext *model.SystemContext) error {
	// Validate folder exists and belongs to company
	folderCollection := systemContext.MongoDB.Collection("folder")
	filter := bson.M{
		"_id":       input.Folder,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	count, err := folderCollection.CountDocuments(context.Background(), filter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate folder", nil)
	}

	if count == 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "Folder not found", nil)
	}

	// Validate materials in area materials
	if err := validateAreaMaterials(input.AreaMaterials, systemContext); err != nil {
		return err
	}

	// Generate unique name
	uniqueName, err := generateUniqueQuotationName(input.Name, systemContext)
	if err != nil {
		return err
	}

	input.Name = uniqueName
	return nil
}

func QuotationCreate(input *database.Quotation, systemContext *model.SystemContext) (*database.Quotation, error) {
	// Validate input
	if err := quotationCreateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("quotation")

	// Calculate totals
	totalCost, totalCharge, totalDiscount, totalNettCharge := calculateQuotationTotals(input.AreaMaterials, input.Discount)

	// Create quotation object
	quotation := &database.Quotation{
		Folder:          input.Folder,
		Company:         systemContext.User.Company,
		Name:            input.Name,
		Description:     input.Description,
		Remark:          input.Remark,
		AreaMaterials:   input.AreaMaterials,
		Discount:        input.Discount,
		TotalDiscount:   totalDiscount,
		TotalCharge:     totalCharge,
		TotalNettCharge: totalNettCharge,
		TotalCost:       totalCost,
		IsStared:        input.IsStared,
		CreatedAt:       time.Now(),
		CreatedBy:       *systemContext.User.ID,
		UpdatedAt:       time.Now(),
		UpdatedBy:       systemContext.User.ID,
		IsDeleted:       false,
	}

	result, err := collection.InsertOne(context.Background(), quotation)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create quotation", nil)
	}

	quotationID := result.InsertedID.(primitive.ObjectID)

	var doc database.Quotation
	err = collection.FindOne(context.Background(), bson.M{"_id": quotationID}).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve quotation", nil)
	}

	return &doc, nil
}

func quotationUpdateValidation(input *database.Quotation, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("quotation")

	// Check if quotation exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var currentQuotation database.Quotation
	err := collection.FindOne(context.Background(), filter).Decode(&currentQuotation)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeNotFound, "Quotation not found", nil)
	}

	// Validate folder exists and belongs to company
	folderCollection := systemContext.MongoDB.Collection("folder")
	folderFilter := bson.M{
		"_id":       input.Folder,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	count, err := folderCollection.CountDocuments(context.Background(), folderFilter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate folder", nil)
	}

	if count == 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "Folder not found", nil)
	}

	// Validate materials in area materials
	if err := validateAreaMaterials(input.AreaMaterials, systemContext); err != nil {
		return err
	}

	// If name is being changed, generate unique name
	if input.Name != currentQuotation.Name {
		uniqueName, err := generateUniqueQuotationName(input.Name, systemContext)
		if err != nil {
			return err
		}
		input.Name = uniqueName
	}

	return nil
}

func QuotationUpdate(input *database.Quotation, systemContext *model.SystemContext) (*database.Quotation, error) {
	// Validate input
	if err := quotationUpdateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("quotation")

	// Check if quotation exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Quotation
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "quotation not found", nil)
	}

	// Calculate totals
	totalCost, totalCharge, totalDiscount, totalNettCharge := calculateQuotationTotals(input.AreaMaterials, input.Discount)

	// Build update object
	updateFields := bson.M{
		"folder":          input.Folder,
		"name":            input.Name,
		"description":     input.Description,
		"remark":          input.Remark,
		"areaMaterials":   input.AreaMaterials,
		"discount":        input.Discount,
		"totalDiscount":   totalDiscount,
		"totalCharge":     totalCharge,
		"totalNettCharge": totalNettCharge,
		"totalCost":       totalCost,
		"updatedAt":       time.Now(),
		"updatedBy":       systemContext.User.ID,
	}

	update := bson.M{"$set": updateFields}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to update quotation", nil)
	}

	// Return updated quotation
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated quotation", nil)
	}

	return &doc, nil
}

func QuotationGetByID(quotationID primitive.ObjectID, systemContext *model.SystemContext) (*database.Quotation, error) {
	collection := systemContext.MongoDB.Collection("quotation")

	// Build filter
	filter := bson.M{
		"_id":       quotationID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Quotation
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "quotation not found", nil)
	}

	return &doc, nil
}

func QuotationList(input model.QuotationListRequest, systemContext *model.SystemContext) (*model.QuotationListResponse, error) {
	collection := systemContext.MongoDB.Collection("quotation")

	// Build base filter
	filter := bson.M{"isDeleted": false, "company": systemContext.User.Company}

	// Add field-specific filters
	if strings.TrimSpace(input.Name) != "" {
		filter["name"] = primitive.Regex{Pattern: input.Name, Options: "i"}
	}
	if strings.TrimSpace(input.Description) != "" {
		filter["description"] = primitive.Regex{Pattern: input.Description, Options: "i"}
	}
	if input.Folder != nil {
		filter["folder"] = input.Folder
	}
	if input.IsStared != nil {
		filter["isStared"] = *input.IsStared
	}

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"name": searchRegex},
				{"description": searchRegex},
				{"remark": searchRegex},
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

	return executeQuotationList(collection, filter, input, systemContext)
}

func QuotationDelete(input primitive.ObjectID, systemContext *model.SystemContext) (*database.Quotation, error) {
	collection := systemContext.MongoDB.Collection("quotation")

	// Check if quotation exists
	filter := bson.M{
		"_id":       input,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Quotation
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "quotation not found", nil)
	}

	// Soft delete
	update := bson.M{
		"$set": bson.M{
			"isDeleted": true,
			"updatedAt": time.Now(),
			"updatedBy": systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to delete quotation", nil)
	}

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	return &doc, nil
}

func QuotationToggleStar(quotationID primitive.ObjectID, isStared bool, systemContext *model.SystemContext) (*database.Quotation, error) {
	collection := systemContext.MongoDB.Collection("quotation")

	// Check if quotation exists
	filter := bson.M{
		"_id":       quotationID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Quotation
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "quotation not found", nil)
	}

	// Toggle star
	update := bson.M{
		"$set": bson.M{
			"isStared":  isStared,
			"updatedAt": time.Now(),
			"updatedBy": systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to toggle star", nil)
	}

	// Return updated quotation
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated quotation", nil)
	}

	return &doc, nil
}

// Helper functions
func executeQuotationList(collection *mongo.Collection, filter bson.M, input model.QuotationListRequest, systemContext *model.SystemContext) (*model.QuotationListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.QuotationList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count quotations", nil)
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
		systemContext.Logger.Error("service.QuotationList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve quotations", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var quotations []bson.M
	if err = cursor.All(context.Background(), &quotations); err != nil {
		systemContext.Logger.Error("service.QuotationList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode quotations", nil)
	}

	response := &model.QuotationListResponse{
		Data:       quotations,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

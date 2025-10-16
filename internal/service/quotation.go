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
	totalCharge, totalDiscount, totalAdditionalCharge, totalNettCharge := calculateQuotationTotals(input.AreaMaterials, input.Discounts, input.AdditionalCharges)

	// Create quotation object
	quotation := &database.Quotation{
		Folder:                input.Folder,
		Company:               systemContext.User.Company,
		Name:                  input.Name,
		ExpiredAt:             input.ExpiredAt,
		Description:           input.Description,
		Remark:                input.Remark,
		AreaMaterials:         input.AreaMaterials,
		Discounts:             input.Discounts,
		AdditionalCharges:     input.AdditionalCharges,
		TotalCharge:           totalCharge,
		TotalDiscount:         totalDiscount,
		TotalAdditionalCharge: totalAdditionalCharge,
		TotalNettCharge:       totalNettCharge,
		IsStared:              input.IsStared,
		CreatedAt:             time.Now(),
		CreatedBy:             *systemContext.User.ID,
		UpdatedAt:             time.Now(),
		UpdatedBy:             systemContext.User.ID,
		IsDeleted:             false,
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
	totalCharge, totalDiscount, totalAdditionalCharge, totalNettCharge := calculateQuotationTotals(input.AreaMaterials, input.Discounts, input.AdditionalCharges)

	// Build update object
	updateFields := bson.M{
		"folder":                input.Folder,
		"name":                  input.Name,
		"expiredAt":             input.ExpiredAt,
		"description":           input.Description,
		"remark":                input.Remark,
		"areaMaterials":         input.AreaMaterials,
		"discounts":             input.Discounts,
		"additionalCharges":     input.AdditionalCharges,
		"totalCharge":           totalCharge,
		"totalDiscount":         totalDiscount,
		"totalAdditionalCharge": totalAdditionalCharge,
		"totalNettCharge":       totalNettCharge,
		"updatedAt":             time.Now(),
		"updatedBy":             systemContext.User.ID,
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
	// Check if user has a company
	if systemContext.User.Company == nil {
		return &model.QuotationListResponse{
			Data:       []bson.M{},
			Page:       1,
			Limit:      10,
			Total:      0,
			TotalPages: 0,
		}, nil
	}

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

func validateMaterialDetail(materialDetail database.SystemAreaMaterialDetail, materialCollection *mongo.Collection, systemContext *model.SystemContext) error {
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

	// Recursively validate template materials
	for _, templateMaterial := range materialDetail.Template {
		if err := validateMaterialDetail(templateMaterial, materialCollection, systemContext); err != nil {
			return err
		}
	}

	return nil
}

func validateAreaMaterials(areaMaterials []database.SystemAreaMaterial, systemContext *model.SystemContext) error {
	materialCollection := systemContext.MongoDB.Collection("material")

	for _, areaMaterial := range areaMaterials {
		for _, materialDetail := range areaMaterial.Materials {
			if err := validateMaterialDetail(materialDetail, materialCollection, systemContext); err != nil {
				return err
			}
		}
	}

	return nil
}

func calculateQuotationTotals(areaMaterials []database.SystemAreaMaterial, discounts []database.SystemDiscount, additionalCharges []database.SystemAdditionalCharge) (float64, float64, float64, float64) {
	var totalCharge float64

	// Sum up area subtotals (use SubTotal values from payload as-is)
	for i := range areaMaterials {
		// Add area SubTotal to overall total charge
		totalCharge += areaMaterials[i].SubTotal
	}

	// Calculate total discounts
	totalDiscount := calculateDiscounts(discounts, totalCharge)

	// Calculate total additional charges
	totalAdditionalCharge := calculateAdditionalCharges(additionalCharges, totalCharge)

	// Calculate net charge (total charge - discount + additional charges)
	totalNettCharge := totalCharge - totalDiscount + totalAdditionalCharge
	if totalNettCharge < 0 {
		totalNettCharge = 0
	}

	return totalCharge, totalDiscount, totalAdditionalCharge, totalNettCharge
}

func calculateDiscounts(discounts []database.SystemDiscount, totalCharge float64) float64 {
	var totalDiscount float64

	for _, discount := range discounts {
		switch discount.Type {
		case enum.DiscountTypeRate:
			totalDiscount += totalCharge * (discount.Value / 100)
		case enum.DiscountTypeAmount:
			totalDiscount += discount.Value
		}
	}

	return totalDiscount
}

func calculateAdditionalCharges(charges []database.SystemAdditionalCharge, totalCharge float64) float64 {
	var totalAdditionalCharge float64

	for _, charge := range charges {
		switch charge.Type {
		case enum.AdditionalChargeTypeRate:
			totalAdditionalCharge += totalCharge * (charge.Value / 100)
		case enum.AdditionalChargeTypeAmount:
			totalAdditionalCharge += charge.Value
		}
	}

	return totalAdditionalCharge
}

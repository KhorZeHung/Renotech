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
	// Validate folder exists and belongs to company (only if folder is provided)
	if input.Folder != nil {
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
			fmt.Println("here")
			return utils.SystemError(enum.ErrorCodeValidation, "Folder not found", nil)
		}
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
		Client:                input.Client,
		Budget:                input.Budget,
		Address:               input.Address,
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

	// Validate folder exists and belongs to company (only if folder is provided)
	if input.Folder != nil {
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
		"client":                input.Client,
		"budget":                input.Budget,
		"address":               input.Address,
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

func quotationCreateFolderValidation(input *model.QuotationCreateFolderRequest, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("quotation")

	// Check if quotation exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate quotation", nil)
	}

	if count == 0 {
		return utils.SystemError(enum.ErrorCodeNotFound, "Quotation not found", nil)
	}

	return nil
}

func QuotationCreateFolder(input *model.QuotationCreateFolderRequest, systemContext *model.SystemContext) (*database.Folder, error) {
	// Validate input
	if err := quotationCreateFolderValidation(input, systemContext); err != nil {
		return nil, err
	}

	// Get quotation
	quotation, err := QuotationGetByID(input.ID, systemContext)
	if err != nil {
		return nil, err
	}

	// Extract unique areas from quotation's AreaMaterials
	areaMap := make(map[string]database.SystemArea)
	for _, areaMaterial := range quotation.AreaMaterials {
		areaMap[areaMaterial.Area.Name] = areaMaterial.Area
	}

	var areas []database.SystemArea
	for _, area := range areaMap {
		areas = append(areas, area)
	}

	// Create folder from quotation data
	folderInput := &database.Folder{
		Name:        input.Name,
		Client:      quotation.Client,
		Budget:      quotation.Budget,
		Address:     quotation.Address,
		Description: quotation.Description,
		Remark:      quotation.Remark,
		Media:       quotation.Media,
		Areas:       areas,
		Status:      "", // Default empty status
	}

	// Create folder using FolderCreate service
	folder, err := FolderCreate(folderInput, systemContext)
	if err != nil {
		return nil, err
	}

	// Update quotation's folder field
	collection := systemContext.MongoDB.Collection("quotation")
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	update := bson.M{
		"$set": bson.M{
			"folder":    folder.ID,
			"updatedAt": time.Now(),
			"updatedBy": systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update quotation folder", nil)
	}

	return folder, nil
}

func quotationMoveValidation(input *model.QuotationMoveRequest, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("quotation")

	// Check if quotation exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate quotation", nil)
	}

	if count == 0 {
		return utils.SystemError(enum.ErrorCodeNotFound, "Quotation not found", nil)
	}

	// Validate target folder exists
	folderCollection := systemContext.MongoDB.Collection("folder")
	folderFilter := bson.M{
		"_id":       input.Folder,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	folderCount, err := folderCollection.CountDocuments(context.Background(), folderFilter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate folder", nil)
	}

	if folderCount == 0 {
		return utils.SystemError(enum.ErrorCodeNotFound, "Folder not found", nil)
	}

	return nil
}

func QuotationMove(input *model.QuotationMoveRequest, systemContext *model.SystemContext) (*database.Quotation, error) {
	// Validate input
	if err := quotationMoveValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("quotation")

	// Update quotation's folder field
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	update := bson.M{
		"$set": bson.M{
			"folder":    input.Folder,
			"updatedAt": time.Now(),
			"updatedBy": systemContext.User.ID,
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to move quotation", nil)
	}

	// Return updated quotation
	var doc database.Quotation
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated quotation", nil)
	}

	return &doc, nil
}

func quotationDuplicateValidation(input *primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("quotation")

	// Check if quotation exists
	filter := bson.M{
		"_id":       input,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate quotation", nil)
	}

	if count == 0 {
		return utils.SystemError(enum.ErrorCodeNotFound, "Quotation not found", nil)
	}

	return nil
}

func QuotationDuplicate(input *primitive.ObjectID, systemContext *model.SystemContext) (*database.Quotation, error) {
	// Validate input
	if err := quotationDuplicateValidation(input, systemContext); err != nil {
		return nil, err
	}

	// Get original quotation
	original, err := QuotationGetByID(*input, systemContext)
	if err != nil {
		return nil, err
	}

	// Generate new name with " (copy)" suffix
	baseName := original.Name + " (copy)"
	uniqueName, err := generateUniqueQuotationName(baseName, systemContext)
	if err != nil {
		return nil, err
	}

	// Calculate totals
	totalCharge, totalDiscount, totalAdditionalCharge, totalNettCharge := calculateQuotationTotals(
		original.AreaMaterials,
		original.Discounts,
		original.AdditionalCharges,
	)

	// Create new quotation with duplicated data
	newQuotation := &database.Quotation{
		Folder:                original.Folder,
		Company:               systemContext.User.Company,
		Name:                  uniqueName,
		Client:                original.Client,
		Budget:                original.Budget,
		Address:               original.Address,
		ExpiredAt:             original.ExpiredAt,
		Description:           original.Description,
		Remark:                original.Remark,
		AreaMaterials:         original.AreaMaterials,
		Discounts:             original.Discounts,
		AdditionalCharges:     original.AdditionalCharges,
		TotalCharge:           totalCharge,
		TotalDiscount:         totalDiscount,
		TotalAdditionalCharge: totalAdditionalCharge,
		TotalNettCharge:       totalNettCharge,
		Media:                 original.Media,
		IsStared:              false, // Reset star status
		CreatedAt:             time.Now(),
		CreatedBy:             *systemContext.User.ID,
		UpdatedAt:             time.Now(),
		UpdatedBy:             systemContext.User.ID,
		IsDeleted:             false,
	}

	collection := systemContext.MongoDB.Collection("quotation")
	result, err := collection.InsertOne(context.Background(), newQuotation)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to duplicate quotation", nil)
	}

	quotationID := result.InsertedID.(primitive.ObjectID)

	var duplicatedDoc database.Quotation
	err = collection.FindOne(context.Background(), bson.M{"_id": quotationID}).Decode(&duplicatedDoc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve duplicated quotation", nil)
	}

	return &duplicatedDoc, nil
}

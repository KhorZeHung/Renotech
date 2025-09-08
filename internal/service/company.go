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
func companyTenantCreateValidation(input *database.Company, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("company")

	// Validate logo path if provided
	if err := utils.ValidateFilePath(input.Logo); err != nil {
		return err
	}

	// Check for duplicate company name (tenant scope)
	if strings.TrimSpace(input.Name) != "" {
		filter := bson.M{
			"name":      input.Name,
			"isDeleted": false,
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate company name", nil)
		}

		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Company name already exists",
				map[string]interface{}{"name": input.Name},
			)
		}
	}

	if len(input.ClientDisplayName) < 1 {
		input.ClientDisplayName = input.Name
	}

	if len(input.SupplierDisplayName) < 1 {
		input.SupplierDisplayName = input.Name
	}

	// Create company object with default values
	input.Owner = systemContext.User.ID
	input.IsDeleted = false
	input.IsEnabled = true
	input.CreatedAt = time.Now()
	input.CreatedBy = systemContext.User.ID
	input.UpdatedAt = time.Now()
	input.UpdatedBy = systemContext.User.ID

	return nil
}

func CompanyTenantCreate(input *database.Company, systemContext *model.SystemContext) (*database.Company, error) {
	// Validate input
	if err := companyTenantCreateValidation(input, systemContext); err != nil {
		return nil, err
	}
	collection := systemContext.MongoDB.Collection("company")
	userColl := systemContext.MongoDB.Collection("user")

	result, err := collection.InsertOne(context.Background(), input)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create company", nil)
	}

	companyID := result.InsertedID.(primitive.ObjectID)

	var doc database.Company

	err = collection.FindOne(context.Background(), bson.M{"_id": companyID}).Decode(&doc)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve company", nil)
	}

	// sync owner to user's company field
	filter := bson.M{
		"_id": systemContext.User.ID,
	}

	update := bson.M{
		"$set": bson.M{
			"company":   companyID,
			"updatedAt": time.Now(),
		},
	}

	_, err = userColl.UpdateOne(context.Background(), filter, update)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve company", nil)
	}

	return &doc, nil
}

func companyUpdateValidation(input *database.Company, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("company")

	filter := bson.M{
		"_id":       input.ID,
		"isDeleted": false,
	}

	var doc database.Company

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	if doc.ID == nil {
		return utils.SystemError(enum.ErrorCodeUnauthorized, "company not found", nil)
	}

	// Validate logo path if provided
	if err := utils.ValidateFilePath(input.Logo); err != nil {
		return err
	}

	// Check for duplicate company name (excluding current company)
	if strings.TrimSpace(input.Name) != "" && input.ID != nil {
		filter := bson.M{
			"name":      input.Name,
			"isDeleted": false,
			"_id":       bson.M{"$ne": input.ID}, // Exclude current company
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate company name", nil)
		}

		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Company name already exists",
				map[string]interface{}{"name": input.Name},
			)
		}
	}

	return nil
}

func CompanyTenantUpdate(input *database.Company, systemContext *model.SystemContext) (*database.Company, error) {
	// Validate input
	err := companyUpdateValidation(input, systemContext)

	if err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("company")

	filter := bson.M{
		"_id":       input.ID,
		"isDeleted": false,
	}

	value := bson.M{
		"$set": bson.M{
			"name":                input.Name,
			"clientDisplayName":   input.ClientDisplayName,
			"supplierDisplayName": input.SupplierDisplayName,
			"address":             input.Address,
			"website":             input.Website,
			"owner":               input.Owner,
			"logo":                input.Logo,
			"contact":             input.Contact,
			"termCondition":       input.TermCondition,
			"isEnabled":           input.IsEnabled,
			"updatedAt":           time.Now(),
			"updatedBy":           systemContext.User.ID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, value)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to update company", err.Error())
	}

	var doc database.Company

	// return company
	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	return &doc, nil
}

func CompanyTenantGet(systemContext *model.SystemContext) (*database.Company, error) {
	// Get company by ID for tenant users
	collection := systemContext.MongoDB.Collection("company")

	filter := bson.M{
		"_id":       systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Company

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	if doc.ID == nil {
		return nil, utils.SystemError(enum.ErrorCodeUnauthorized, "company not found", nil)
	}

	return &doc, nil
}

// Admin services
func companyAdminCreateValidation(input *database.Company, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("company")

	// Validate logo path if provided
	if err := utils.ValidateFilePath(input.Logo); err != nil {
		return err
	}

	// Check for duplicate company name (global scope)
	if strings.TrimSpace(input.Name) != "" {
		filter := bson.M{
			"name":      input.Name,
			"isDeleted": false,
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicate company name", nil)
		}

		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Company name already exists",
				map[string]interface{}{"name": input.Name},
			)
		}
	}

	return nil
}

func CompanyAdminCreate(input *database.Company, systemContext *model.SystemContext) (*database.Company, error) {
	// Validate input
	if err := companyAdminCreateValidation(input, systemContext); err != nil {
		return nil, err
	}
	collection := systemContext.MongoDB.Collection("company")

	// Create company object with default values
	company := &database.Company{
		Name:                input.Name,
		ClientDisplayName:   input.ClientDisplayName,
		SupplierDisplayName: input.SupplierDisplayName,
		Address:             input.Address,
		Website:             input.Website,
		Owner:               input.Owner, // Allow admin to set any owner
		Logo:                input.Logo,
		Contact:             input.Contact,
		TermCondition:       input.TermCondition,
		IsDeleted:           false,
		IsEnabled:           true,
		CreatedAt:           time.Now(),
		CreatedBy:           systemContext.User.ID,
		UpdatedAt:           time.Now(),
		UpdatedBy:           systemContext.User.ID,
	}

	result, err := collection.InsertOne(context.Background(), company)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create company", nil)
	}

	companyID := result.InsertedID.(primitive.ObjectID)

	var doc database.Company
	err = collection.FindOne(context.Background(), bson.M{"_id": companyID}).Decode(&doc)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve company", nil)
	}

	return &doc, nil
}

func CompanyAdminUpdate(input *database.Company, systemContext *model.SystemContext) (*database.Company, error) {
	// Validate input
	if err := companyUpdateValidation(input, systemContext); err != nil {
		return nil, err
	}

	// check if company exists
	collection := systemContext.MongoDB.Collection("company")

	filter := bson.M{
		"_id":       input.ID,
		"isDeleted": false,
	}

	var doc database.Company

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	if doc.ID == nil {
		return nil, utils.SystemError(enum.ErrorCodeUnauthorized, "company not found", nil)
	}

	// update company object
	value := bson.M{
		"$set": bson.M{
			"name":                input.Name,
			"clientDisplayName":   input.ClientDisplayName,
			"supplierDisplayName": input.SupplierDisplayName,
			"address":             input.Address,
			"website":             input.Website,
			"owner":               input.Owner,
			"logo":                input.Logo,
			"contact":             input.Contact,
			"termCondition":       input.TermCondition,
			"isEnabled":           input.IsEnabled,
			"updatedAt":           time.Now(),
			"updatedBy":           systemContext.User.ID,
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, value)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to update company", err.Error())
	}

	// return company
	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	return &doc, nil
}

func CompanyAdminList(input model.CompanyListRequest, systemContext *model.SystemContext) (*model.CompanyListResponse, error) {
	collection := systemContext.MongoDB.Collection("company")

	// Build base filter - admin can see all companies
	filter := bson.M{"isDeleted": false}

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
	if strings.TrimSpace(input.Address) != "" {
		filter["address"] = primitive.Regex{Pattern: input.Address, Options: "i"}
	}
	if strings.TrimSpace(input.Website) != "" {
		filter["website"] = primitive.Regex{Pattern: input.Website, Options: "i"}
	}
	if strings.TrimSpace(input.Contact) != "" {
		filter["contact"] = primitive.Regex{Pattern: input.Contact, Options: "i"}
	}
	if strings.TrimSpace(input.Owner) != "" {
		if ownerID, err := primitive.ObjectIDFromHex(input.Owner); err == nil {
			filter["owner"] = ownerID
		}
	}
	if input.IsEnabled != nil {
		filter["isEnabled"] = *input.IsEnabled
	}

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"name": searchRegex},
				{"clientDisplayName": searchRegex},
				{"supplierDisplayName": searchRegex},
				{"address": searchRegex},
				{"website": searchRegex},
				{"contact": searchRegex},
			},
		}

		// Combine existing filter with search filter
		if len(filter) > 1 { // More than just isDeleted
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

	return executeCompanyList(collection, filter, input, systemContext)
}

// shared service
func CompanyDelete(input *primitive.ObjectID, systemContext *model.SystemContext) (*database.Company, error) {
	// check if company exists
	collection := systemContext.MongoDB.Collection("company")

	filter := bson.M{
		"_id":       input,
		"isDeleted": false,
	}

	var doc database.Company

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	if doc.ID == nil {
		return nil, utils.SystemError(enum.ErrorCodeUnauthorized, "company not found", nil)
	}

	// update company object
	value := bson.M{
		"$set": bson.M{
			"isDeleted": true,
			"updatedAt": time.Now(),
			"updatedBy": systemContext.User.ID,
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, value)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to update company", err.Error())
	}

	// return company
	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	return &doc, nil
}

// Helper functions
func executeCompanyList(collection *mongo.Collection, filter bson.M, input model.CompanyListRequest, systemContext *model.SystemContext) (*model.CompanyListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.CompanyList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count companies", nil)
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
		systemContext.Logger.Error("service.CompanyList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve companies", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var companies []bson.M
	if err = cursor.All(context.Background(), &companies); err != nil {
		systemContext.Logger.Error("service.CompanyList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode companies", nil)
	}

	response := &model.CompanyListResponse{
		Data:       companies,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

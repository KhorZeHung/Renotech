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

func createActionLog(description string, systemContext *model.SystemContext) database.SystemActionLog {
	return database.SystemActionLog{
		Description: description,
		Time:        time.Now(),
		ByName:      systemContext.User.Username,
		ById:        systemContext.User.ID,
	}
}

func calculateProjectTotals(areaMaterials []database.SystemAreaMaterial, discount database.SystemDiscount) (float64, float64, float64, float64) {
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

func validatePICUsers(picIDs []primitive.ObjectID, systemContext *model.SystemContext) error {
	if len(picIDs) == 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "At least one PIC is required", nil)
	}

	userCollection := systemContext.MongoDB.Collection("user")

	for _, picID := range picIDs {
		filter := bson.M{
			"_id":       picID,
			"company":   systemContext.User.Company,
			"isDeleted": false,
		}

		count, err := userCollection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate PIC user", nil)
		}

		if count == 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"PIC user not found or not in same company",
				map[string]interface{}{"userId": picID.Hex()},
			)
		}
	}

	return nil
}

func projectCreateValidation(input *model.ProjectCreateRequest, systemContext *model.SystemContext) (*database.Quotation, error) {
	// Validate quotation exists and belongs to company
	quotationCollection := systemContext.MongoDB.Collection("quotation")
	quotationFilter := bson.M{
		"_id":       input.QuotationID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var quotation database.Quotation
	err := quotationCollection.FindOne(context.Background(), quotationFilter).Decode(&quotation)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Quotation not found", nil)
	}

	// Check if folder already has a project
	projectCollection := systemContext.MongoDB.Collection("project")
	projectFilter := bson.M{
		"folder":    quotation.Folder,
		"isDeleted": false,
	}

	count, err := projectCollection.CountDocuments(context.Background(), projectFilter)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to check existing project", nil)
	}

	if count > 0 {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Folder already has a project", nil)
	}

	// Validate PIC users
	if err := validatePICUsers(input.PIC, systemContext); err != nil {
		return nil, err
	}

	return &quotation, nil
}

func ProjectCreate(input *model.ProjectCreateRequest, systemContext *model.SystemContext) (*database.Project, error) {
	// Validate input and get quotation
	quotation, err := projectCreateValidation(input, systemContext)
	if err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("project")

	// Create initial action log
	actionLogs := []database.SystemActionLog{
		createActionLog(fmt.Sprintf("Project created from quotation: %s", quotation.Name), systemContext),
	}

	// Create project object from quotation data
	project := &database.Project{
		Folder:              quotation.Folder,
		Quotation:           input.QuotationID,
		Description:         quotation.Description,
		Remark:              quotation.Remark,
		AreaMaterials:       quotation.AreaMaterials,
		Discount:            quotation.Discount,
		TotalDiscount:       quotation.TotalDiscount,
		TotalCharge:         quotation.TotalCharge,
		TotalNettCharge:     quotation.TotalNettCharge,
		TotalCost:           quotation.TotalCost,
		Company:             quotation.Company,
		CreatedAt:           time.Now(),
		CreatedBy:           *systemContext.User.ID,
		UpdatedAt:           time.Now(),
		UpdatedBy:           systemContext.User.ID,
		EstimatedCompleteAt: input.EstimatedCompleteAt,
		ActionLogs:          actionLogs,
		PIC:                 input.PIC,
		IsDeleted:           false,
	}

	result, err := collection.InsertOne(context.Background(), project)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create project", nil)
	}

	projectID := result.InsertedID.(primitive.ObjectID)

	var doc database.Project
	err = collection.FindOne(context.Background(), bson.M{"_id": projectID}).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve project", nil)
	}

	return &doc, nil
}

func projectUpdateValidation(input *model.ProjectUpdateRequest, systemContext *model.SystemContext) (*database.Project, error) {
	collection := systemContext.MongoDB.Collection("project")

	// Check if project exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var currentProject database.Project
	err := collection.FindOne(context.Background(), filter).Decode(&currentProject)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Project not found", nil)
	}

	// Validate PIC users
	if err := validatePICUsers(input.PIC, systemContext); err != nil {
		return nil, err
	}

	// Validate quotation if provided and changed
	if !input.Quotation.IsZero() && input.Quotation != currentProject.Quotation {
		quotationCollection := systemContext.MongoDB.Collection("quotation")
		quotationFilter := bson.M{
			"_id":       input.Quotation,
			"company":   systemContext.User.Company,
			"isDeleted": false,
		}

		var quotation database.Quotation
		err := quotationCollection.FindOne(context.Background(), quotationFilter).Decode(&quotation)
		if err != nil {
			return nil, utils.SystemError(enum.ErrorCodeNotFound, "Quotation not found", nil)
		}

		// Ensure quotation belongs to the same folder as the project
		if quotation.Folder != currentProject.Folder {
			return nil, utils.SystemError(enum.ErrorCodeValidation, "Quotation must belong to the same folder as the project", nil)
		}
	}

	return &currentProject, nil
}

func ProjectUpdate(input *model.ProjectUpdateRequest, systemContext *model.SystemContext) (*database.Project, error) {
	// Validate input
	currentProject, err := projectUpdateValidation(input, systemContext)
	if err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("project")

	// Build action logs for changes
	actionLogs := currentProject.ActionLogs

	if input.Description != currentProject.Description {
		actionLogs = append(actionLogs, createActionLog("Project description updated", systemContext))
	}
	if input.Remark != currentProject.Remark {
		actionLogs = append(actionLogs, createActionLog("Project remark updated", systemContext))
	}
	if !input.EstimatedCompleteAt.Equal(currentProject.EstimatedCompleteAt) {
		actionLogs = append(actionLogs, createActionLog(fmt.Sprintf("Estimated completion date changed to %s", input.EstimatedCompleteAt.Format("2006-01-02")), systemContext))
	}
	if !input.Quotation.IsZero() && input.Quotation != currentProject.Quotation {
		actionLogs = append(actionLogs, createActionLog("Project quotation updated", systemContext))
	}

	// Check if PIC changed
	picChanged := len(input.PIC) != len(currentProject.PIC)
	if !picChanged {
		picMap := make(map[string]bool)
		for _, pic := range currentProject.PIC {
			picMap[pic.Hex()] = true
		}
		for _, pic := range input.PIC {
			if !picMap[pic.Hex()] {
				picChanged = true
				break
			}
		}
	}
	if picChanged {
		actionLogs = append(actionLogs, createActionLog("Project PIC updated", systemContext))
	}

	// Determine which fields to update
	updateFields := bson.M{
		"description":         input.Description,
		"remark":              input.Remark,
		"pic":                 input.PIC,
		"estimatedCompleteAt": input.EstimatedCompleteAt,
		"actionLogs":          actionLogs,
		"updatedAt":           time.Now(),
		"updatedBy":           systemContext.User.ID,
	}

	// Update quotation if provided
	if !input.Quotation.IsZero() {
		updateFields["quotation"] = input.Quotation
	}

	// Calculate and update financial fields if areaMaterials or discount provided
	if len(input.AreaMaterials) > 0 || input.Discount.Type != "" {
		// Use provided areaMaterials or current ones
		areaMaterials := input.AreaMaterials
		if len(areaMaterials) == 0 {
			areaMaterials = currentProject.AreaMaterials
		}

		// Use provided discount or current one
		discount := input.Discount
		if discount.Type == "" {
			discount = currentProject.Discount
		}

		// Calculate new totals
		totalCost, totalCharge, totalDiscount, totalNettCharge := calculateProjectTotals(areaMaterials, discount)

		// Update financial fields
		updateFields["areaMaterials"] = areaMaterials
		updateFields["discount"] = discount
		updateFields["totalCost"] = totalCost
		updateFields["totalCharge"] = totalCharge
		updateFields["totalDiscount"] = totalDiscount
		updateFields["totalNettCharge"] = totalNettCharge

		// Add action log for financial changes
		actionLogs = append(actionLogs, createActionLog("Project costs and materials updated", systemContext))
		updateFields["actionLogs"] = actionLogs
	}

	// Build update object
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	update := bson.M{"$set": updateFields}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to update project", nil)
	}

	// Return updated project
	var doc database.Project
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated project", nil)
	}

	return &doc, nil
}

func ProjectGetByID(projectID primitive.ObjectID, systemContext *model.SystemContext) (*database.Project, error) {
	collection := systemContext.MongoDB.Collection("project")

	// Build filter
	filter := bson.M{
		"_id":       projectID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Project
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "project not found", nil)
	}

	return &doc, nil
}

func ProjectList(input model.ProjectListRequest, systemContext *model.SystemContext) (*model.ProjectListResponse, error) {
	collection := systemContext.MongoDB.Collection("project")

	// Build base filter
	filter := bson.M{"isDeleted": false, "company": systemContext.User.Company}

	// Add field-specific filters
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

	return executeProjectList(collection, filter, input, systemContext)
}

func ProjectDelete(input primitive.ObjectID, systemContext *model.SystemContext) (*database.Project, error) {
	collection := systemContext.MongoDB.Collection("project")

	// Check if project exists
	filter := bson.M{
		"_id":       input,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Project
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "project not found", nil)
	}

	// Add deletion action log
	actionLogs := doc.ActionLogs
	actionLogs = append(actionLogs, createActionLog("Project deleted", systemContext))

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
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to delete project", nil)
	}

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	return &doc, nil
}

// Helper functions
func executeProjectList(collection *mongo.Collection, filter bson.M, input model.ProjectListRequest, systemContext *model.SystemContext) (*model.ProjectListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.ProjectList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count projects", nil)
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
		systemContext.Logger.Error("service.ProjectList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve projects", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var projects []bson.M
	if err = cursor.All(context.Background(), &projects); err != nil {
		systemContext.Logger.Error("service.ProjectList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode projects", nil)
	}

	response := &model.ProjectListResponse{
		Data:       projects,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

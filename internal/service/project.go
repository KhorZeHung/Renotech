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

func projectCreateValidation(input *database.Project, systemContext *model.SystemContext) error {
	// Validate materials in area materials (reuse from quotation)
	if err := validateAreaMaterials(input.AreaMaterials, systemContext); err != nil {
		return err
	}

	// Generate unique name (auto-adds number if duplicate)
	uniqueName, err := generateUniqueProjectName(input.Name, systemContext)
	if err != nil {
		return err
	}

	input.Name = uniqueName
	return nil
}

func ProjectCreate(input *database.Project, systemContext *model.SystemContext) (*database.Project, error) {
	// Validate input
	if err := projectCreateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("project")

	// Calculate totals (reuse quotation calculation)
	totalCharge, totalDiscount, totalAdditionalCharge, totalNettCharge := calculateQuotationTotals(input.AreaMaterials, input.Discounts, input.AdditionalCharges)

	// Create initial action log
	actionLogs := []database.SystemActionLog{
		{
			Description: fmt.Sprintf("Project created by %s", systemContext.User.Username),
			Time:        time.Now(),
			ByName:      systemContext.User.Username,
			ById:        systemContext.User.ID,
		},
	}

	// Create project object
	project := &database.Project{
		Name:                  input.Name,
		Client:                input.Client,
		Budget:                input.Budget,
		Address:               input.Address,
		Description:           input.Description,
		Remark:                input.Remark,
		AreaMaterials:         input.AreaMaterials,
		TermCondition:         input.TermCondition,
		Discounts:             input.Discounts,
		AdditionalCharges:     input.AdditionalCharges,
		TotalCharge:           totalCharge,
		TotalDiscount:         totalDiscount,
		TotalAdditionalCharge: totalAdditionalCharge,
		TotalNettCharge:       totalNettCharge,
		Media:                 input.Media,
		MaterialSummary:       []database.ProjectMaterialSummary{}, // Initialize empty
		EstimatedCompleteAt:   input.EstimatedCompleteAt,
		ActionLogs:            actionLogs,
		Pic:                   input.Pic,
		Company:               systemContext.User.Company,
		IsDeleted:             false,
		CreatedAt:             time.Now(),
		CreatedBy:             *systemContext.User.ID,
		UpdatedAt:             time.Now(),
		UpdatedBy:             systemContext.User.ID,
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

func projectConvertFromQuotationValidation(input *model.ProjectConvertRequest, systemContext *model.SystemContext) (*database.Quotation, error) {
	// Validate quotation exists and belongs to company
	quotationCollection := systemContext.MongoDB.Collection("quotation")
	filter := bson.M{
		"_id":       input.QuotationID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var quotation database.Quotation
	err := quotationCollection.FindOne(context.Background(), filter).Decode(&quotation)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Quotation not found", nil)
	}

	// Validate PIC users exist and belong to company
	if len(input.PIC) == 0 {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "At least one PIC is required", nil)
	}

	userCollection := systemContext.MongoDB.Collection("user")
	for i, picID := range input.PIC {
		var userDoc database.User
		err := userCollection.FindOne(context.Background(), bson.M{
			"_id":       picID,
			"company":   systemContext.User.Company,
			"isDeleted": false,
		}).Decode(&userDoc)

		if err != nil {
			return nil, utils.SystemError(
				enum.ErrorCodeValidation,
				"PIC user not found or does not belong to your company",
				map[string]interface{}{"picIndex": i, "picId": picID.Hex()},
			)
		}
	}

	// Validate estimated completion date is in the future
	if input.EstimatedCompleteAt.Before(time.Now()) {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Estimated completion date must be in the future", nil)
	}

	return &quotation, nil
}

func ProjectConvertFromQuotation(input *model.ProjectConvertRequest, systemContext *model.SystemContext) (*database.Project, error) {
	// Validate input
	quotation, err := projectConvertFromQuotationValidation(input, systemContext)
	if err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("project")

	// Generate unique project name based on quotation name
	uniqueName, err := generateUniqueProjectName(quotation.Name, systemContext)
	if err != nil {
		return nil, err
	}

	// Create initial action log
	actionLogs := []database.SystemActionLog{
		{
			Description: fmt.Sprintf("Project created from quotation: %s", quotation.Name),
			Time:        time.Now(),
			ByName:      systemContext.User.Username,
			ById:        systemContext.User.ID,
		},
	}

	// Create project from quotation
	project := &database.Project{
		Name:                  uniqueName,
		Client:                quotation.Client,
		Budget:                quotation.Budget,
		Address:               quotation.Address,
		Description:           quotation.Description,
		Remark:                quotation.Remark,
		AreaMaterials:         quotation.AreaMaterials,
		TermCondition:         quotation.TermCondition,
		Discounts:             quotation.Discounts,
		AdditionalCharges:     quotation.AdditionalCharges,
		TotalCharge:           quotation.TotalCharge,
		TotalDiscount:         quotation.TotalDiscount,
		TotalAdditionalCharge: quotation.TotalAdditionalCharge,
		TotalNettCharge:       quotation.TotalNettCharge,
		Media:                 quotation.Media,
		MaterialSummary:       []database.ProjectMaterialSummary{}, // Initialize empty
		EstimatedCompleteAt:   input.EstimatedCompleteAt,
		ActionLogs:            actionLogs,
		Pic:                   input.PIC,
		Company:               systemContext.User.Company,
		IsDeleted:             false,
		CreatedAt:             time.Now(),
		CreatedBy:             *systemContext.User.ID,
		UpdatedAt:             time.Now(),
		UpdatedBy:             systemContext.User.ID,
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

func projectUpdateValidation(input *model.ProjectUpdateRequest, systemContext *model.SystemContext) error {
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
		return utils.SystemError(enum.ErrorCodeNotFound, "Project not found", nil)
	}

	// Validate materials in area materials (reuse quotation validation)
	if err := validateAreaMaterials(input.AreaMaterials, systemContext); err != nil {
		return err
	}

	// Validate PIC users exist and belong to company
	if len(input.Pic) == 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "At least one PIC is required", nil)
	}

	userCollection := systemContext.MongoDB.Collection("user")
	for i, picID := range input.Pic {
		var userDoc database.User
		err := userCollection.FindOne(context.Background(), bson.M{
			"_id":       picID,
			"company":   systemContext.User.Company,
			"isDeleted": false,
		}).Decode(&userDoc)

		if err != nil {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"PIC user not found or does not belong to your company",
				map[string]interface{}{"picIndex": i, "picId": picID.Hex()},
			)
		}
	}

	// If name is being changed, generate unique name
	if input.Name != currentProject.Name {
		uniqueName, err := generateUniqueProjectName(input.Name, systemContext)
		if err != nil {
			return err
		}
		input.Name = uniqueName
	}

	return nil
}

// Helper function to generate action logs for project updates
func generateProjectUpdateActionLogs(oldProject *database.Project, input *model.ProjectUpdateRequest, systemContext *model.SystemContext) []database.SystemActionLog {
	var changes []string

	// Check name change
	if input.Name != oldProject.Name {
		changes = append(changes, fmt.Sprintf("Name changed from '%s' to '%s'", oldProject.Name, input.Name))
	}

	// Check client changes
	if input.Client.Name != oldProject.Client.Name || input.Client.Contact != oldProject.Client.Contact || input.Client.Email != oldProject.Client.Email {
		changes = append(changes, "Client information updated")
	}

	// Check budget change
	if input.Budget != oldProject.Budget {
		changes = append(changes, fmt.Sprintf("Budget changed from %.2f to %.2f", oldProject.Budget, input.Budget))
	}

	// Check address changes
	oldAddr := oldProject.Address
	newAddr := input.Address
	if oldAddr.Line1 != newAddr.Line1 || oldAddr.Line2 != newAddr.Line2 || oldAddr.Line3 != newAddr.Line3 ||
		oldAddr.Postcode != newAddr.Postcode || oldAddr.City != newAddr.City || oldAddr.State != newAddr.State {
		changes = append(changes, "Address updated")
	}

	// Check description change
	if input.Description != oldProject.Description {
		changes = append(changes, "Description updated")
	}

	// Check remark change
	if input.Remark != oldProject.Remark {
		changes = append(changes, "Remark updated")
	}

	// Check area materials change
	if len(input.AreaMaterials) != len(oldProject.AreaMaterials) {
		changes = append(changes, fmt.Sprintf("Area materials changed (from %d to %d areas)", len(oldProject.AreaMaterials), len(input.AreaMaterials)))
	} else {
		// Check if materials content changed
		materialsChanged := false
		for i := range input.AreaMaterials {
			if len(input.AreaMaterials[i].Materials) != len(oldProject.AreaMaterials[i].Materials) {
				materialsChanged = true
				break
			}
		}
		if materialsChanged {
			changes = append(changes, "Area materials updated")
		}
	}

	// Check term conditions change
	if len(input.TermCondition) != len(oldProject.TermCondition) {
		changes = append(changes, fmt.Sprintf("Terms & conditions changed (from %d to %d items)", len(oldProject.TermCondition), len(input.TermCondition)))
	}

	// Check discounts change
	if len(input.Discounts) != len(oldProject.Discounts) {
		changes = append(changes, fmt.Sprintf("Discounts changed (from %d to %d items)", len(oldProject.Discounts), len(input.Discounts)))
	}

	// Check additional charges change
	if len(input.AdditionalCharges) != len(oldProject.AdditionalCharges) {
		changes = append(changes, fmt.Sprintf("Additional charges changed (from %d to %d items)", len(oldProject.AdditionalCharges), len(input.AdditionalCharges)))
	}

	// Check media change
	if len(input.Media) != len(oldProject.Media) {
		changes = append(changes, fmt.Sprintf("Media updated (from %d to %d items)", len(oldProject.Media), len(input.Media)))
	}

	// Check estimated completion date change
	if !input.EstimatedCompleteAt.Equal(oldProject.EstimatedCompleteAt) {
		changes = append(changes, fmt.Sprintf("Estimated completion date changed from %s to %s",
			oldProject.EstimatedCompleteAt.Format("2006-01-02"),
			input.EstimatedCompleteAt.Format("2006-01-02")))
	}

	// Check PIC changes
	if len(input.Pic) != len(oldProject.Pic) {
		changes = append(changes, fmt.Sprintf("PIC changed (from %d to %d persons)", len(oldProject.Pic), len(input.Pic)))
	} else {
		// Check if PIC IDs changed
		picChanged := false
		oldPicMap := make(map[string]bool)
		for _, pic := range oldProject.Pic {
			oldPicMap[pic.Hex()] = true
		}
		for _, pic := range input.Pic {
			if !oldPicMap[pic.Hex()] {
				picChanged = true
				break
			}
		}
		if picChanged {
			changes = append(changes, "PIC updated")
		}
	}

	// Generate action logs from changes
	actionLogs := make([]database.SystemActionLog, 0, len(changes))
	currentTime := time.Now()

	for _, change := range changes {
		actionLogs = append(actionLogs, database.SystemActionLog{
			Description: change,
			Time:        currentTime,
			ByName:      systemContext.User.Username,
			ById:        systemContext.User.ID,
		})
	}

	return actionLogs
}

func ProjectUpdate(input *model.ProjectUpdateRequest, systemContext *model.SystemContext) (*database.Project, error) {
	// Validate input
	if err := projectUpdateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("project")

	// Check if project exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Project
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Project not found", nil)
	}

	// Generate action logs for changes
	newActionLogs := generateProjectUpdateActionLogs(&doc, input, systemContext)

	// Append new action logs to existing ones
	updatedActionLogs := append(doc.ActionLogs, newActionLogs...)

	// Calculate totals (reuse quotation calculation)
	totalCharge, totalDiscount, totalAdditionalCharge, totalNettCharge := calculateQuotationTotals(input.AreaMaterials, input.Discounts, input.AdditionalCharges)

	// Build update object
	updateFields := bson.M{
		"name":                  input.Name,
		"client":                input.Client,
		"budget":                input.Budget,
		"address":               input.Address,
		"description":           input.Description,
		"remark":                input.Remark,
		"areaMaterials":         input.AreaMaterials,
		"termCondition":         input.TermCondition,
		"discounts":             input.Discounts,
		"media":                 input.Media,
		"additionalCharges":     input.AdditionalCharges,
		"totalCharge":           totalCharge,
		"totalDiscount":         totalDiscount,
		"totalAdditionalCharge": totalAdditionalCharge,
		"totalNettCharge":       totalNettCharge,
		"estimatedCompleteAt":   input.EstimatedCompleteAt,
		"pic":                   input.Pic,
		"actionLogs":            updatedActionLogs,
		"updatedAt":             time.Now(),
		"updatedBy":             systemContext.User.ID,
	}

	update := bson.M{"$set": updateFields}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update project", nil)
	}

	// Return updated project
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
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Project not found", nil)
	}

	return &doc, nil
}

func ProjectList(input model.ProjectListRequest, systemContext *model.SystemContext) (*model.ProjectListResponse, error) {
	// Check if user has a company
	if systemContext.User.Company == nil {
		return &model.ProjectListResponse{
			Data:       []bson.M{},
			Page:       1,
			Limit:      10,
			Total:      0,
			TotalPages: 0,
		}, nil
	}

	collection := systemContext.MongoDB.Collection("project")

	// Build base filter
	filter := bson.M{"isDeleted": false, "company": systemContext.User.Company}

	// Add field-specific filters
	if strings.TrimSpace(input.Name) != "" {
		filter["name"] = primitive.Regex{Pattern: input.Name, Options: "i"}
	}
	if strings.TrimSpace(input.Description) != "" {
		filter["description"] = primitive.Regex{Pattern: input.Description, Options: "i"}
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
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Project not found", nil)
	}

	// Create deletion action log
	deletionLog := database.SystemActionLog{
		Description: fmt.Sprintf("Project deleted by %s", systemContext.User.Username),
		Time:        time.Now(),
		ByName:      systemContext.User.Username,
		ById:        systemContext.User.ID,
	}

	// Append deletion log to existing action logs
	updatedActionLogs := append(doc.ActionLogs, deletionLog)

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
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to delete project", nil)
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

	// Build sort options - use bson.D to preserve order for multiple sort fields
	var sortOptions bson.D
	if len(input.Sort) > 0 {
		// Convert bson.M to bson.D to preserve field order
		for key, value := range input.Sort {
			sortOptions = append(sortOptions, bson.E{Key: key, Value: value})
		}
	} else {
		// Default sort by createdAt descending
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

func generateUniqueProjectName(baseName string, systemContext *model.SystemContext) (string, error) {
	collection := systemContext.MongoDB.Collection("project")

	// Trim the base name
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		baseName = "Untitled Project"
	}

	// Check if the base name exists
	filter := bson.M{
		"name":      baseName,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return "", utils.SystemError(enum.ErrorCodeInternal, "Failed to check project name uniqueness", nil)
	}

	// If no duplicate, return the base name
	if count == 0 {
		return baseName, nil
	}

	// Find the next available number
	counter := 1
	for {
		newName := fmt.Sprintf("%s (%d)", baseName, counter)
		filter := bson.M{
			"name":      newName,
			"company":   systemContext.User.Company,
			"isDeleted": false,
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return "", utils.SystemError(enum.ErrorCodeInternal, "Failed to check project name uniqueness", nil)
		}

		if count == 0 {
			return newName, nil
		}

		counter++
		// Prevent infinite loop
		if counter > 10000 {
			return "", utils.SystemError(enum.ErrorCodeInternal, "Failed to generate unique project name", nil)
		}
	}
}

func ProjectGetProcurementStatus(projectID primitive.ObjectID, systemContext *model.SystemContext) (*model.ProjectProcurementStatusResponse, error) {
	// Validate project exists and belongs to user's company
	project, err := ProjectGetByID(projectID, systemContext)
	if err != nil {
		return nil, err
	}

	// Build material map with area information
	type materialAreaInfo struct {
		MaterialID   *primitive.ObjectID
		MaterialName string
		AreaName     string
		Required     float64
		Ordered      float64
	}

	materialAreas := make(map[string][]materialAreaInfo)

	// Aggregate materials from all areas
	for _, areaMaterial := range project.AreaMaterials {
		for _, materialDetail := range areaMaterial.Materials {
			if materialDetail.Material == nil {
				continue
			}

			materialKey := materialDetail.Material.Hex()
			materialAreas[materialKey] = append(materialAreas[materialKey], materialAreaInfo{
				MaterialID:   materialDetail.Material,
				MaterialName: materialDetail.Name,
				AreaName:     areaMaterial.Area.Name,
				Required:     materialDetail.Quantity,
				Ordered:      0, // Will be filled from orders
			})
		}
	}

	// Get all project orders
	orderCollection := systemContext.MongoDB.Collection("order")
	filter := bson.M{
		"project":   projectID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	cursor, err := orderCollection.Find(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.ProjectGetProcurementStatus", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve orders", nil)
	}
	defer cursor.Close(context.Background())

	var orders []database.Order
	if err = cursor.All(context.Background(), &orders); err != nil {
		systemContext.Logger.Error("service.ProjectGetProcurementStatus - Failed to decode orders", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode orders", nil)
	}

	// Build material procurement details
	materials := []model.MaterialProcurementDetail{}

	for materialKey, areas := range materialAreas {
		if len(areas) == 0 {
			continue
		}

		materialDetail := model.MaterialProcurementDetail{
			MaterialID:        materialKey,
			Name:              areas[0].MaterialName,
			TotalRequired:     0,
			TotalOrdered:      0,
			ProcurementStatus: enum.ProcurementStatusNotOrdered,
			ByArea:            []model.AreaProcurementDetail{},
			Orders:            []model.OrderProcurementReference{},
		}

		// Calculate total required across all areas
		for _, area := range areas {
			materialDetail.TotalRequired += area.Required
		}

		// Calculate total ordered from all orders
		orderReferences := make(map[string]*model.OrderProcurementReference)

		for _, order := range orders {
			for _, item := range order.Items {
				if item.Material != nil && item.Material.Hex() == materialKey {
					materialDetail.TotalOrdered += item.Quantity

					// Track order reference
					orderKey := order.ID.Hex()
					if _, exists := orderReferences[orderKey]; !exists {
						orderReferences[orderKey] = &model.OrderProcurementReference{
							OrderID:   orderKey,
							PONumber:  order.PONumber,
							Quantity:  item.Quantity,
							Status:    order.Status,
							Supplier:  order.Supplier.Name,
							OrderDate: order.OrderDate,
						}
					} else {
						// Accumulate quantity if same material appears multiple times in same order
						orderReferences[orderKey].Quantity += item.Quantity
					}
				}
			}
		}

		// Convert order references map to slice
		for _, orderRef := range orderReferences {
			materialDetail.Orders = append(materialDetail.Orders, *orderRef)
		}

		// Calculate procurement status
		if materialDetail.TotalOrdered == 0 {
			materialDetail.ProcurementStatus = enum.ProcurementStatusNotOrdered
		} else if materialDetail.TotalOrdered < materialDetail.TotalRequired {
			materialDetail.ProcurementStatus = enum.ProcurementStatusPartial
		} else if materialDetail.TotalOrdered == materialDetail.TotalRequired {
			materialDetail.ProcurementStatus = enum.ProcurementStatusFullyOrdered
		} else {
			materialDetail.ProcurementStatus = enum.ProcurementStatusOverOrdered
		}

		// Build area breakdown
		for _, area := range areas {
			areaOrdered := 0.0

			// Calculate how much of this area's material has been ordered
			// This is a simplified calculation - in reality, you might want to track this more precisely
			if materialDetail.TotalRequired > 0 {
				areaOrdered = (area.Required / materialDetail.TotalRequired) * materialDetail.TotalOrdered
			}

			areaStatus := enum.ProcurementStatusNotOrdered
			if areaOrdered == 0 {
				areaStatus = enum.ProcurementStatusNotOrdered
			} else if areaOrdered < area.Required {
				areaStatus = enum.ProcurementStatusPartial
			} else if areaOrdered == area.Required {
				areaStatus = enum.ProcurementStatusFullyOrdered
			} else {
				areaStatus = enum.ProcurementStatusOverOrdered
			}

			materialDetail.ByArea = append(materialDetail.ByArea, model.AreaProcurementDetail{
				AreaName: area.AreaName,
				Required: area.Required,
				Ordered:  areaOrdered,
				Status:   areaStatus,
			})
		}

		materials = append(materials, materialDetail)
	}

	response := &model.ProjectProcurementStatusResponse{
		Materials: materials,
	}

	return response, nil
}

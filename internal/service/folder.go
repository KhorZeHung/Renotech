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

func folderCreateValidation(input *database.Folder, systemContext *model.SystemContext) error {
	// Validate media
	if err := validateFolderMedia(input.Media, systemContext); err != nil {
		return err
	}

	// Generate unique name
	uniqueName, err := generateUniqueFolderName(input.Name, systemContext)
	if err != nil {
		return err
	}

	input.Name = uniqueName
	return nil
}

func FolderCreate(input *database.Folder, systemContext *model.SystemContext) (*database.Folder, error) {
	// Validate input
	if err := folderCreateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("folder")

	// Create folder object
	folder := &database.Folder{
		Name:        input.Name,
		Company:     systemContext.User.Company,
		Client:      input.Client,
		Budget:      input.Budget,
		Address:     input.Address,
		Description: input.Description,
		Remark:      input.Remark,
		Status:      input.Status,
		Media:       input.Media,
		Areas:       input.Areas,
		CreatedAt:   time.Now(),
		CreatedBy:   *systemContext.User.ID,
		UpdatedAt:   time.Now(),
		UpdatedBy:   systemContext.User.ID,
		IsDeleted:   false,
	}

	result, err := collection.InsertOne(context.Background(), folder)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create folder", nil)
	}

	folderID := result.InsertedID.(primitive.ObjectID)

	var doc database.Folder
	err = collection.FindOne(context.Background(), bson.M{"_id": folderID}).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve folder", nil)
	}

	return &doc, nil
}

func folderUpdateValidation(input *database.Folder, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("folder")

	// Check if folder exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var currentFolder database.Folder
	err := collection.FindOne(context.Background(), filter).Decode(&currentFolder)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeNotFound, "Folder not found", nil)
	}

	// Validate media
	if err := validateFolderMedia(input.Media, systemContext); err != nil {
		return err
	}

	// If name is being changed, generate unique name
	if input.Name != currentFolder.Name {
		uniqueName, err := generateUniqueFolderName(input.Name, systemContext)
		if err != nil {
			return err
		}
		input.Name = uniqueName
	}

	return nil
}

func FolderUpdate(input *database.Folder, systemContext *model.SystemContext) (*database.Folder, error) {
	// Validate input
	if err := folderUpdateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("folder")

	// Check if folder exists
	filter := bson.M{
		"_id":       input.ID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Folder
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "folder not found", nil)
	}

	// Build update object
	updateFields := bson.M{
		"name":        input.Name,
		"client":      input.Client,
		"budget":      input.Budget,
		"address":     input.Address,
		"description": input.Description,
		"remark":      input.Remark,
		"status":      input.Status,
		"media":       input.Media,
		"areas":       input.Areas,
		"updatedAt":   time.Now(),
		"updatedBy":   systemContext.User.ID,
	}

	update := bson.M{"$set": updateFields}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to update folder", nil)
	}

	// Return updated folder
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated folder", nil)
	}

	return &doc, nil
}

func FolderGetByID(folderID primitive.ObjectID, systemContext *model.SystemContext) (*database.Folder, error) {
	collection := systemContext.MongoDB.Collection("folder")

	// Build filter
	filter := bson.M{
		"_id":       folderID,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Folder
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "folder not found", nil)
	}

	return &doc, nil
}

func FolderList(input model.FolderListRequest, systemContext *model.SystemContext) (*model.FolderListResponse, error) {
	// Check if user has a company
	if systemContext.User.Company == nil {
		return &model.FolderListResponse{
			Data:       []bson.M{},
			Page:       1,
			Limit:      10,
			Total:      0,
			TotalPages: 0,
		}, nil
	}

	collection := systemContext.MongoDB.Collection("folder")

	// Build base filter
	filter := bson.M{"isDeleted": false, "company": systemContext.User.Company}

	// Add field-specific filters
	if strings.TrimSpace(input.Name) != "" {
		filter["name"] = primitive.Regex{Pattern: input.Name, Options: "i"}
	}
	if strings.TrimSpace(input.ClientName) != "" {
		filter["clientName"] = primitive.Regex{Pattern: input.ClientName, Options: "i"}
	}
	if strings.TrimSpace(input.ClientContact) != "" {
		filter["clientContact"] = primitive.Regex{Pattern: input.ClientContact, Options: "i"}
	}
	if strings.TrimSpace(input.ClientEmail) != "" {
		filter["clientEmail"] = primitive.Regex{Pattern: input.ClientEmail, Options: "i"}
	}
	if strings.TrimSpace(input.ProjectAddress) != "" {
		filter["projectAddress"] = primitive.Regex{Pattern: input.ProjectAddress, Options: "i"}
	}
	if input.Status != "" {
		filter["status"] = input.Status
	}

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"name": searchRegex},
				{"clientName": searchRegex},
				{"clientContact": searchRegex},
				{"clientEmail": searchRegex},
				{"projectAddress": searchRegex},
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

	return executeFolderList(collection, filter, input, systemContext)
}

func FolderDelete(input primitive.ObjectID, systemContext *model.SystemContext) (*database.Folder, error) {
	collection := systemContext.MongoDB.Collection("folder")

	// Check if folder exists
	filter := bson.M{
		"_id":       input,
		"company":   systemContext.User.Company,
		"isDeleted": false,
	}

	var doc database.Folder
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "folder not found", nil)
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
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to delete folder", nil)
	}

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	return &doc, nil
}

// Helper functions
func executeFolderList(collection *mongo.Collection, filter bson.M, input model.FolderListRequest, systemContext *model.SystemContext) (*model.FolderListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.FolderList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count folders", nil)
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
		systemContext.Logger.Error("service.FolderList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve folders", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var folders []bson.M
	if err = cursor.All(context.Background(), &folders); err != nil {
		systemContext.Logger.Error("service.FolderList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode folders", nil)
	}

	response := &model.FolderListResponse{
		Data:       folders,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

func generateUniqueFolderName(baseName string, systemContext *model.SystemContext) (string, error) {
	collection := systemContext.MongoDB.Collection("folder")

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

func validateFolderMedia(media []database.SystemMedia, systemContext *model.SystemContext) error {
	if len(media) > 0 {
		mediaCollection := systemContext.MongoDB.Collection("media")

		for i, mediaItem := range media {
			// Skip validation if path is empty
			if strings.TrimSpace(mediaItem.Path) == "" {
				return utils.SystemError(enum.ErrorCodeBadRequest, fmt.Sprintf("Empty path in media no.%v", i+1), nil)
			}

			// Check if media exists, has correct module, and belongs to company
			filter := bson.M{
				"path":    mediaItem.Path,
				"module":  "folder",
				"company": systemContext.User.Company,
			}

			count, err := mediaCollection.CountDocuments(context.Background(), filter)
			if err != nil {
				return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate media", nil)
			}

			if count == 0 {
				return utils.SystemError(
					enum.ErrorCodeValidation,
					"Media not found or invalid module/company",
					map[string]interface{}{"path": mediaItem.Path},
				)
			}
		}
	}

	return nil
}

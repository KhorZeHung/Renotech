package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/utils"
)

func MediaCreate(input *database.Media, systemContext *model.SystemContext) (*database.Media, error) {
	collection := systemContext.MongoDB.Collection("media")

	// Set default fields and normalize path
	input.CreatedAt = time.Now()
	input.Path = strings.ReplaceAll(input.Path, "\\", "/")
	input.Path = strings.ReplaceAll(input.Path, "//", "/")
	input.Type = mediaAssignType(input.Extension)

	result, err := collection.InsertOne(context.Background(), input)

	if err != nil {
		return nil, err
	}

	oid := result.InsertedID.(primitive.ObjectID)

	var doc database.Media

	_ = collection.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&doc)

	return &doc, nil
}

func MediaDelete(input primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("media")

	filter := bson.M{"_id": input}

	var doc database.Media

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	if doc.ID == nil {
		return fmt.Errorf("media not found")
	}

	_, err := collection.DeleteOne(context.Background(), filter)

	if err != nil {
		return err
	}

	err = os.Remove(doc.Path)

	if err != nil {
		return err
	}

	return nil
}

func MediaDeleteByPath(filePath string, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("media")

	// Normalize the path for comparison
	normalizedPath := strings.ReplaceAll(filePath, "\\", "/")
	normalizedPath = strings.TrimPrefix(normalizedPath, "./")

	// Find media record with matching path, company, and creator validation
	filter := bson.M{
		"path":      normalizedPath,
		"company":   *systemContext.User.Company,
		"createdBy": *systemContext.User.ID,
	}

	var doc database.Media
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return utils.SystemError(
				enum.ErrorCodeNotFound,
				"Media file not found or you don't have permission to delete it",
				map[string]interface{}{"path": filePath},
			)
		}
		return err
	}

	// Delete from database
	_, err = collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return err
	}

	// Delete physical file
	err = os.Remove(doc.Path)
	if err != nil && !os.IsNotExist(err) {
		// Log error but don't fail if file doesn't exist
		return err
	}

	return nil
}

func MediaList(input model.MediaListRequest) (*model.MediaListResponse, error) {
	// Get MongoDB connection (unprotected endpoint, so no system context)
	mongoDB := utils.MongoGet()
	collection := mongoDB.Collection("media")

	// Build base filter
	filter := bson.M{}

	// Add field-specific filters
	if strings.TrimSpace(input.Name) != "" {
		filter["name"] = primitive.Regex{Pattern: input.Name, Options: "i"}
	}
	if strings.TrimSpace(input.Type) != "" {
		filter["type"] = input.Type
	}
	if strings.TrimSpace(input.Extension) != "" {
		filter["extension"] = input.Extension
	}
	if strings.TrimSpace(input.FileName) != "" {
		filter["fileName"] = primitive.Regex{Pattern: input.FileName, Options: "i"}
	}

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"name": searchRegex},
				{"fileName": searchRegex},
			},
		}

		// Combine existing filter with search filter
		if len(filter) > 0 {
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

	return executeMediaList(collection, filter, input)
}

func MediaValidateAccess(filePath string, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("media")

	// Normalize the path for comparison and remove ./ prefix if present
	normalizedPath := strings.ReplaceAll(filePath, "\\", "/")
	filePath = strings.TrimPrefix(filePath, "./")

	// Find media record with matching path and user's company
	filter := bson.M{
		"path":    normalizedPath,
		"company": *systemContext.User.Company,
	}

	var doc database.Media
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeUnauthorized, "Media file not found or access denied", nil)
	}

	return nil
}

// Helper function for media list execution
func executeMediaList(collection *mongo.Collection, filter bson.M, input model.MediaListRequest) (*model.MediaListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count media", nil)
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
		// Default sort by createdAt descending (newest first)
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
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve media", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var media []bson.M
	if err = cursor.All(context.Background(), &media); err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode media", nil)
	}

	response := &model.MediaListResponse{
		Data:       media,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

// Media validation helper functions
func ValidateMediaPath(path string, systemContext *model.SystemContext) error {
	if strings.TrimSpace(path) == "" {
		return nil // Empty path is valid (optional field)
	}

	collection := systemContext.MongoDB.Collection("media")

	// Normalize path for comparison and remove ./ prefix if present
	normalizedPath := strings.ReplaceAll(path, "\\", "/")
	normalizedPath = strings.TrimPrefix(normalizedPath, "./")

	// Find media record with matching path, company
	filter := bson.M{
		"path":    normalizedPath,
		"company": *systemContext.User.Company,
	}

	var doc database.Media
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Media path not found",
			map[string]interface{}{
				"path": path,
			},
		)
	}

	return nil
}

func ValidateMediaPaths(paths []string, systemContext *model.SystemContext) error {
	for i, path := range paths {
		if err := ValidateMediaPath(path, systemContext); err != nil {
			// Add index information for array validation
			if appErr, ok := err.(*model.AppError); ok {
				if details, ok := appErr.Details.(map[string]interface{}); ok {
					details["index"] = i
				}
			}
			return err
		}
	}
	return nil
}

func mediaAssignType(ext string) enum.MediaType {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".svg", ".ico":
		return enum.MediaTypeImage
	case ".mp4", ".mov", ".avi", ".mkv", ".wmv", ".flv", ".webm", ".mpeg", ".mpg", ".m4v":
		return enum.MediaTypeVideo
	case ".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".wma", ".aiff", ".opus":
		return enum.MediaTypeAudio
	case ".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".xls", ".xlsx", ".ppt", ".pptx", ".csv", ".md", ".html":
		return enum.MediaTypeDocument
	default:
		return enum.MediaTypeDocument
	}
}

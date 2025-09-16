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
func userCreateValidation(input *database.User, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("user")

	// Validate profile picture path if provided
	if err := utils.ValidateFilePath(input.ProfilePicture); err != nil {
		return err
	}

	// Check for email uniqueness across the entire system (for enabled, non-deleted users)
	emailFilter := bson.M{
		"email":     input.Email,
		"isDeleted": false,
		"isEnabled": true,
	}

	emailCount, err := collection.CountDocuments(context.Background(), emailFilter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to check email uniqueness", nil)
	}

	if emailCount > 0 {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Email already exists in the system",
			map[string]interface{}{
				"email": input.Email,
			},
		)
	}

	// Check for duplicates within company (only for enabled, non-deleted users)
	if input.Company != nil {
		filter := bson.M{
			"company":   input.Company,
			"isDeleted": false,
			"isEnabled": true,
			"$or": []bson.M{
				{"username": input.Username},
				{"contact": input.Contact},
			},
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicates", nil)
		}

		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Username or contact already exists in this company",
				map[string]interface{}{
					"username": input.Username,
					"contact":  input.Contact,
				},
			)
		}
	}

	return nil
}

func UserTenantCreate(input *database.User, systemContext *model.SystemContext) (*model.UserCreateResponse, error) {
	// Validate input
	if err := userCreateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("user")

	// Hash password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to hash password", nil)
	}

	// Create user object
	user := &database.User{
		Username:       input.Username,
		Email:          input.Email,
		Password:       hashedPassword,
		Contact:        input.Contact,
		Company:        input.Company,
		ProfilePicture: input.ProfilePicture,
		Permissions:    []string{}, 
		Type:           enum.UserTypeTenant,
		Comment:        input.Comment,
		IsDeleted:      false,
		IsEnabled:      true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		LastActiveTime: time.Now(),
	}

	result, err := collection.InsertOne(context.Background(), user)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create user", nil)
	}

	userID := result.InsertedID.(primitive.ObjectID)

	var doc database.User
	err = collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve user", nil)
	}

	// login to user
	loginInput := model.LoginRequest{
		Email:    input.Email,
		Password: input.Password,
	}

	token, respErr := AuthLogin(&loginInput, systemContext)

	if respErr != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to login user", nil)
	}

	response := model.UserCreateResponse{
		Token: token.Token,
		User:  doc,
	}

	return &response, nil
}

func userUpdateValidation(input *database.User, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("user")

	// Validate profile picture path if provided
	if err := utils.ValidateFilePath(input.ProfilePicture); err != nil {
		return err
	}

	// Get current user to find their company
	currentUser := &database.User{}
	err := collection.FindOne(context.Background(), bson.M{
		"_id":       input.ID,
		"isDeleted": false,
	}).Decode(currentUser)

	if err != nil {
		return utils.SystemError(enum.ErrorCodeNotFound, "User not found", nil)
	}

	// Check for email uniqueness across the entire system (excluding current user)
	emailFilter := bson.M{
		"email":     input.Email,
		"isDeleted": false,
		"isEnabled": true,
		"_id":       bson.M{"$ne": input.ID}, // Exclude current user
	}

	emailCount, err := collection.CountDocuments(context.Background(), emailFilter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to check email uniqueness", nil)
	}

	if emailCount > 0 {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Email already exists in the system",
			map[string]interface{}{
				"email": input.Email,
			},
		)
	}

	// Check for duplicates within company (excluding current user)
	if currentUser.Company != nil {
		filter := bson.M{
			"company":   currentUser.Company,
			"isDeleted": false,
			"isEnabled": true,
			"_id":       bson.M{"$ne": input.ID}, // Exclude current user
			"$or": []bson.M{
				{"username": input.Username},
				{"contact": input.Contact},
			},
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicates", nil)
		}

		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Username or contact already exists in this company",
				map[string]interface{}{
					"username": input.Username,
					"contact":  input.Contact,
				},
			)
		}
	}

	return nil
}

func UserTenantUpdate(input *database.User, systemContext *model.SystemContext) (*database.User, error) {
	// Validate input
	if err := userUpdateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("user")

	// Check if user exists
	filter := bson.M{
		"_id":       input.ID,
		"isDeleted": false,
	}

	var doc database.User
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "user not found", nil)
	}

	// Build update object
	updateFields := bson.M{
		"username":       input.Username,
		"email":          input.Email,
		"contact":        input.Contact,
		"profilePicture": input.ProfilePicture,
		"comment":        input.Comment,
		"updatedAt":      time.Now(),
	}

	update := bson.M{"$set": updateFields}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to update user", nil)
	}

	// Return updated user
	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated user", nil)
	}

	return &doc, nil
}

func UserTenantGetByID(userID primitive.ObjectID, systemContext *model.SystemContext) (*database.User, error) {
	collection := systemContext.MongoDB.Collection("user")

	// Build filter
	filter := bson.M{
		"_id":       userID,
		"isDeleted": false,
	}

	var doc database.User
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "user not found", nil)
	}

	return &doc, nil
}

func UserTenantList(input model.UserListRequest, systemContext *model.SystemContext) (*model.UserListResponse, error) {
	collection := systemContext.MongoDB.Collection("user")

	// Build base filter
	filter := bson.M{"isDeleted": false, "company": systemContext.User.Company}

	// Add field-specific filters
	if strings.TrimSpace(input.Username) != "" {
		filter["username"] = primitive.Regex{Pattern: input.Username, Options: "i"}
	}
	if strings.TrimSpace(input.Email) != "" {
		filter["email"] = primitive.Regex{Pattern: input.Email, Options: "i"}
	}
	if strings.TrimSpace(input.Contact) != "" {
		filter["contact"] = primitive.Regex{Pattern: input.Contact, Options: "i"}
	}
	if input.Type != "" {
		filter["type"] = input.Type
	}
	if input.IsEnabled != nil {
		filter["isEnabled"] = *input.IsEnabled
	}

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"username": searchRegex},
				{"email": searchRegex},
				{"contact": searchRegex},
				{"comment": searchRegex},
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

	return executeUserList(collection, filter, input, systemContext)
}


// Admin services
func userAdminCreateValidation(input *database.User, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("user")

	// Validate profile picture path if provided
	if err := utils.ValidateFilePath(input.ProfilePicture); err != nil {
		return err
	}

	// Check for email uniqueness across the entire system (for enabled, non-deleted users)
	emailFilter := bson.M{
		"email":     input.Email,
		"isDeleted": false,
		"isEnabled": true,
	}

	emailCount, err := collection.CountDocuments(context.Background(), emailFilter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to check email uniqueness", nil)
	}

	if emailCount > 0 {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Email already exists in the system",
			map[string]interface{}{
				"email": input.Email,
			},
		)
	}

	// Check for duplicates within company (only for enabled, non-deleted users)
	if input.Company != nil {
		filter := bson.M{
			"company":   input.Company,
			"isDeleted": false,
			"isEnabled": true,
			"$or": []bson.M{
				{"username": input.Username},
				{"contact": input.Contact},
			},
		}

		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicates", nil)
		}

		if count > 0 {
			return utils.SystemError(
				enum.ErrorCodeValidation,
				"Username or contact already exists in this company",
				map[string]interface{}{
					"username": input.Username,
					"contact":  input.Contact,
				},
			)
		}
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to hash password", nil)
	}

	// Create user object
	input.Password = hashedPassword
	input.Company = nil
	input.Type = enum.UserTypeSystemAdmin
	input.IsDeleted = false
	input.CreatedAt = time.Now()
	input.CreatedBy = systemContext.User.ID
	input.UpdatedAt = time.Now()
	input.UpdatedBy = systemContext.User.ID
	input.LastActiveTime = time.Time{}

	return nil
}

func UserAdminCreate(input *database.User, systemContext *model.SystemContext) (*database.User, error) {
	// Validate input
	if err := userAdminCreateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("user")

	result, err := collection.InsertOne(context.Background(), input)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create user", nil)
	}

	userID := result.InsertedID.(primitive.ObjectID)

	var doc database.User

	err = collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve user", nil)
	}

	return &doc, nil
}

func userAdminUpdateValidation(input *database.User, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("user")

	// Validate profile picture path if provided
	if err := utils.ValidateFilePath(input.ProfilePicture); err != nil {
		return err
	}

	// Check if user exists
	filter := bson.M{
		"_id":       input.ID,
		"isDeleted": false,
	}

	var doc database.User
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeNotFound, "user not found", nil)
	}

	// Check for email uniqueness across the entire system (excluding current user)
	emailFilter := bson.M{
		"email":     input.Email,
		"isDeleted": false,
		"isEnabled": true,
		"_id":       bson.M{"$ne": input.ID}, // Exclude current user
	}

	emailCount, err := collection.CountDocuments(context.Background(), emailFilter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to check email uniqueness", nil)
	}

	if emailCount > 0 {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Email already exists in the system",
			map[string]interface{}{
				"email": input.Email,
			},
		)
	}

	// Check for duplicates within company (excluding current user)
	filter = bson.M{
		"type":      enum.UserTypeSystemAdmin,
		"isDeleted": false,
		"isEnabled": true,
		"_id":       bson.M{"$ne": input.ID}, // Exclude current user
		"$or": []bson.M{
			{"username": input.Username},
			{"contact": input.Contact},
		},
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to check for duplicates", nil)
	}

	if count > 0 {
		return utils.SystemError(
			enum.ErrorCodeValidation,
			"Username or contact already exists in this company",
			map[string]interface{}{
				"username": input.Username,
				"contact":  input.Contact,
			},
		)
	}

	return nil
}

func UserAdminUpdate(input *database.User, systemContext *model.SystemContext) (*database.User, error) {
	// Validate input
	if err := userAdminUpdateValidation(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("user")

	filter := bson.M{
		"_id":       input.ID,
		"isDeleted": false,
	}

	// Build update object
	updateFields := bson.M{
		"username":       input.Username,
		"email":          input.Email,
		"contact":        input.Contact,
		"company":        input.Company,
		"profilePicture": input.ProfilePicture,
		"permissions":    input.Permissions,
		"type":           input.Type,
		"comment":        input.Comment,
		"isEnabled":      input.IsEnabled,
		"updatedAt":      time.Now(),
		"updatedBy":      systemContext.User.ID,
	}

	// Hash password if provided
	if strings.TrimSpace(input.Password) != "" {
		hashedPassword, err := utils.HashPassword(input.Password)
		if err != nil {
			return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to hash password", nil)
		}
		updateFields["password"] = hashedPassword
	}

	update := bson.M{"$set": updateFields}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to update user", nil)
	}

	// Return updated user
	var doc database.User

	err = collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve updated user", nil)
	}

	return &doc, nil
}

func UserAdminList(input model.UserListRequest, systemContext *model.SystemContext) (*model.UserListResponse, error) {
	collection := systemContext.MongoDB.Collection("user")

	// Build base filter
	filter := bson.M{"isDeleted": false}

	// Add field-specific filters
	if strings.TrimSpace(input.Username) != "" {
		filter["username"] = primitive.Regex{Pattern: input.Username, Options: "i"}
	}
	if strings.TrimSpace(input.Email) != "" {
		filter["email"] = primitive.Regex{Pattern: input.Email, Options: "i"}
	}
	if strings.TrimSpace(input.Contact) != "" {
		filter["contact"] = primitive.Regex{Pattern: input.Contact, Options: "i"}
	}
	if input.Type != "" {
		filter["type"] = input.Type
	}
	if input.IsEnabled != nil {
		filter["isEnabled"] = *input.IsEnabled
	}
	if input.Company != "" {
		if companyID, err := primitive.ObjectIDFromHex(input.Company); err == nil {
			filter["company"] = companyID
		}
	}

	// Add global search filter
	if strings.TrimSpace(input.Search) != "" {
		searchRegex := primitive.Regex{Pattern: input.Search, Options: "i"}
		searchFilter := bson.M{
			"$or": []bson.M{
				{"username": searchRegex},
				{"email": searchRegex},
				{"contact": searchRegex},
				{"comment": searchRegex},
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

	return executeUserList(collection, filter, input, systemContext)
}

// Share service
func UserDelete(input primitive.ObjectID, systemContext *model.SystemContext) (*database.User, error) {
	collection := systemContext.MongoDB.Collection("user")

	// Check if user exists
	filter := bson.M{
		"_id":       input,
		"isDeleted": false,
	}

	var doc database.User
	err := collection.FindOne(context.Background(), filter).Decode(&doc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "user not found", nil)
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
		return nil, utils.SystemError(enum.ErrorCodeInternal, "failed to delete user", nil)
	}

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	return &doc, nil
}

// Helper functions
func executeUserList(collection *mongo.Collection, filter bson.M, input model.UserListRequest, systemContext *model.SystemContext) (*model.UserListResponse, error) {
	// Get total count
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("service.UserList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to count users", nil)
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
		systemContext.Logger.Error("service.UserList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve users", nil)
	}
	defer cursor.Close(context.Background())

	// Decode results
	var users []bson.M
	if err = cursor.All(context.Background(), &users); err != nil {
		systemContext.Logger.Error("service.UserList", zap.Error(err))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode users", nil)
	}

	response := &model.UserListResponse{
		Data:       users,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response, nil
}

func userChangePasswordValidation(input *model.ChangePasswordRequest, userID primitive.ObjectID, systemContext *model.SystemContext) (*database.User, error) {
	collection := systemContext.MongoDB.Collection("user")

	// Check if passwords match
	if input.NewPassword != input.ConfirmPassword {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "New passwords do not match", nil)
	}

	// Find user by ID
	filter := bson.M{
		"_id":       userID,
		"isDeleted": false,
		"isEnabled": true,
	}

	var user database.User
	err := collection.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "User not found", nil)
	}

	// Verify old password
	if err := utils.VerifyPassword(user.Password, input.OldPassword); err != nil {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Current password is incorrect", nil)
	}

	return &user, nil
}

func UserChangePassword(input *model.ChangePasswordRequest, userID primitive.ObjectID, systemContext *model.SystemContext) (*model.ChangePasswordResponse, error) {
	// Validate input and find user
	user, err := userChangePasswordValidation(input, userID, systemContext)
	if err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("user")

	// Hash new password
	hashedPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to hash password", nil)
	}

	// Update user password
	update := bson.M{
		"$set": bson.M{
			"password":  hashedPassword,
			"updatedAt": time.Now(),
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": user.ID}, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update password", nil)
	}

	systemContext.Logger.Info("Password changed successfully", zap.String("userID", user.ID.Hex()))

	return &model.ChangePasswordResponse{
		Message: "Password has been changed successfully",
	}, nil
}

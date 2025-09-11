package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/utils"
)

func authLoginValidation(input *model.LoginRequest, systemContext *model.SystemContext) (*database.User, error) {
	collection := systemContext.MongoDB.Collection("user")

	// Find user by email
	filter := bson.M{
		"email":     input.Email,
		"isDeleted": false,
		"isEnabled": true,
	}

	var user database.User
	err := collection.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeUnauthorized, "Invalid email or password", nil)
	}

	// Verify password
	if err := utils.VerifyPassword(user.Password, input.Password); err != nil {
		return nil, utils.SystemError(enum.ErrorCodeUnauthorized, "Invalid email or password", nil)
	}

	return &user, nil
}

func AuthLogin(input *model.LoginRequest, systemContext *model.SystemContext) (*model.LoginResponse, error) {
	// Validate credentials
	user, err := authLoginValidation(input, systemContext)
	if err != nil {
		return nil, err
	}

	// Create JWT claims
	claims := &model.JWTClaims{
		UserID:    *user.ID,
		CompanyID: user.Company,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Get JWT secret from environment or use default
	secret := utils.GetEnvString("JWT_SECRET", "your-secret-key")
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to generate token", nil)
	}

	// Update last active time
	collection := systemContext.MongoDB.Collection("user")
	filter := bson.M{"_id": user.ID}
	update := bson.M{
		"$set": bson.M{
			"lastActiveTime": time.Now(),
		},
	}
	_, _ = collection.UpdateOne(context.Background(), filter, update)

	return &model.LoginResponse{
		Token: tokenString,
	}, nil
}

func authForgotPasswordValidation(input *model.ForgotPasswordRequest, systemContext *model.SystemContext) (*database.User, error) {
	collection := systemContext.MongoDB.Collection("user")

	// Find user by email
	filter := bson.M{
		"email":     input.Email,
		"isDeleted": false,
		"isEnabled": true,
	}

	var user database.User
	err := collection.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Email not found in the system", nil)
	}

	return &user, nil
}

func AuthForgotPassword(input *model.ForgotPasswordRequest, systemContext *model.SystemContext) (*model.ForgotPasswordResponse, error) {
	// Validate email and find user
	user, err := authForgotPasswordValidation(input, systemContext)
	if err != nil {
		return nil, err
	}

	// Generate secure random token
	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to generate reset token", nil)
	}
	resetToken := hex.EncodeToString(tokenBytes)

	// Set token expiry to 5 minutes from now
	tokenExpiry := time.Now().Add(5 * time.Minute)

	// Store reset token in separate collection
	tokenCollection := systemContext.MongoDB.Collection("password_reset_token")
	
	// First, mark any existing tokens for this email as used
	_, _ = tokenCollection.UpdateMany(
		context.Background(),
		bson.M{"email": input.Email, "isUsed": false},
		bson.M{"$set": bson.M{"isUsed": true}},
	)

	// Create new reset token record
	resetTokenDoc := &database.PasswordResetToken{
		Email:      input.Email,
		ResetToken: resetToken,
		ExpiresAt:  tokenExpiry,
		CreatedAt:  time.Now(),
		IsUsed:     false,
	}

	_, err = tokenCollection.InsertOne(context.Background(), resetTokenDoc)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to save reset token", nil)
	}

	// Send reset email
	err = utils.SendPasswordResetEmail(user.Email, resetToken)
	if err != nil {
		systemContext.Logger.Error("Failed to send password reset email", zap.Error(err), zap.String("email", user.Email))
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to send reset email", nil)
	}

	systemContext.Logger.Info("Password reset email sent", zap.String("email", user.Email))

	return &model.ForgotPasswordResponse{
		Message: "Password reset email has been sent to your email address",
	}, nil
}

func authResetPasswordValidation(input *model.ResetPasswordRequest, systemContext *model.SystemContext) (*database.User, *database.PasswordResetToken, error) {
	// Check if passwords match
	if input.NewPassword != input.ConfirmPassword {
		return nil, nil, utils.SystemError(enum.ErrorCodeValidation, "Passwords do not match", nil)
	}

	// Find reset token
	tokenCollection := systemContext.MongoDB.Collection("password_reset_token")
	filter := bson.M{
		"email":      input.Email,
		"resetToken": input.Token,
		"expiresAt":  bson.M{"$gt": time.Now()}, // Token not expired
		"isUsed":     false,                      // Token not used
	}

	var resetToken database.PasswordResetToken
	err := tokenCollection.FindOne(context.Background(), filter).Decode(&resetToken)
	if err != nil {
		return nil, nil, utils.SystemError(enum.ErrorCodeUnauthorized, "Invalid or expired reset token", nil)
	}

	// Find user by email
	userCollection := systemContext.MongoDB.Collection("user")
	userFilter := bson.M{
		"email":     input.Email,
		"isDeleted": false,
		"isEnabled": true,
	}

	var user database.User
	err = userCollection.FindOne(context.Background(), userFilter).Decode(&user)
	if err != nil {
		return nil, nil, utils.SystemError(enum.ErrorCodeNotFound, "User not found", nil)
	}

	return &user, &resetToken, nil
}

func AuthResetPassword(input *model.ResetPasswordRequest, systemContext *model.SystemContext) (*model.ResetPasswordResponse, error) {
	// Validate token and find user
	user, resetToken, err := authResetPasswordValidation(input, systemContext)
	if err != nil {
		return nil, err
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to hash password", nil)
	}

	// Update user password
	userCollection := systemContext.MongoDB.Collection("user")
	update := bson.M{
		"$set": bson.M{
			"password":  hashedPassword,
			"updatedAt": time.Now(),
		},
	}

	_, err = userCollection.UpdateOne(context.Background(), bson.M{"_id": user.ID}, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update password", nil)
	}

	// Mark reset token as used
	tokenCollection := systemContext.MongoDB.Collection("password_reset_token")
	_, err = tokenCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": resetToken.ID},
		bson.M{"$set": bson.M{"isUsed": true}},
	)
	if err != nil {
		systemContext.Logger.Error("Failed to mark reset token as used", zap.Error(err))
	}

	systemContext.Logger.Info("Password reset successfully", zap.String("email", user.Email))

	return &model.ResetPasswordResponse{
		Message: "Password has been reset successfully",
	}, nil
}

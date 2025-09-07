package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
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

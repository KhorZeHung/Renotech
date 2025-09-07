package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/utils"
)

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.SendErrorResponse(c, utils.SystemError(enum.ErrorCodeUnauthorized, "Authorization header required", nil))
			c.Abort()
			return
		}

		// Check Bearer token format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			utils.SendErrorResponse(c, utils.SystemError(enum.ErrorCodeUnauthorized, "Invalid authorization header format", nil))
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, utils.SystemError(enum.ErrorCodeUnauthorized, "Invalid token signing method", nil)
			}
			secret := utils.GetEnvString("JWT_SECRET", "your-secret-key")
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			utils.SendErrorResponse(c, utils.SystemError(enum.ErrorCodeUnauthorized, "Invalid or expired token", nil))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*model.JWTClaims)
		if !ok {
			utils.SendErrorResponse(c, utils.SystemError(enum.ErrorCodeUnauthorized, "Invalid token claims", nil))
			c.Abort()
			return
		}

		// Get system context from request
		requestID, _ := c.Get("RequestID")
		systemContext := utils.SystemContextWithRequestID(requestID.(string))

		// Fetch user from database
		userCollection := systemContext.MongoDB.Collection("user")
		var user database.User
		err = userCollection.FindOne(context.Background(), bson.M{
			"_id":       claims.UserID,
			"isDeleted": false,
			"isEnabled": true,
		}).Decode(&user)

		if err != nil {
			utils.SendErrorResponse(c, utils.SystemError(enum.ErrorCodeUnauthorized, "User not found or inactive", nil))
			c.Abort()
			return
		}

		// Update system context with user information
		systemContext.User = user

		// Store the updated system context back to the gin context
		c.Set("SystemContext", systemContext)

		c.Next()
	}
}

func AuthenticateUser(systemContext *model.SystemContext, userID primitive.ObjectID) error {
	userCollection := systemContext.MongoDB.Collection("user")
	var user database.User
	
	err := userCollection.FindOne(context.Background(), bson.M{
		"_id":       userID,
		"isDeleted": false,
		"isEnabled": true,
	}).Decode(&user)

	if err != nil {
		return utils.SystemError(enum.ErrorCodeUnauthorized, "User not found or inactive", nil)
	}

	// Update system context with user information
	systemContext.User = user
	return nil
}
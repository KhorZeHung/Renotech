package model

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"renotech.com.my/internal/database"
)

type SystemContext struct {
	MongoDB *mongo.Database
	Logger  *zap.Logger
	User    database.User
}

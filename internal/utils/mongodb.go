package utils

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var systemClient *mongo.Client
var systemDb *mongo.Database

func MongoInit() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var err error

	systemClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017/"))

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	systemDb = systemClient.Database("renotech")

	return nil
}

func MongoCleanUp() {
	if systemClient != nil {
		_ = systemClient.Disconnect(context.Background())
	}
}

func MongoGet() *mongo.Database {
	if systemClient == nil {
		_ = MongoInit()
	}

	return systemDb
}

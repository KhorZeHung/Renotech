package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	systemClient *mongo.Client
	systemDb     *mongo.Database
	mongoMutex   sync.RWMutex
	mongoURI     string
	dbName       string
)

func MongoInit() error {
	mongoMutex.Lock()
	defer mongoMutex.Unlock()

	// Load configuration from environment
	loadMongoConfig()

	// Create connection options
	clientOptions := options.Client().ApplyURI(mongoURI)

	// Set connection pool options
	if maxPoolSize := getEnvInt("MONGO_MAX_POOL_SIZE", 100); maxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(uint64(maxPoolSize))
	}
	if minPoolSize := getEnvInt("MONGO_MIN_POOL_SIZE", 10); minPoolSize > 0 {
		clientOptions.SetMinPoolSize(uint64(minPoolSize))
	}

	// Set timeout
	timeout := time.Duration(getEnvInt("MONGO_TIMEOUT", 20)) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var err error
	systemClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err = systemClient.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	systemDb = systemClient.Database(dbName)
	log.Printf("Successfully connected to MongoDB database: %s", dbName)
	return nil
}

func MongoCleanUp() {
	mongoMutex.Lock()
	defer mongoMutex.Unlock()

	if systemClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := systemClient.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		} else {
			log.Println("Successfully disconnected from MongoDB")
		}
		systemClient = nil
		systemDb = nil
	}
}

func MongoGet() *mongo.Database {
	mongoMutex.RLock()
	if systemClient != nil && systemDb != nil {
		db := systemDb
		mongoMutex.RUnlock()
		return db
	}
	mongoMutex.RUnlock()

	// Need to initialize
	if err := MongoInit(); err != nil {
		log.Printf("Failed to initialize MongoDB connection: %v", err)
		return nil
	}

	mongoMutex.RLock()
	defer mongoMutex.RUnlock()
	return systemDb
}

// MongoHealthCheck checks if the MongoDB connection is healthy
func MongoHealthCheck() error {
	mongoMutex.RLock()
	client := systemClient
	mongoMutex.RUnlock()

	if client == nil {
		return fmt.Errorf("MongoDB client is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return client.Ping(ctx, readpref.Primary())
}

// loadMongoConfig loads MongoDB configuration from environment variables
func loadMongoConfig() {
	mongoURI = "mongodb+srv://khorzehung:Eh85Nmsm3VyjvcZM@renotechcluster.5f3prsq.mongodb.net/?retryWrites=true&w=majority&appName=RenotechCluster"
	dbName = "renotech"
}

// getEnvString gets string environment variable with default
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets integer environment variable with default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

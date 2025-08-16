package logs

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"sync"
	"time"
)

var (
	logger      *zap.Logger
	currentDate string
	mu          sync.Mutex
)

func LoggingSetup() {
	// Get the current date for the log filename
	currentDate = time.Now().Format(time.DateOnly)
	logFileName := "./logs/log-histories/" + currentDate + ".log"

	// Check if the logs directory exists, and create it if it doesn't
	err := os.MkdirAll("./logs/log-histories", os.ModePerm)
	if err != nil {
		fmt.Println(err.Error())
		zap.L().Fatal("Failed to create logs directory", zap.Error(err))
	}

	// Configure the logger to write to both the log file and stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFileName, // Write logs to the file
		"stdout",    // Also write logs to stdout
	}

	// Build the logger
	logger, err = config.Build()
	if err != nil {
		fmt.Println(err.Error())
		zap.L().Fatal("Failed to initialize logger", zap.Error(err))
	}

	go loggerRotation()

	// Log initialization info
	logger.Info("Logging initialized.", zap.Any("date", currentDate))
}

func loggerRotation() {
	for {
		// Calculate time until next midnight
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		durationUntilMidnight := nextMidnight.Sub(now)

		// Sleep until midnight
		time.Sleep(durationUntilMidnight)

		mu.Lock()
		logger = nil
		LoggingSetup()
		mu.Unlock()
	}
}

func LoggerGet() *zap.Logger {
	return logger
}

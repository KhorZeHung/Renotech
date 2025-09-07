package logs

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger      *zap.Logger
	currentDate string
	mu          sync.Mutex
)

func LoggingSetup() {
	// Get configuration from environment
	logLevel := getLogLevel()
	logDir := getEnvString("LOG_DIR", "./logs/log-histories")

	// Get the current date for the log filename
	currentDate = time.Now().Format(time.DateOnly)
	logFileName := logDir + "/" + currentDate + ".log"

	// Check if the logs directory exists, and create it if it doesn't
	err := os.MkdirAll(logDir, 0755) // More restrictive permissions
	if err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
		return
	}

	// Configure the logger with structured format
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(logLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{logFileName, "stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Build the logger
	logger, err = config.Build()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	go loggerRotation()

	// Log initialization info
	logger.Info("Logging system initialized",
		zap.String("date", currentDate),
		zap.String("level", logLevel.String()),
		zap.String("logFile", logFileName),
	)
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

// LoggerWithRequestID returns a logger with request ID field
func LoggerWithRequestID(requestID string) *zap.Logger {
	if logger == nil {
		return zap.NewNop()
	}
	return logger.With(zap.String("requestId", requestID))
}

// getLogLevel gets log level from environment
func getLogLevel() zapcore.Level {
	levelStr := getEnvString("LOG_LEVEL", "info")
	switch strings.ToLower(levelStr) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// getEnvString gets string environment variable with default
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Package utils provides shared utilities such as logger initialization and helper functions.
package utils

import (
    "log"
    "go.uber.org/zap"
)

var (
    // Logger is a sugared logger for structured logging throughout the application.
    Logger *zap.SugaredLogger
)

// InitLogger initializes a production‑grade zap logger. It should be called early in main.
func InitLogger() {
    l, err := zap.NewProduction()
    if err != nil {
        // If zap fails to initialize, fall back to the standard logger and exit.
        log.Fatalf("failed to initialize zap logger: %v", err)
    }
    Logger = l.Sugar()
}

// Info logs an informational message with optional key/value fields.
func Info(msg string, fields ...interface{}) {
    if Logger != nil {
        Logger.Infow(msg, fields...)
    }
}

// Error logs an error message with optional key/value fields.
func Error(msg string, fields ...interface{}) {
    if Logger != nil {
        Logger.Errorw(msg, fields...)
    }
}

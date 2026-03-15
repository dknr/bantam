// Package logging provides structured logging utilities.
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type verboseKey struct{}

// LoggerKey is the context key for the logger.
type loggerKey int

const (
	loggerKeyConst loggerKey = 0
)

var LoggerKey = loggerKeyConst

// FromContext retrieves a logger from context.
func FromContext(ctx context.Context) logr.Logger {
	if logger, ok := ctx.Value(LoggerKey).(logr.Logger); ok {
		return logger
	}
	return logr.Discard()
}

// NewLogger creates a new logger instance with both file and console output.
// Console output is only shown when verbose mode is enabled.
func NewLogger(logsDir string, verbose bool) logr.Logger {
	// Create logs directory
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		logsDir = os.TempDir()
	}

	// Open log file
	logFile := filepath.Join(logsDir, "bantam.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		file, _ = os.CreateTemp("", "bantam-*.log")
		logFile = file.Name()
	}

	// Console encoder (colors enabled)
	consoleEnc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	// Write to both console and file
	writers := []zapcore.WriteSyncer{
		zapcore.AddSync(file),
	}

	// Only add console output if verbose is enabled
	if verbose {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	core := zapcore.NewCore(
		consoleEnc,
		zapcore.NewMultiWriteSyncer(writers...),
		zap.InfoLevel,
	)

	logger := zap.New(core, zap.AddCaller())
	return zapr.NewLogger(logger)
}

// NewContextWithLogger returns a context with the logger attached.
func NewContextWithLogger(ctx context.Context, logger logr.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// SetVerbose updates the verbose flag in the context.
func SetVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKey{}, verbose)
}

// IsVerbose checks if verbose mode is enabled in the context.
func IsVerbose(ctx context.Context) bool {
	verbose, _ := ctx.Value(verboseKey{}).(bool)
	return verbose
}

// Info logs an info message.
func Info(ctx context.Context, msg string, keysAndValues ...any) {
	logger := FromContext(ctx)
	logger.Info(msg, keysAndValues...)
}

// Error logs an error message.
func Error(ctx context.Context, err error, msg string, keysAndValues ...any) {
	logger := FromContext(ctx)
	logger.Error(err, msg, keysAndValues...)
}

// Debug logs a debug message.
func Debug(ctx context.Context, msg string, keysAndValues ...any) {
	logger := FromContext(ctx)
	logger.Info(msg, keysAndValues...)
}

// WithValues returns a new context with additional logger values.
func WithValues(ctx context.Context, keysAndValues ...any) context.Context {
	logger := FromContext(ctx)
	newLogger := logger.WithValues(keysAndValues...)
	return context.WithValue(ctx, LoggerKey, newLogger)
}

// Now returns the current time.
func Now() time.Time {
	return time.Now()
}

// Since returns the duration since t.
func Since(t time.Time) int {
	return int(time.Since(t).Milliseconds())
}

// PrintJSON prints pretty-printed JSON to stderr with the given label.
 	func PrintJSON(label string, data any) {
 		jsonBytes, err := json.MarshalIndent(data, "", "  ")
 		if err != nil {
 			// Fallback to single line if marshaling fails
 			jsonBytes, _ = json.Marshal(data)
 		}
 		fmt.Fprintf(os.Stderr, "\n%s:\n%s\n\n", label, string(jsonBytes))
 	}

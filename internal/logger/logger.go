package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type contextKey string

const loggerContextKey contextKey = "logger"

// Overridable for tests.
var userHomeDir = os.UserHomeDir

// DefaultLogPath resolves the standard OS-specific path for state files.
// It respects the XDG Base Directory Specification.
// It is a pure function and does NOT create any directories.
func DefaultLogPath() (string, error) {
	// 1. Respect XDG environment variable if set
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "lazyos", "lazyos.log"), nil
	}

	// 2. Fall back to standard XDG default: ~/.local/state
	homeDir, err := userHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".local", "state", "lazyos", "lazyos.log"), nil
}

// SetupFile opens or creates the appropriate log file based on config.
// It ensures all parent directories exist before attempting to create the file.
func SetupFile(logFile string) (*slog.Logger, *os.File, string, error) {
	path := logFile
	if path == "" {
		var err error
		path, err = DefaultLogPath()
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to resolve log path: %w", err)
		}
	}

	// Safely ensure the target directory exists, whether it's the default or custom
	logDir := filepath.Dir(path)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to open log file: %w", err)
	}

	log := Setup(file, slog.LevelDebug)
	return log, file, path, nil
}

// Setup creates a new slog.NewJSONHandler using the provided io.Writer.
func Setup(w io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// WithLogger returns a new context with the provided logger embedded.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// FromContext extracts the logger from the context.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerContextKey).(*slog.Logger); ok {
		return logger
	}
	// Return a silent logger to avoid panics if context lacks a logger
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

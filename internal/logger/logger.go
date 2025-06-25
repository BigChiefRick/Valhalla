package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Logger wraps slog.Logger with convenience methods
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance
func New() *Logger {
	var handler slog.Handler

	// Determine log format
	format := viper.GetString("log-format")
	if format == "" {
		format = "text"
	}

	// Configure handler based on format
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// Set debug level if enabled
	if viper.GetBool("debug") {
		opts.Level = slog.LevelDebug
	}

	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	return &Logger{Logger: logger}
}

// Fatal logs a fatal error and exits
func (l *Logger) Fatal(msg string, args ...any) {
	l.Error(msg, args...)
	os.Exit(1)
}

// WithProvider returns a logger with provider context
func (l *Logger) WithProvider(provider string) *Logger {
	return &Logger{
		Logger: l.With("provider", provider),
	}
}

// WithComponent returns a logger with component context
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.With("component", component),
	}
}

// WithOperation returns a logger with operation context
func (l *Logger) WithOperation(operation string) *Logger {
	return &Logger{
		Logger: l.With("operation", operation),
	}
}

// Progress logs progress messages with consistent formatting
func (l *Logger) Progress(msg string, current, total int) {
	l.Info(msg, 
		"progress", current,
		"total", total,
		"percentage", float64(current)/float64(total)*100,
	)
}

// StartOperation logs the start of an operation
func (l *Logger) StartOperation(operation string, args ...any) {
	allArgs := append([]any{"status", "started"}, args...)
	l.Info(operation, allArgs...)
}

// CompleteOperation logs the completion of an operation
func (l *Logger) CompleteOperation(operation string, args ...any) {
	allArgs := append([]any{"status", "completed"}, args...)
	l.Info(operation, allArgs...)
}

// FailOperation logs the failure of an operation
func (l *Logger) FailOperation(operation string, err error, args ...any) {
	allArgs := append([]any{"status", "failed", "error", err}, args...)
	l.Error(operation, allArgs...)
}

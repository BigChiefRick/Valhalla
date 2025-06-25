package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Logger wraps standard logger with structured logging capabilities
type Logger struct {
	logger *log.Logger
	format string
	level  LogLevel
	fields map[string]interface{}
}

// LogLevel represents the logging level
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// New creates a new logger instance
func New() *Logger {
	format := viper.GetString("log-format")
	if format == "" {
		format = "text"
	}

	level := LevelInfo
	if viper.GetBool("debug") {
		level = LevelDebug
	}

	return &Logger{
		logger: log.New(os.Stdout, "", 0),
		format: strings.ToLower(format),
		level:  level,
		fields: make(map[string]interface{}),
	}
}

// logEntry represents a structured log entry
type logEntry struct {
	Time    time.Time              `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"msg"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// log writes a log entry
func (l *Logger) log(level LogLevel, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	// Parse key-value pairs
	fields := make(map[string]interface{})
	for k, v := range l.fields {
		fields[k] = v
	}

	// Add args as key-value pairs
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key := fmt.Sprintf("%v", args[i])
			fields[key] = args[i+1]
		}
	}

	entry := logEntry{
		Time:    time.Now(),
		Level:   l.levelString(level),
		Message: msg,
		Fields:  fields,
	}

	var output string
	if l.format == "json" {
		jsonBytes, _ := json.Marshal(entry)
		output = string(jsonBytes)
	} else {
		output = l.formatText(entry)
	}

	l.logger.Println(output)
}

// levelString returns string representation of log level
func (l *Logger) levelString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// formatText formats log entry as text
func (l *Logger) formatText(entry logEntry) string {
	output := fmt.Sprintf("%s [%s] %s", 
		entry.Time.Format("2006-01-02 15:04:05"),
		entry.Level,
		entry.Message)

	if len(entry.Fields) > 0 {
		var fieldPairs []string
		for k, v := range entry.Fields {
			fieldPairs = append(fieldPairs, fmt.Sprintf("%s=%v", k, v))
		}
		output += " " + strings.Join(fieldPairs, " ")
	}

	return output
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// Fatal logs a fatal error and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.Error(msg, args...)
	os.Exit(1)
}

// With returns a new logger with additional fields
func (l *Logger) With(key string, value interface{}) *Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		logger: l.logger,
		format: l.format,
		level:  l.level,
		fields: newFields,
	}
}

// WithProvider returns a logger with provider context
func (l *Logger) WithProvider(provider string) *Logger {
	return l.With("provider", provider)
}

// WithComponent returns a logger with component context
func (l *Logger) WithComponent(component string) *Logger {
	return l.With("component", component)
}

// WithOperation returns a logger with operation context
func (l *Logger) WithOperation(operation string) *Logger {
	return l.With("operation", operation)
}

// Progress logs progress messages with consistent formatting
func (l *Logger) Progress(msg string, current, total int) {
	l.Info(msg,
		"progress", current,
		"total", total,
		"percentage", float64(current)/float64(total)*100)
}

// StartOperation logs the start of an operation
func (l *Logger) StartOperation(operation string, args ...interface{}) {
	allArgs := append([]interface{}{"status", "started"}, args...)
	l.Info(operation, allArgs...)
}

// CompleteOperation logs the completion of an operation
func (l *Logger) CompleteOperation(operation string, args ...interface{}) {
	allArgs := append([]interface{}{"status", "completed"}, args...)
	l.Info(operation, allArgs...)
}

// FailOperation logs the failure of an operation
func (l *Logger) FailOperation(operation string, err error, args ...interface{}) {
	allArgs := append([]interface{}{"status", "failed", "error", err}, args...)
	l.Error(operation, allArgs...)
}

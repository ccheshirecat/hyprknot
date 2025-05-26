package logger

import (
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// NewLogger creates a new logger instance based on configuration
func NewLogger(level, format, output string) (*logrus.Logger, error) {
	logger := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(logLevel)

	// Set log format
	switch strings.ToLower(format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// Set output
	switch strings.ToLower(output) {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		// Assume it's a file path
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		logger.SetOutput(file)
	}

	return logger, nil
}

// NewMultiLogger creates a logger that writes to multiple outputs
func NewMultiLogger(level, format string, outputs ...string) (*logrus.Logger, error) {
	logger := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(logLevel)

	// Set log format
	switch strings.ToLower(format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// Create multi-writer for multiple outputs
	var writers []io.Writer
	for _, output := range outputs {
		switch strings.ToLower(output) {
		case "stdout":
			writers = append(writers, os.Stdout)
		case "stderr":
			writers = append(writers, os.Stderr)
		default:
			// Assume it's a file path
			file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				return nil, err
			}
			writers = append(writers, file)
		}
	}

	if len(writers) > 0 {
		logger.SetOutput(io.MultiWriter(writers...))
	}

	return logger, nil
}

// WithFields creates a logger entry with fields
func WithFields(logger *logrus.Logger, fields map[string]interface{}) *logrus.Entry {
	return logger.WithFields(logrus.Fields(fields))
}

// WithField creates a logger entry with a single field
func WithField(logger *logrus.Logger, key string, value interface{}) *logrus.Entry {
	return logger.WithField(key, value)
}

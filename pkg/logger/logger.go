package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New instantiates a production-ready zap logger with sane defaults for JSON structured logging.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}

// Must is a helper that panics when the logger cannot be created.
func Must(logger *zap.Logger, err error) *zap.Logger {
	if err != nil {
		panic(err)
	}
	return logger
}

// Named returns a child logger with the provided component name.
func Named(base *zap.Logger, component string) *zap.Logger {
	if base == nil {
		return zap.NewNop()
	}
	return base.Named(component)
}

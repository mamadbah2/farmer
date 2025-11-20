# `pkg/logger`

Sane defaults for Zap logging across the service.

## API
- `New() (*zap.Logger, error)`: returns a production-configured logger with ISO-8601 timestamps.
- `Must(logger *zap.Logger, err error) *zap.Logger`: helper to panic when logger creation fails (used in `cmd/server`).
- `Named(base *zap.Logger, component string) *zap.Logger`: safe helper that falls back to `zap.NewNop()` when no base logger is available.

## Usage
```go
base := logger.Must(logger.New())
reposLogger := logger.Named(base, "repo.sheets")
reposLogger.Info("row appended")
```

## Why Zap?
- Structured JSON output for easier ingestion into ELK/Datadog.
- High-performance logging suitable for high-traffic webhooks.
- Consistent configuration makes logs predictable across environments.

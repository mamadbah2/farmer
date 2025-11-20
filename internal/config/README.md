# `internal/config`

Single source of truth for runtime configuration.

## Key Types
- `Config`: top-level struct grouping `Server`, `WhatsApp`, `Sheets`, and `Reporting` settings.
- `ServerConfig`: exposes `Port` used by the Gin server.
- `WhatsAppConfig`: contains access token, phone number ID, verify token, API host/version, and target group ID.
- `SheetsConfig`: Google Sheets service-account JSON path + spreadsheet ID.
- `ReportingConfig`: cron expression + timezone used by the future scheduler.

## Load Flow
1. `Load(envFile string)` optionally loads a `.env` file via `godotenv`.
2. Environment variables are read and defaulted where necessary (e.g. `APP_PORT`, `WHATSAPP_BASE_URL`).
3. `Validate()` is executed to ensure every required value is set before the server continues.

## Usage
```go
cfg, err := config.Load("")
if err != nil {
    panic(err)
}
fmt.Println("server listening on", cfg.Server.Port)
```

## Extending Config
- Add new fields to the relevant struct and wire them inside `Load()`.
- Update validation logic with clear error messages.
- Document the new env var inside `.env.example` and the root README.

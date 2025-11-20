# Farmer WhatsApp Automation Backend

Smart poultry operations platform that ingests WhatsApp worker updates, structures the data into Google Sheets, computes KPIs, and replies with actionable insights.

## Table of Contents
- [Features](#features)
- [Architecture](#architecture)
- [Runtime Flow](#runtime-flow)
- [Google Sheets Schema](#google-sheets-schema)
- [Configuration](#configuration)
- [Running Locally](#running-locally)
- [HTTP Endpoints](#http-endpoints)
- [Payload Examples](#payload-examples)
- [Development Notes](#development-notes)

## Features
- ✅ WhatsApp webhook verification and message ingestion (Gin HTTP server).
- ✅ Natural-language-ish command parsing for `/eggs`, `/feed`, `/mortality`, `/sales`, `/expenses`.
- ✅ Central command dispatcher that validates, persists to Google Sheets, and streams quick summaries back to workers.
- ✅ Google Sheets repository for append + read analytics with service account auth.
- ✅ Reporting service with daily + weekly KPI builders ready for scheduler-driven broadcasts.
- ✅ Structured logging with Zap and graceful shutdown handling.
- ✅ Modular packages (`internal`, `pkg`) to keep business logic isolated from transport.

## Architecture

```
┌────────────────────────┐
│ WhatsApp Cloud Webhook │
└─────────────┬──────────┘
              │  JSON payload
        ┌─────▼─────┐
        │ Gin Router│
        └─────┬─────┘
        ┌─────▼────────────────────────────────┐
        │ Webhook Handler                       │
        │ - verifies GET challenge              │
        │ - binds POST payloads                 │
        │ - delegates to Messaging service      │
        └─────┬────────────────────────────────┘
        ┌─────▼──────────────────┐
        │ WhatsApp Service       │
        │ - parses commands      │
        │ - calls dispatcher     │
        │ - sends replies via    │
        │   pkg/clients/whatsapp │
        └─────┬──────────────────┘
        ┌─────▼──────────────────────────┐
        │ Command Dispatcher             │
        │ - validates args               │
        │ - builds domain records        │
        │ - persists via sheets repo     │
        │ - asks reporting svc for KPIs  │
        └─────┬──────────────────────────┘
        ┌─────▼──────────────────────────┐
        │ Google Sheets Repository       │
        │ - writes rows per command      │
        │ - exposes read ranges for KPIs │
        └────────────────────────────────┘
```

Cron-based broadcasting will reuse the reporting service output and WhatsApp service group messaging once the scheduler is wired (config already in place).

## Runtime Flow
1. **Inbound message** arrives at `/webhook` → Gin unmarshals into `WebhookPayload`.
2. **WhatsApp service** extracts text, runs `models.ParseCommand`, and picks the relevant command template.
3. **Command dispatcher** converts arguments to typed records (egg count, feed kg, etc.). It persists the values using the Sheets repository and requests optional analytics from the reporting service.
4. **Reporting service** reads Google Sheets ranges to compute per-period totals or daily/weekly reports.
5. **WhatsApp service** formats the final reply (dispatcher message + helper tips) and sends it using the REST client (`pkg/clients/whatsapp`).

## Google Sheets Schema

Each tab is immutable append-only. Ranges used across the app:

| Sheet       | Range      | Columns (order)                                        |
|-------------|------------|--------------------------------------------------------|
| `Eggs`      | `Eggs!A:C` | Date (ISO), Quantity, Notes                            |
| `Feed`      | `Feed!A:C` | Date, FeedKg, Population                               |
| `Mortality` | `Mortality!A:C` | Date, Quantity, Reason                          |
| `Sales`     | `Sales!A:E`| Date, Client, Quantity, PricePerUnit, Paid             |
| `Expenses`  | `Expenses!A:C` | Date, Label, Amount                              |

Reporting helpers consume the same ranges for aggregates, so keep column order consistent.

## Configuration

The config loader (`internal/config`) reads `.env` + environment variables and validates all required fields. Key settings:

| Variable | Description |
|----------|-------------|
| `APP_PORT` | HTTP port (default `8080`). |
| `WHATSAPP_TOKEN` | Meta access token. |
| `WHATSAPP_PHONE_NUMBER_ID` | Business phone number ID. |
| `META_VERIFY_TOKEN` | Token used during webhook verification. |
| `WHATSAPP_BASE_URL` | API base (default `https://graph.facebook.com`). |
| `WHATSAPP_API_VERSION` | API version (default `v20.0`). |
| `WHATSAPP_GROUP_ID` | Target group for future scheduled broadcasts. |
| `GOOGLE_SHEETS_CREDENTIALS_PATH` | Absolute path to service account JSON. |
| `GOOGLE_SHEET_DATABASE_ID` | Spreadsheet ID holding the farm data. |
| `REPORT_CRON_SCHEDULE` | Cron expression for daily report job (`0 20 * * *`). |
| `TIMEZONE` | Location string for scheduler (default `Africa/Conakry`). |

See `.env.example` for a template.

## Running Locally

```bash
cp .env.example .env        # edit secrets & sheet IDs
go mod tidy                 # download dependencies
go run ./cmd/server         # start the webhook/API server
```

The server prints JSON logs by default; graceful shutdown is triggered with `Ctrl+C`.

## HTTP Endpoints

| Method | Path           | Description |
|--------|----------------|-------------|
| GET    | `/webhook`     | Meta challenge verification. |
| POST   | `/webhook`     | Receive WhatsApp webhook callbacks. |
| POST   | `/send-message`| Send manual/automated outbound message. |
| GET    | `/healthz`     | Simple readiness probe for uptime checks. |

## Payload Examples

### Webhook Verification (GET)
```
GET /webhook?hub.mode=subscribe&hub.verify_token=custom-secret&hub.challenge=CHALLENGE
```

### Incoming Webhook (POST `/webhook`)
```json
{
  "object": "whatsapp_business_account",
  "entry": [
    {
      "id": "123",
      "changes": [
        {
          "value": {
            "messages": [
              {
                "from": "2348012345678",
                "id": "wamid.HBg",
                "timestamp": "1732025600",
                "type": "text",
                "text": { "body": "/eggs 120 trays" }
              }
            ]
          }
        }
      ]
    }
  ]
}
```

### Outbound Manual Message (POST `/send-message`)
```json
{
  "to": "2348012345678",
  "message": "Inventory arrives at noon",
  "preview_url": false
}
```

## Development Notes
- **Logging**: `pkg/logger` provides a production Zap logger; use `logger.Named("component")` to keep scopes clean.
- **Testing**: Run `go test ./...` to ensure all packages compile; unit tests can be added per package (table-driven style recommended).
- **Extending commands**: add new `CommandType`, extend dispatcher to parse/persist, and update WhatsApp replies for worker guidance.
- **Reporting**: `internal/service/reporting` already exposes `GenerateDailyReport`/`GenerateWeeklyReport`; plug these into a cron job + WhatsApp group broadcast when ready.
- **Schedulers**: Config already includes cron + timezone, so wiring robfig/cron or Cloud Scheduler should be straightforward.

Have fun building smarter farms! 🐔

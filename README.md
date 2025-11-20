# Farmer WhatsApp Automation Backend

Smart Poultry Farm automation service that ingests WhatsApp worker updates, classifies structured commands, and replies with actionable prompts for daily operations.

## Features
- ✅ WhatsApp webhook verification and message ingestion
- ✅ Command parsing for `/eggs`, `/feed`, `/mortality`, `/sales`, `/expenses`
- ✅ Automated replies via WhatsApp Cloud API (resty client)
- ✅ Clean layered architecture (config, clients, services, handlers)
- ✅ Structured logging with Zap
- ✅ Google Sheets persistence + lightweight reporting summaries
- ✅ Ready for future integrations (dashboards, IoT, scheduled reports)

## Project Layout
```
cmd/server/main.go          # Application entrypoint
internal/config             # Env loading & validation
internal/domain/models      # DTOs for webhook + commands
internal/server/handlers    # Gin handlers
internal/server/router      # HTTP router wiring
internal/service/whatsapp   # Business logic & orchestration
pkg/clients/whatsapp        # WhatsApp Cloud API rest client
pkg/logger                  # Zap logger helper
```

## Getting Started
1. **Requirements**
  - Go 1.25+
  - WhatsApp Cloud API credentials (token, phone number ID, verify token)
  - Google Cloud service account JSON with Sheets API enabled (read/write)

2. **Setup**
   ```bash
   cp .env.example .env
   # populate the values before running
   ```

3. **Install dependencies & run**
   ```bash
   go mod tidy
   go run ./cmd/server
   ```

4. **Environment variables** (see `.env.example`)
   - `APP_PORT` – HTTP port (default 8080)
   - `WHATSAPP_TOKEN` – Meta access token
   - `WHATSAPP_PHONE_NUMBER_ID` – Phone number ID attached to your WhatsApp Business account
   - `META_VERIFY_TOKEN` – Verification token you configured in Meta dashboard
   - `WHATSAPP_BASE_URL` (optional) – defaults to `https://graph.facebook.com`
   - `WHATSAPP_API_VERSION` (optional) – defaults to `v20.0`
  - `GOOGLE_SHEETS_CREDENTIALS_PATH` – Absolute path to service account JSON credentials
  - `GOOGLE_SHEET_DATABASE_ID` – Target spreadsheet ID to store farm records

## HTTP Endpoints
| Method | Path           | Description |
|--------|----------------|-------------|
| GET    | `/webhook`     | Meta challenge verification |
| POST   | `/webhook`     | Receive WhatsApp webhook callbacks |
| POST   | `/send-message`| Send manual/automated outbound message |
| GET    | `/healthz`     | Simple readiness probe |

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

## Future Work
- Persist parsed command payloads for analytics (Google Sheets, reporting)
- Schedule reminders & integrate IoT sensor alerts
- Detailed telemetry and distributed tracing

Happy farming! 🐔

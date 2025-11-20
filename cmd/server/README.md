# `cmd/server`

Entrypoint for the Farmer WhatsApp automation backend. Everything outside this
folder should be reusable packages; only the `main` package lives here.

## Responsibilities
- Load configuration via `internal/config` and fail fast when required env vars are missing.
- Initialize the structured Zap logger (`pkg/logger`).
- Build infrastructure dependencies: Google Sheets repository, reporting service,
  command dispatcher, WhatsApp service, HTTP handlers/router.
- Start the Gin HTTP server and block until an interrupt/terminate signal arrives.
- Perform graceful shutdown (10s timeout) to drain in-flight requests.

## Dependency Wiring
```
config.Load → GoogleSheetRepository → Reporting Service
                     ↘ commands.Service ↘ WhatsApp Service ↘ handlers/router
```

`main.go` keeps the wiring readable by:
1. Constructing each layer from the bottom (repo) up to the transport.
2. Naming child loggers (e.g. `svc.reporting`) for easier log filtering.
3. Passing a context-aware HTTP server with sensible timeouts (15s read/write, 60s idle).

## Extending the Entrypoint
- Register new services here so they can be injected into handlers.
- Add background jobs (cron, schedulers) near the server boot logic after config is loaded.
- Keep `main.go` glue-only—real business logic belongs under `internal/`.

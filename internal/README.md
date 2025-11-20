# `internal/`

Private application packages that implement the Farmer backend. Go forbids other
modules from importing them, which keeps transport + business logic encapsulated.

## Sub-packages

| Package | Description |
|---------|-------------|
| `config` | Environment loading + validation for server, WhatsApp, Google Sheets, and reporting scheduler settings. |
| `domain` | DTOs and helper structs for WhatsApp payloads, commands, outbound messages, and sheet records. |
| `repository` | Persistence adapters. Currently ships a Google Sheets repository with read/write helpers. |
| `server` | HTTP surface area (Gin router + handlers) that translate HTTP concerns into service calls. |
| `service` | Core business logic: command dispatcher, reporting analytics, and WhatsApp messaging orchestration. |

## Design Conventions
- **Dependency direction**: `service -> repository` (never opposite), `server -> service`, `cmd -> internal/*`.
- **Context first**: every exported function that performs I/O accepts `context.Context` as its first parameter.
- **Logging**: inject `*zap.Logger` into services/repos for scoped, structured logging.
- **Tests**: colocate table-driven tests next to implementation files (not added yet but structure is ready).

Each subfolder contains its own README detailing public APIs and collaboration patterns.

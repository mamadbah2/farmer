# `pkg/`

Reusable helpers that could live outside this module if we decided to publish them. Everything inside `pkg` is import-safe for other modules.

## Layout
| Package | Description |
|---------|-------------|
| `clients/whatsapp` | Thin REST client for the WhatsApp Cloud API built on top of Resty. |
| `logger` | Zap logger factory helpers (`New`, `Must`, `Named`). |

Use `pkg` for infrastructure helpers only—business logic belongs under `internal/`.

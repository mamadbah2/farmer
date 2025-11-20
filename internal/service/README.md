# `internal/service`

Business logic layer orchestrating commands, analytics, and messaging. Services are plain Go structs with explicit dependencies passed through constructors.

## Packages
| Package | Purpose |
|---------|---------|
| `commands` | Parses structured worker updates, persists them to Google Sheets, and returns confirmation strings. |
| `reporting` | Computes aggregates (daily, weekly, ad-hoc summaries) based on sheet data. |
| `whatsapp` | Handles webhook validation, command routing, and outbound replies via the WhatsApp Cloud API client. |

## Common Patterns
- **Constructor helpers**: `NewService(...)` or `NewMetaWhatsAppService(...)` accept interfaces to enable unit testing/mocking.
- **Logging**: every service accepts a `*zap.Logger`. Pass `zap.NewNop()` when logging is optional.
- **Context**: exported methods start with `ctx context.Context` for cancellation/timeout propagation.
- **Error hygiene**: packages expose sentinel errors (e.g. `commands.ErrInvalidArguments`) for precise error handling higher up the stack.

Dive into each subfolder README for detailed APIs.

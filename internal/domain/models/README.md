# `internal/domain/models`

Domain objects shared across services, handlers, and repositories.

## Command Parsing
- `CommandType`: enum for `/eggs`, `/feed`, `/mortality`, `/sales`, `/expenses`, plus `unknown`.
- `Command`: normalized representation with `Type`, original `Raw` string, and tokenized `Args`.
- `ParseCommand(message string)`: trims, lower-cases, strips leading `/`, and returns a `Command` for downstream services.

## WhatsApp Payloads
Mirror Meta's webhook schema so Gin can bind payloads directly:
- `WebhookPayload → Entry → Change → Value` (metadata, contacts, messages, statuses, errors).
- `InboundMessage` captures supported message types (text, interactive, media) but we currently only use text/button/list payloads.
- `MessageStatus`, `WebhookError` are available for future delivery tracking.

## Outbound Contracts
- `OutboundMessageRequest`: request body accepted by `/send-message` endpoint.
- `AutomationReply`: canned responses per command type used by the WhatsApp service.

## Sheet Record DTOs
Although stored in `internal/domain/models`, the structs `EggRecord`, `FeedRecord`, etc. are defined alongside the command dispatcher to align with sheet column ordering. Update both the model definition and dispatcher write logic when the sheet schema evolves.

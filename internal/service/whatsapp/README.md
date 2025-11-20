# `internal/service/whatsapp`

Bridges HTTP handlers with WhatsApp Cloud API while coordinating command dispatching.

## Interfaces
- `MessagingService`: consumed by the handlers. Provides `VerifyWebhookToken`, `HandleWebhook`, and `SendOutbound`.
- `MetaWhatsAppService`: production implementation using the REST client under `pkg/clients/whatsapp`.

## Core Methods
- `VerifyWebhookToken(mode, verifyToken, challenge)`: enforces `mode=subscribe` and compares tokens before returning the challenge string to Meta.
- `HandleWebhook(ctx, payload)`: iterates through entries/changes/messages, extracts text via `extractMessageText`, and routes to `handleInboundMessage`.
- `handleInboundMessage`: parses the text into a `models.Command`, delegates to the command dispatcher, and sends replies. Handles unknown commands + dispatcher errors gracefully.
- `SendOutbound`: manual API for operations to broadcast information without going through command ingestion.

## Command Guidance
`commandReplies` map holds onboarding tips per command. Even when storage fails, workers still receive actionable syntax reminders.

## Timeouts & Reliability
Every outbound call uses `context.WithTimeout(..., 10*time.Second)` to avoid stuck HTTP requests to Meta. Errors are logged with relevant metadata via Zap.

## Extending Functionality
- Add template support by expanding `extractMessageText` to read `msg.Type`.
- Wire group messaging by adding new methods on the WhatsApp client and exposing them here.
- Record telemetry by wrapping `handleInboundMessage` with metrics instrumentation.

# `internal/server`

HTTP surface for the Farmer backend. Built with Gin for lightweight routing.

## Structure
- `handlers/`: request handlers that validate payloads and call services.
- `router/`: central place where middleware + routes are declared.

## WebhookHandler
Methods:
- `Verify`: handles Meta's GET challenge flow. Delegates to `MessagingService.VerifyWebhookToken` and returns the challenge string.
- `Receive`: binds POST payloads into `models.WebhookPayload`, invokes `MessagingService.HandleWebhook`, and surfaces errors with HTTP 500.
- `SendMessage`: exposes a helper endpoint to push outbound notifications using WhatsApp Cloud API.

## Router
`router.New()` configures:
- Release mode Gin engine.
- Panic recovery middleware.
- `zapLoggerMiddleware` to log method/path/status/duration for every request.
- Routes for `/webhook`, `/send-message`, `/healthz`.

## Adding Routes
1. Create a handler method that takes `*gin.Context` and talks to a service.
2. Register the handler in `router.New`.
3. Update README/HTTP docs if the endpoint is public.

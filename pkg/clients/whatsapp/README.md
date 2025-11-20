# `pkg/clients/whatsapp`

Resty-based client for the WhatsApp Cloud API. Keeps HTTP concerns separate from business logic.

## Construction
```go
client := whatsapp.NewClient(cfg.WhatsApp)
```
- Injects headers (`Authorization`, `Content-Type`) and sets a 15s timeout.
- Base URL = `<WHATSAPP_BASE_URL>/<WHATSAPP_API_VERSION>`; `SendTextMessage` appends `/{phoneNumberID}/messages`.

## API
- `SendTextMessage(ctx, SendTextMessageRequest) (*SendTextMessageResponse, error)`
  - `SendTextMessageRequest` contains `To`, `Body`, and `PreviewURL` flag.
  - Returns IDs of created messages or an error containing the Meta API code/message.

## Error Handling
- Uses Resty's `SetError` to deserialize Meta error payloads, then wraps the message/code into a Go error for upstream logging.
- Propagates context cancellation to abort pending HTTP requests.

## Next Steps
- Add media/template send helpers as the bot grows.
- Consider rate-limit/backoff logic if Meta responses warrant retries.

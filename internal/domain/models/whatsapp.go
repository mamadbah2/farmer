package models

// WebhookPayload mirrors the structure sent by Meta's WhatsApp Cloud API webhook callbacks.
type WebhookPayload struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

// WebhookEntry represents one entry payload within the webhook body.
type WebhookEntry struct {
	ID      string          `json:"id"`
	Changes []WebhookChange `json:"changes"`
}

// WebhookChange captures the actual notification contents.
type WebhookChange struct {
	Value WebhookValue `json:"value"`
	Field string       `json:"field"`
}

// WebhookValue contains message metadata, contacts and message events sent by users.
type WebhookValue struct {
	MessagingProduct string           `json:"messaging_product"`
	Metadata         Metadata         `json:"metadata"`
	Contacts         []Contact        `json:"contacts"`
	Messages         []InboundMessage `json:"messages"`
	Statuses         []MessageStatus  `json:"statuses"`
	Errors           []WebhookError   `json:"errors"`
}

// Metadata contains WhatsApp phone identifiers for the business account.
type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

// Contact represents the WhatsApp user initiating the conversation.
type Contact struct {
	Profile ContactProfile `json:"profile"`
	WaID    string         `json:"wa_id"`
}

// ContactProfile contains the human-friendly contact name.
type ContactProfile struct {
	Name string `json:"name"`
}

// InboundMessage aggregates all supported inbound WhatsApp message shapes we care about.
type InboundMessage struct {
	From        string              `json:"from"`
	ID          string              `json:"id"`
	Timestamp   string              `json:"timestamp"`
	Type        string              `json:"type"`
	Text        *TextContent        `json:"text,omitempty"`
	Interactive *InteractiveContent `json:"interactive,omitempty"`
	Image       *MediaContent       `json:"image,omitempty"`
	Audio       *MediaContent       `json:"audio,omitempty"`
	Document    *MediaContent       `json:"document,omitempty"`
}

// TextContent contains text messages body.
type TextContent struct {
	Body string `json:"body"`
}

// InteractiveContent represents button/list replies.
type InteractiveContent struct {
	Type        string       `json:"type"`
	ButtonReply *ButtonReply `json:"button_reply,omitempty"`
	ListReply   *ListReply   `json:"list_reply,omitempty"`
}

// ButtonReply models a pressed button payload.
type ButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ListReply models a selected list item payload.
type ListReply struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// MediaContent represents media attachments minimal metadata.
type MediaContent struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	Sha256   string `json:"sha256"`
	Filename string `json:"filename"`
}

// MessageStatus represents delivery/read receipts coming from WhatsApp.
type MessageStatus struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Timestamp   string `json:"timestamp"`
	RecipientID string `json:"recipient_id"`
}

// WebhookError exposes errors returned from Meta during webhook notifications.
type WebhookError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

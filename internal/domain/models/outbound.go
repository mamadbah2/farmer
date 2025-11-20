package models

// OutboundMessageRequest represents requests to send a message manually via the API.
type OutboundMessageRequest struct {
	To         string `json:"to" binding:"required"`
	Message    string `json:"message" binding:"required"`
	PreviewURL bool   `json:"preview_url"`
}

// AutomationReply describes the response that will be sent back to the worker based on
// the parsed command.
type AutomationReply struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

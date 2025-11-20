package whatsapp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/mamadbah2/farmer/internal/config"
)

// Client exposes WhatsApp Cloud API operations used by the application.
type Client interface {
	SendTextMessage(ctx context.Context, req SendTextMessageRequest) (*SendTextMessageResponse, error)
}

// APIClient is a resty-backed implementation of Client.
type APIClient struct {
	httpClient    *resty.Client
	phoneNumberID string
}

// NewClient builds a WhatsApp API client using the provided configuration values.
func NewClient(cfg config.WhatsAppConfig) *APIClient {
	base := strings.TrimSuffix(cfg.BaseURL, "/")

	restyClient := resty.New()
	restyClient.
		SetBaseURL(fmt.Sprintf("%s/%s", base, cfg.APIVersion)).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken)).
		SetHeader("Content-Type", "application/json").
		SetTimeout(15 * time.Second)

	return &APIClient{
		httpClient:    restyClient,
		phoneNumberID: cfg.PhoneNumberID,
	}
}

// SendTextMessageRequest represents a simplified text message payload.
type SendTextMessageRequest struct {
	To         string
	Body       string
	PreviewURL bool
}

// SendTextMessageResponse mirrors the successful response from Meta.
type SendTextMessageResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// apiError represents a WhatsApp Cloud API error payload.
type apiError struct {
	Error struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorData    any    `json:"error_data"`
		ErrorSubcode int    `json:"error_subcode"`
		FBTraceID    string `json:"fbtrace_id"`
	} `json:"error"`
}

func (c *APIClient) SendTextMessage(ctx context.Context, req SendTextMessageRequest) (*SendTextMessageResponse, error) {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                req.To,
		"type":              "text",
		"text": map[string]any{
			"body":        req.Body,
			"preview_url": req.PreviewURL,
		},
	}

	result := new(SendTextMessageResponse)
	apiErr := new(apiError)

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(payload).
		SetResult(result).
		SetError(apiErr).
		Post(fmt.Sprintf("%s/messages", c.phoneNumberID))
	if err != nil {
		return nil, fmt.Errorf("send whatsapp message: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		message := ""
		code := resp.StatusCode()
		if apiErr != nil {
			message = apiErr.Error.Message
			if apiErr.Error.Code != 0 {
				code = apiErr.Error.Code
			}
		}
		return nil, fmt.Errorf("whatsapp api error: code=%d, message=%s", code, message)
	}

	return result, nil
}

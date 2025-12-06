package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	apiURL     = "https://api.anthropic.com/v1/messages"
	apiVersion = "2023-06-01"
	model      = "claude-3-haiku-20240307"
	maxTokens  = 1024
)

// Client defines the interface for AI text processing.
type Client interface {
	TranslateToCommand(ctx context.Context, input string) (string, error)
	ProcessConversation(ctx context.Context, state ConversationState, input string) (ConversationState, string, error)
}

// ConversationState holds the accumulated data from the user.
type ConversationState struct {
	Step string `json:"step"` // "COLLECTING", "CONFIRMING", "COMPLETED"

	// Data fields
	EggsBand1 *int `json:"eggs_band_1,omitempty"`
	EggsBand2 *int `json:"eggs_band_2,omitempty"`
	EggsBand3 *int `json:"eggs_band_3,omitempty"`

	SalesQty *int `json:"sales_qty,omitempty"` // In trays (alvéoles)

	MortalityQty  *int   `json:"mortality_qty,omitempty"`
	MortalityBand string `json:"mortality_band,omitempty"`

	FeedReceived *bool    `json:"feed_received,omitempty"`
	FeedQty      *float64 `json:"feed_qty,omitempty"`
	Notes        string   `json:"notes,omitempty"`

	// History tracks the conversation context
	History []Message `json:"history,omitempty"`
}

type anthropicClient struct {
	httpClient *resty.Client
}

// NewClient creates a configured Anthropic client.
func NewClient(apiKey string) Client {
	client := resty.New().
		SetHeader("x-api-key", apiKey).
		SetHeader("anthropic-version", apiVersion).
		SetHeader("content-type", "application/json").
		SetTimeout(15 * time.Second)

	return &anthropicClient{httpClient: client}
}

type messageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (c *anthropicClient) TranslateToCommand(ctx context.Context, input string) (string, error) {
	// Legacy method, kept for compatibility if needed, but we are moving to ProcessConversation
	return "", nil
}

func (c *anthropicClient) ProcessConversation(ctx context.Context, state ConversationState, input string) (ConversationState, string, error) {
	// Create a view of state without history for the prompt to avoid token waste/confusion
	promptState := state
	promptState.History = nil
	stateJSON, _ := json.Marshal(promptState)

	systemPrompt := fmt.Sprintf(`You are a helpful farm assistant for a poultry farm. Your job is to collect daily data from the farmer to fill an Excel sheet.
	
	Current State of Data (JSON):
	%s

	The user will send a message. You must update the state based on what they say and generate a reply.
	
	REQUIRED INFORMATION (Ask in this order if missing):
	1. Production (Eggs): Quantity for Band 1, Band 2, and Band 3. (User might give total, ask for breakdown if needed, or if they say "100, 120, 130" assume order 1, 2, 3).
	2. Mortality: Any dead birds? How many and which band? (If 0, that's valid).
	3. Stock/Observations: Did they receive feed? If yes, how many bags? Any problems?

	RULES:
	- CRITICAL: PRESERVE STATE. You MUST copy all existing non-null values from the input "Current State" to the "updated_state" in your response. Never drop existing data.
	- CRITICAL: You MUST update the JSON fields in "updated_state" when the user provides NEW information.
	- CRITICAL: Output valid JSON. Escape newlines in the "reply" string (use \n). Do not put real line breaks inside the string value.
	- If the user provides data, update the JSON fields.
	- If data is missing, your 'reply' should ask for the NEXT missing item in the priority list.
	- If feed_received is true, you MUST ask for "feed_qty" (number of bags) if it is missing.
	- If the user says "Rien a signaler" or "RAS" for observations, set Notes to "RAS".
	- If ALL required fields (Eggs B1-3, Mortality, Feed/Notes) are filled (or explicitly set to 0/None), set the "step" to "COMPLETED".
	- If the user gives all info at once, fill everything and set "step" to "COMPLETED".
	- Your output must be ONLY a JSON object with this structure:
	  {
		"updated_state": {
			"step": "COLLECTING" or "COMPLETED",
			"eggs_band_1": (integer or null),
			"eggs_band_2": (integer or null),
			"eggs_band_3": (integer or null),
			"mortality_qty": (integer or null),
			"mortality_band": (string or ""),
			"feed_received": (boolean or null),
			"feed_qty": (float or null),
			"notes": (string)
		},
		"reply": "Text to send to the farmer"
	  }
	- The 'reply' should be in French, polite, and concise.
	`, string(stateJSON))

	// Append current user message to history
	currentHistory := append(state.History, Message{Role: "user", Content: input})

	// Prefill the assistant response to force JSON
	messagesToSend := append(currentHistory, Message{Role: "assistant", Content: "{"})

	reqBody := messageRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  messagesToSend,
	}

	var respBody messageResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(reqBody).
		SetResult(&respBody).
		Post(apiURL)

	if err != nil {
		return state, "", fmt.Errorf("anthropic api call: %w", err)
	}
	if resp.IsError() {
		return state, "", fmt.Errorf("anthropic api error: %s", resp.String())
	}
	if len(respBody.Content) == 0 {
		return state, "", fmt.Errorf("empty response from ai")
	}

	// Reconstruct the full JSON since we prefilled the opening brace
	responseText := "{" + respBody.Content[0].Text

	fmt.Printf("--- DEBUG AI RESPONSE ---\n%s\n-------------------------\n", responseText)

	// Clean up potential markdown code blocks if Claude wraps the JSON
	responseText = strings.TrimSpace(responseText)
	if strings.HasPrefix(responseText, "```json") {
		responseText = strings.TrimPrefix(responseText, "```json")
		responseText = strings.TrimSuffix(responseText, "```")
	} else if strings.HasPrefix(responseText, "```") {
		responseText = strings.TrimPrefix(responseText, "```")
		responseText = strings.TrimSuffix(responseText, "```")
	}
	responseText = strings.TrimSpace(responseText)

	// Parse the AI response
	var aiResult struct {
		UpdatedState ConversationState `json:"updated_state"`
		Reply        string            `json:"reply"`
	}

	if err := json.Unmarshal([]byte(responseText), &aiResult); err != nil {
		// Fallback if AI didn't return valid JSON (rare with Claude 3 but possible)
		// We return the old state and a generic error message to the user
		return state, "Désolé, je n'ai pas bien compris. Pouvez-vous répéter ?", fmt.Errorf("failed to unmarshal ai response: %w. Response was: %s", err, responseText)
	}

	// Update history in the returned state
	newState := aiResult.UpdatedState
	newState.History = append(currentHistory, Message{Role: "assistant", Content: aiResult.Reply})

	return newState, aiResult.Reply, nil
}

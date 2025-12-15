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
	ProcessConversation(ctx context.Context, state ConversationState, input string, role string) (ConversationState, string, error)
}

// ConversationState holds the accumulated data from the user.
type ConversationState struct {
	Step string `json:"step"` // "COLLECTING", "CONFIRMING", "COMPLETED"

	// Data fields
	EggsBand1 *int `json:"eggs_band_1,omitempty"`
	EggsBand2 *int `json:"eggs_band_2,omitempty"`
	EggsBand3 *int `json:"eggs_band_3,omitempty"`

	SalesQty *int `json:"sales_qty,omitempty"` // In trays (alvéoles)

	MortalityBand1 *int `json:"mortality_band_1,omitempty"`
	MortalityBand2 *int `json:"mortality_band_2,omitempty"`
	MortalityBand3 *int `json:"mortality_band_3,omitempty"`

	FeedReceived *bool    `json:"feed_received,omitempty"`
	FeedQty      *float64 `json:"feed_qty,omitempty"`
	Notes        string   `json:"notes,omitempty"`

	// Seller fields (Abdullah)
	SaleQty        *int     `json:"sale_qty,omitempty"`        // Alveoles vendues
	SalePrice      *float64 `json:"sale_price,omitempty"`      // Prix unitaire
	SaleClient     *string  `json:"sale_client,omitempty"`     // Nom du client
	SalePaid       *float64 `json:"sale_paid,omitempty"`       // Montant payé
	ReceptionQty   *int     `json:"reception_qty,omitempty"`   // Alveoles reçues
	ReceptionPrice *float64 `json:"reception_price,omitempty"` // Prix unitaire réception

	// Expense fields (Saikou)
	ExpenseCategory  *string  `json:"expense_category,omitempty"`
	ExpenseQty       *float64 `json:"expense_qty,omitempty"`
	ExpenseUnitPrice *float64 `json:"expense_unit_price,omitempty"`
	ExpenseNotes     *string  `json:"expense_notes,omitempty"`

	// History tracks the conversation context
	History []Message `json:"history,omitempty"`
}

// Merge updates the current state with non-null values from the new state.
// It ensures that previously collected data is not lost if the AI fails to return it.
func (s *ConversationState) Merge(newState ConversationState) {
	s.Step = newState.Step
	s.History = newState.History

	// Farmer fields
	if newState.EggsBand1 != nil {
		s.EggsBand1 = newState.EggsBand1
	}
	if newState.EggsBand2 != nil {
		s.EggsBand2 = newState.EggsBand2
	}
	if newState.EggsBand3 != nil {
		s.EggsBand3 = newState.EggsBand3
	}
	if newState.MortalityBand1 != nil {
		s.MortalityBand1 = newState.MortalityBand1
	}
	if newState.MortalityBand2 != nil {
		s.MortalityBand2 = newState.MortalityBand2
	}
	if newState.MortalityBand3 != nil {
		s.MortalityBand3 = newState.MortalityBand3
	}
	if newState.FeedReceived != nil {
		s.FeedReceived = newState.FeedReceived
	}
	if newState.FeedQty != nil {
		s.FeedQty = newState.FeedQty
	}
	if newState.Notes != "" {
		s.Notes = newState.Notes
	}

	// Seller fields
	if newState.SaleQty != nil {
		s.SaleQty = newState.SaleQty
	}
	if newState.SalePrice != nil {
		s.SalePrice = newState.SalePrice
	}
	if newState.SaleClient != nil {
		s.SaleClient = newState.SaleClient
	}
	if newState.SalePaid != nil {
		s.SalePaid = newState.SalePaid
	}
	if newState.ReceptionQty != nil {
		s.ReceptionQty = newState.ReceptionQty
	}
	if newState.ReceptionPrice != nil {
		s.ReceptionPrice = newState.ReceptionPrice
	}

	// Expense fields
	if newState.ExpenseCategory != nil {
		s.ExpenseCategory = newState.ExpenseCategory
	}
	if newState.ExpenseQty != nil {
		s.ExpenseQty = newState.ExpenseQty
	}
	if newState.ExpenseUnitPrice != nil {
		s.ExpenseUnitPrice = newState.ExpenseUnitPrice
	}
	if newState.ExpenseNotes != nil {
		s.ExpenseNotes = newState.ExpenseNotes
	}
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

func (c *anthropicClient) ProcessConversation(ctx context.Context, state ConversationState, input string, role string) (ConversationState, string, error) {
	// Create a view of state without history for the prompt to avoid token waste/confusion
	promptState := state
	promptState.History = nil
	stateJSON, _ := json.Marshal(promptState)

	var systemPrompt string

	if role == "seller" {
		systemPrompt = fmt.Sprintf(`You are a helpful assistant for the farm's sales manager (Abdullah). Your job is to collect sales and reception data.
		
		Current State of Data (JSON):
		%s

		REQUIRED INFORMATION (Ask in this order if missing):
		1. Sales: Did you sell eggs? If yes:
		   - Quantity (trays/alvéoles)
		   - Unit Price (per tray)
		   - Client Name
		   - Amount Paid (Montant payé)
		2. Reception: Did you receive eggs? If yes:
		   - Quantity (trays/alvéoles)
		   - Unit Price (if applicable)

		RULES:
		- CRITICAL: PRESERVE STATE. Copy all existing non-null values.
		- CRITICAL: Output valid JSON. The "reply" field MUST be a single line string. Use literal "\n" for line breaks. Do NOT use actual newlines in the string value.
		- If the user provides data, update the JSON fields.
		- If data is missing, ask for the NEXT missing item.
		- If the user says "No sales" or "No reception", you can mark those fields as 0 or handle accordingly.
		- If ALL required fields for the reported activity are filled, set "step" to "COMPLETED".
		- Your output must be ONLY a JSON object with this structure:
		  {
			"updated_state": {
				"step": "COLLECTING" or "COMPLETED",
				"sale_qty": (int or null),
				"sale_price": (float or null),
				"sale_client": (string or null),
				"sale_paid": (float or null),
				"reception_qty": (int or null),
				"reception_price": (float or null),
				"notes": (string)
			},
			"reply": "Text to send to the seller (French)"
		  }
		`, string(stateJSON))
	} else if role == "expense_manager" {
		systemPrompt = fmt.Sprintf(`You are a helpful assistant for the farm's expense manager (Saikou). Your job is to collect expense data.
		
		Current State of Data (JSON):
		%s

		REQUIRED INFORMATION (Ask in this order if missing):
		1. Expense Details:
		   - Category (Rubrique/Dépense)
		   - Quantity
		   - Unit Price
		   - Notes (Motif/Observation)

		RULES:
		- CRITICAL: PRESERVE STATE. Copy all existing non-null values.
		- CRITICAL: Output valid JSON. The "reply" field MUST be a single line string. Use literal "\n" for line breaks. Do NOT use actual newlines in the string value.
		- If the user provides data, update the JSON fields.
		- If data is missing, ask for the NEXT missing item.
		- If ALL required fields for the reported activity are filled, set "step" to "COMPLETED".
		- Your output must be ONLY a JSON object with this structure:
		  {
			"updated_state": {
				"step": "COLLECTING" or "COMPLETED",
				"expense_category": (string or null),
				"expense_qty": (float or null),
				"expense_unit_price": (float or null),
				"expense_notes": (string or null)
			},
			"reply": "Text to send to the expense manager (French)"
		  }
		`, string(stateJSON))
	} else {
		// Default to Farmer (Chaby)
		systemPrompt = fmt.Sprintf(`You are a helpful farm assistant for a poultry farm. Your job is to collect daily data from the farmer to fill an Excel sheet.
		
		Current State of Data (JSON):
		%s

		The user will send a message. You must update the state based on what they say and generate a reply.
		
		REQUIRED INFORMATION (Ask in this order if missing):
		1. Production (Eggs): Quantity for Band 1, Band 2, and Band 3. (User might give total, ask for breakdown if needed, or if they say "100, 120, 130" assume order 1, 2, 3).
		2. Mortality: How many dead birds in Band 1, Band 2, and Band 3? (If 0, that's valid).
		3. Stock/Observations: Did they receive feed? If yes, how many bags? Any problems?

		RULES:
		- CRITICAL: PRESERVE STATE. You MUST copy all existing non-null values from the input "Current State" to the "updated_state" in your response. Never drop existing data.
		- CRITICAL: You MUST update the JSON fields in "updated_state" when the user provides NEW information.
		- CRITICAL: Output valid JSON. The "reply" field MUST be a single line string. Use literal "\n" for line breaks. Do NOT use actual newlines in the string value.
		- If the user provides data, update the JSON fields.
		- If data is missing, your 'reply' should ask for the NEXT missing item in the priority list.
		- If feed_received is true, you MUST ask for "feed_qty" (number of bags) if it is missing.
		- If the user says "Rien a signaler" or "RAS" for observations, set Notes to "RAS".
		- If ALL required fields (Eggs B1-3, Mortality B1-3, Feed/Notes) are filled (or explicitly set to 0/None), set the "step" to "COMPLETED".
		- If the user gives all info at once, fill everything and set "step" to "COMPLETED".
		- IMPORTANT: If the user provides ALL the information in a single message (Eggs, Mortality, Feed), you MUST set "step" to "COMPLETED" immediately.
		- Your output must be ONLY a JSON object with this structure:
		  {
			"updated_state": {
				"step": "COLLECTING" or "COMPLETED",
				"eggs_band_1": (integer or null),
				"eggs_band_2": (integer or null),
				"eggs_band_3": (integer or null),
				"mortality_band_1": (integer or null),
				"mortality_band_2": (integer or null),
				"mortality_band_3": (integer or null),
				"feed_received": (boolean or null),
				"feed_qty": (float or null),
				"notes": (string)
			},
			"reply": "Text to send to the farmer"
		  }
		- The 'reply' should be in French, polite, and concise.
		`, string(stateJSON))
	}

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
		// Attempt to fix common JSON errors (newlines in strings)
		sanitized := sanitizeJSON(responseText)
		if sanitized != responseText {
			if err2 := json.Unmarshal([]byte(sanitized), &aiResult); err2 == nil {
				goto Success
			}
		}

		// Fallback if AI didn't return valid JSON (rare with Claude 3 but possible)
		// We return the old state and a generic error message to the user
		return state, "Désolé, je n'ai pas bien compris. Pouvez-vous répéter ?", fmt.Errorf("failed to unmarshal ai response: %w. Response was: %s", err, responseText)
	}

Success:
	// Update history in the returned state
	newState := aiResult.UpdatedState
	newState.History = append(currentHistory, Message{Role: "assistant", Content: aiResult.Reply})

	return newState, aiResult.Reply, nil
}

func sanitizeJSON(input string) string {
	// Locate the "reply" field
	key := "\"reply\""
	keyIdx := strings.Index(input, key)
	if keyIdx == -1 {
		return input
	}

	// Find the start of the value (first quote after key)
	// input[keyIdx:] starts with "reply"...
	// We need to skip "reply" and find the colon and then the quote.

	// Let's search for the colon after key
	colonIdx := strings.Index(input[keyIdx:], ":")
	if colonIdx == -1 {
		return input
	}

	// Now search for the quote after the colon
	valueStartRel := strings.Index(input[keyIdx+colonIdx:], "\"")
	if valueStartRel == -1 {
		return input
	}

	valueStartAbs := keyIdx + colonIdx + valueStartRel + 1 // +1 to skip the opening quote

	// Find the end of the value. Since we assume "reply" is the last field, we can look for the last quote in the string.
	// But to be safer, we can look for the last quote before the last closing brace.

	lastBraceIdx := strings.LastIndex(input, "}")
	if lastBraceIdx == -1 {
		return input
	}

	valueEndAbs := strings.LastIndex(input[:lastBraceIdx], "\"")
	if valueEndAbs == -1 || valueEndAbs <= valueStartAbs {
		return input
	}

	// Extract content
	content := input[valueStartAbs:valueEndAbs]

	// Escape newlines
	escaped := strings.ReplaceAll(content, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "")

	return input[:valueStartAbs] + escaped + input[valueEndAbs:]
}

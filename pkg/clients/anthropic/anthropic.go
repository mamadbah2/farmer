package anthropic

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	apiURL     = "https://api.anthropic.com/v1/messages"
	apiVersion = "2023-06-01"
	model      = "claude-3-haiku-20240307" // Haiku est rapide et suffisant pour ça
	maxTokens  = 100
)

// Client defines the interface for AI text processing.
type Client interface {
	TranslateToCommand(ctx context.Context, input string) (string, error)
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
		SetTimeout(10 * time.Second)

	return &anthropicClient{httpClient: client}
}

type messageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (c *anthropicClient) TranslateToCommand(ctx context.Context, input string) (string, error) {
	systemPrompt := `You are a farm assistant. Your goal is to translate natural language into specific strict commands.
	
	The available commands are:
	1. /eggs [band1_qty] [band2_qty] [band3_qty] [notes...] (e.g., "/eggs 120 130 110", "/eggs 150 140 160 3 cracked")
	   - There are ALWAYS 3 bands of chickens.
	   - If the user provides a single total, try to infer or ask, but if they list 3 numbers, map them to band 1, 2, and 3 respectively.
	   - If the user says "Band 1: 50, Band 2: 60, Band 3: 70", output "/eggs 50 60 70".
	2. /feed [kg] [population_optional] (e.g., "/feed 25.5", "/feed 50 1000")
	3. /mortality [quantity] [reason...] (e.g., "/mortality 2 heat stress")
	4. /sales [qty] [price_unit] [paid_optional] [client...] (e.g., "/sales 10 25000", "/sales 5 10000 50000 John Doe")
	5. /expenses [amount] [label...] (e.g., "/expenses 50000 vaccines")

	Rules:
	- Output ONLY the command string. No explanations.
	- If the input is vague or doesn't match a command, output "unknown".
	- Convert units if necessary (e.g. if user says "30 trays of eggs", and a tray is 30 eggs, output the total count).
	- Today is the context.
	`

	reqBody := messageRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []message{
			{Role: "user", Content: input},
		},
	}

	var respBody messageResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(reqBody).
		SetResult(&respBody).
		Post(apiURL)

	if err != nil {
		return "", fmt.Errorf("anthropic api call: %w", err)
	}

	if resp.IsError() {
		return "", fmt.Errorf("anthropic api error: %s", resp.String())
	}

	if len(respBody.Content) == 0 {
		return "", fmt.Errorf("empty response from ai")
	}

	return respBody.Content[0].Text, nil
}

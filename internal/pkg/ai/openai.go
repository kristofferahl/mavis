package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/kristofferahl/mavis/internal/pkg/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient implements the Client interface using OpenAI
type OpenAIClient struct {
	client *openai.Client
	config config.OpenAIConfig
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(c config.OpenAIConfig) (*OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &OpenAIClient{
		client: &client,
		config: c,
	}, nil
}

// GenerateFieldDefaults generates default values for fields based on a prepared prompt
func (c *OpenAIClient) GenerateFieldDefaults(ctx context.Context, prompt string) (FieldDefaults, error) {
	log.Debug("generating commit defaults", "client", "openai", "model", c.config.Model)
	defaults := FieldDefaults{}

	chatCompletion, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model:               openai.ChatModel(c.config.Model),
		MaxCompletionTokens: openai.Int(int64(c.config.MaxCompletionTokens)),
		Temperature:         openai.Float(c.config.Temperature),
	})

	if err != nil {
		return defaults, fmt.Errorf("failed to generate commit message: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return defaults, fmt.Errorf("no response from AI")
	}

	if len(chatCompletion.Choices) > 1 {
		log.Debug("multiple AI response choices returned", "count", len(chatCompletion.Choices))
	}

	response := chatCompletion.Choices[0].Message.Content
	if response == "" {
		return defaults, fmt.Errorf("AI response is empty")
	}

	log.Debug(
		"received AI response",
		"response",
		response,
		"total_tokens",
		chatCompletion.Usage.TotalTokens,
	)

	if err := json.Unmarshal([]byte(response), &defaults); err != nil {
		return defaults, fmt.Errorf("failed to unmarshal AI response: %w", err)
	}

	return defaults, nil
}

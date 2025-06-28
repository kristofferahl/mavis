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

// Client wraps the OpenAI client for AI-powered commit message generation
type Client struct {
	openai *openai.Client
}

// NewClient creates a new AI client
func NewClient() *Client {
	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)

	return &Client{
		openai: &client,
	}
}

// GenerateFieldDefaults generates default values for fields based on a git diff
func (c *Client) GenerateFieldDefaults(ctx context.Context, config *config.Config, gitDiff string) error {
	if gitDiff == "" {
		return fmt.Errorf("git diff is empty, nothing to commit")
	}

	fields, err := json.Marshal(config.Fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields to JSON: %w", err)
	}

	customPrompt := os.Getenv("MAVIS_AI_PROMPT")
	prompt := fmt.Sprintf(`Generate default values for the following fields based on the git diff.
Respond with a JSON object where each key is the field Title and the value is the suggested default value.
The response must be a valid JSON object and contain no wrapping characters. %s

Fields (json):
%s

Git diff:
%s`, customPrompt, string(fields), gitDiff)

	log.Debug("generating commit defaults", "prompt", prompt)

	chatCompletion, err := c.openai.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model:       openai.ChatModelGPT4_1Mini, // Cost-effective for simple tasks
		MaxTokens:   openai.Int(500),            // Limit response length
		Temperature: openai.Float(0.2),          // Lower temperature for more consistent output
	})

	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return fmt.Errorf("no response from AI")
	}

	if len(chatCompletion.Choices) > 1 {
		log.Debug("multiple AI response choices returned", "count", len(chatCompletion.Choices))
	}

	response := chatCompletion.Choices[0].Message.Content
	if response == "" {
		return fmt.Errorf("AI response is empty")
	}

	log.Debug(
		"received AI response",
		"response",
		response,
		"total_tokens",
		chatCompletion.Usage.TotalTokens,
	)

	var defaults map[string]interface{}
	if err := json.Unmarshal([]byte(response), &defaults); err != nil {
		return fmt.Errorf("failed to unmarshal AI response: %w", err)
	}

	for _, f := range config.Fields {
		key := f.Title
		if defaultValue, ok := defaults[key]; ok {
			log.Debug("setting default value for field", "field", key, "value", defaultValue)
			f.Default = defaultValue
		}
	}

	return nil
}

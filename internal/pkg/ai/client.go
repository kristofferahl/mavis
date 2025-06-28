package ai

import (
	"context"
	"fmt"

	"github.com/kristofferahl/mavis/internal/pkg/config"
)

type FieldDefaults map[string]interface{}

// Client interface for AI-powered commit message generation
type Client interface {
	GenerateFieldDefaults(ctx context.Context, prompt string) (FieldDefaults, error)
}

// NewClient creates a new AI client based on the provider in config
func NewClient(cfg config.AIConfig) (Client, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAIClient(cfg.OpenAI)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.Provider)
	}
}

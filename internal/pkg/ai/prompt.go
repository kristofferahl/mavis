package ai

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kristofferahl/mavis/internal/pkg/config"
)

// GeneratePrompt creates a prompt for AI-powered commit message generation
func GeneratePrompt(config *config.Config, gitDiff string) (string, error) {
	if gitDiff == "" {
		return "", fmt.Errorf("git diff is empty, nothing to commit")
	}

	fields, err := json.Marshal(config.Fields)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fields to JSON: %w", err)
	}

	// Use config custom prompt with fallback to env var for backward compatibility
	customPrompt := config.AI.CustomPrompt
	if customPrompt == "" {
		customPrompt = os.Getenv("MAVIS_AI_PROMPT")
	}

	prompt := fmt.Sprintf(`Generate default values for the following fields based on the git diff.
Respond with a JSON object where each key is the field Title and the value is the suggested default value.
The response must be a valid JSON object and contain no wrapping characters. %s

Fields (json):
%s

Git diff:
%s`, customPrompt, string(fields), gitDiff)

	return prompt, nil
}

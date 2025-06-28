package ai

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/log"
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

	prompt := fmt.Sprintf(`
# Commit defaults generation prompt
Generate default values for fields based on the git diff.
Respond with a JSON object where each key is the field Title and the value is the suggested default value.
The response must be a valid JSON object and contain no wrapping characters.
%s

## Fields (json):
%s

## Git diff:
`, config.AI.CustomPrompt, string(fields))

	log.Debug("generated base prompt", "prompt", prompt)

	prompt += gitDiff
	return prompt, nil
}

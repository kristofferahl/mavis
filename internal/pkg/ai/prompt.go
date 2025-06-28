package ai

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/kristofferahl/mavis/internal/pkg/config"
)

// GeneratePrompt creates a prompt for AI-powered commit message generation
func GeneratePrompt(config *config.Config, gitDiff string, gitBranch string) (string, error) {
	if gitDiff == "" {
		return "", fmt.Errorf("git diff is empty, nothing to commit")
	}

	fields, err := json.Marshal(config.Fields)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fields to JSON: %w", err)
	}

	prompt := fmt.Sprintf(`# Commit Message Generation Prompt
Generate default values for each field, based on the git diff and branch.
Respond with a JSON object where the key match the field Title and the value is the suggested default value.
The response must be a valid JSON object and contain no wrapping characters.
%s

Fields (json):
%s

Git branch:
%s

Git diff:
%s`, config.AI.CustomPrompt, string(fields), gitBranch, gitDiff)

	log.Debug("generated prompt for AI", "prompt", prompt)

	return prompt, nil
}

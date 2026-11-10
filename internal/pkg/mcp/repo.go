package mcp

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// getRepoPath returns the root of the git repository that contains dir.
func getRepoPath(ctx context.Context, dir string) (string, error) {
	if dir == "" {
		return "", errors.New("directory is required")
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("%s must be an absolute path", dir)
	}

	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("%s is not in a git repository: %s", dir, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("%s is not in a git repository: %w", dir, err)
	}
	return strings.TrimSpace(string(output)), nil
}

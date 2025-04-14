package config

import (
	"os"
	"path/filepath"
	"strings"
)

func resolvePath(path string) (string, error) {
	path = strings.TrimPrefix(path, "path:")
	if strings.HasPrefix(path, "~/") {
		dirname, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(dirname, path[2:])
	}
	return path, nil
}

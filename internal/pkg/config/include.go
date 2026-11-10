package config

import (
	"path/filepath"
	"strings"
)

type Include struct {
	When string `yaml:"when,omitempty" json:"when,omitempty"`
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
}

func (i Include) Match(path string) bool {
	if i.When == "" {
		return true
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	whenPath, err := resolvePath(i.When)
	if err != nil {
		return false
	}
	whenPath, err = filepath.Abs(whenPath)
	if err != nil {
		return false
	}
	return isWithin(path, whenPath)
}

// isWithin reports whether path is dir or inside it, comparing whole path
// segments so that /a/mavis does not match /a/mavis-other.
func isWithin(path, dir string) bool {
	if path == dir {
		return true
	}
	if !strings.HasSuffix(dir, string(filepath.Separator)) {
		dir += string(filepath.Separator)
	}
	return strings.HasPrefix(path, dir)
}

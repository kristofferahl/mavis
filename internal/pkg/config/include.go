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
	return strings.HasPrefix(path, whenPath)
}

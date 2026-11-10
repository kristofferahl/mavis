package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Given a root config with an include scoped to <root>/work,
// the include applies only when the config is read for a directory under it.
func TestRead_IncludesMatchTheGivenDirectory(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	includePath := filepath.Join(root, "work.yaml")

	writeFile(t, includePath, "theme: dracula\n")
	rootPath := filepath.Join(root, "config.yaml")
	writeFile(t, rootPath, fmt.Sprintf(`theme: charm
include:
  - when: %q
    path: %q
`, workDir, includePath))

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"directory inside the include scope applies the include", workDir, "dracula"},
		{"nested directory inside the include scope applies the include", filepath.Join(workDir, "repo", "sub"), "dracula"},
		{"directory outside the include scope keeps the root config", filepath.Join(root, "personal"), "charm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(rootPath)
			if err := c.Read(tt.dir); err != nil {
				t.Fatalf("Read(%q) failed: %v", tt.dir, err)
			}
			if c.Theme != tt.want {
				t.Errorf("theme = %q, want %q", c.Theme, tt.want)
			}
		})
	}
}

// The directory passed to Read decides which includes apply,
// not the working directory of the process.
func TestRead_IgnoresProcessWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	includePath := filepath.Join(root, "work.yaml")

	writeFile(t, includePath, "theme: dracula\n")
	rootPath := filepath.Join(root, "config.yaml")
	writeFile(t, rootPath, fmt.Sprintf(`theme: charm
include:
  - when: %q
    path: %q
`, workDir, includePath))

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workDir)

	c := New(rootPath)
	if err := c.Read(filepath.Join(root, "personal")); err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if c.Theme != "charm" {
		t.Errorf("theme = %q, want %q (process cwd must not affect include matching)", c.Theme, "charm")
	}
}

// An include without a `when` condition applies for every directory.
func TestRead_UnconditionalIncludeAlwaysApplies(t *testing.T) {
	root := t.TempDir()
	includePath := filepath.Join(root, "always.yaml")

	writeFile(t, includePath, "theme: dracula\n")
	rootPath := filepath.Join(root, "config.yaml")
	writeFile(t, rootPath, fmt.Sprintf(`theme: charm
include:
  - path: %q
`, includePath))

	c := New(rootPath)
	if err := c.Read(filepath.Join(root, "anywhere")); err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if c.Theme != "dracula" {
		t.Errorf("theme = %q, want %q", c.Theme, "dracula")
	}
}

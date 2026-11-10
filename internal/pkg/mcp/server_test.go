package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kristofferahl/mavis/internal/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
)

const configWithTemplate = `template: "%s: {{description}}"
fields:
  - type: input
    title: summary
    required: true
    format:
      - key: description
        format: "{{value}}"
`

// writeConfig writes a minimal config whose template starts with prefix.
func writeConfig(t *testing.T, path, prefix string) {
	t.Helper()
	content := strings.Replace(configWithTemplate, "%s", prefix, 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fileLoader reads the config file at path, like the real server does.
func fileLoader(path string) ConfigLoader {
	return func(dir string) (*config.Config, error) {
		c := config.New(path)
		if err := c.Read(dir); err != nil {
			return nil, err
		}
		return c, nil
	}
}

func callTool(t *testing.T, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) (string, bool) {
	t.Helper()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type %T", res.Content[0])
	}
	return text.Text, res.IsError
}

// Given a running server, when the config file changes,
// then the next prepare_commit returns the new template.
func TestPrepareCommit_ReflectsConfigChangesAfterStart(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, cfgPath, "before")
	s := NewServer(fileLoader(cfgPath))

	first, isErr := callTool(t, s.handlePrepareCommit, nil)
	if isErr {
		t.Fatalf("prepare_commit failed: %s", first)
	}

	writeConfig(t, cfgPath, "after")
	second, isErr := callTool(t, s.handlePrepareCommit, nil)
	if isErr {
		t.Fatalf("prepare_commit failed: %s", second)
	}

	var got PrepareCommitResult
	if err := json.Unmarshal([]byte(second), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Template, "after:") {
		t.Errorf("template = %q, want it to start with %q", got.Template, "after:")
	}
}

// Given a running server, when the config file changes,
// then the next preview_commit renders with the new template.
func TestPreviewCommit_ReflectsConfigChangesAfterStart(t *testing.T) {
	t.Chdir(initRepo(t))
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, cfgPath, "before")
	s := NewServer(fileLoader(cfgPath))

	values := map[string]any{"values": `{"summary":"x"}`}
	if out, isErr := callTool(t, s.handlePreviewCommit, values); isErr {
		t.Fatalf("preview_commit failed: %s", out)
	}

	writeConfig(t, cfgPath, "after")
	out, isErr := callTool(t, s.handlePreviewCommit, values)
	if isErr {
		t.Fatalf("preview_commit failed: %s", out)
	}

	var got PreviewCommitResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Message != "after: x" {
		t.Errorf("message = %q, want %q", got.Message, "after: x")
	}
}

// A config that fails to load is reported as a tool error, not a server crash.
func TestTools_ReportConfigLoadErrors(t *testing.T) {
	t.Chdir(initRepo(t))
	s := NewServer(func(string) (*config.Config, error) {
		return nil, errors.New("boom")
	})

	tests := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		args    map[string]any
	}{
		{"prepare_commit", s.handlePrepareCommit, nil},
		{"preview_commit", s.handlePreviewCommit, map[string]any{"values": `{}`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, isErr := callTool(t, tt.handler, tt.args)
			if !isErr {
				t.Fatalf("expected tool error, got %s", out)
			}
			if !strings.Contains(out, "failed to load config: boom") {
				t.Errorf("error %q does not mention the load failure", out)
			}
		})
	}
}

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
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

// git runs a git command in dir and fails the test on error.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// stageFile writes a file in repo and stages it.
func stageFile(t *testing.T, repo, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", name)
}

type handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

func callTool(t *testing.T, h handler, args map[string]any) (string, bool) {
	t.Helper()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type %T", res.Content[0])
	}
	return text.Text, res.IsError
}

func mustCall[T any](t *testing.T, h handler, args map[string]any) T {
	t.Helper()
	out, isErr := callTool(t, h, args)
	if isErr {
		t.Fatalf("tool failed: %s", out)
	}
	var got T
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}
	return got
}

// Given a running server, when the config file changes,
// then the next prepare_commit returns the new template.
func TestPrepareCommit_ReflectsConfigChangesAfterStart(t *testing.T) {
	repo := initRepo(t)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, cfgPath, "before")
	s := NewServer(fileLoader(cfgPath))
	args := map[string]any{"path": repo}

	first := mustCall[PrepareCommitResult](t, s.handlePrepareCommit, args)
	if !strings.HasPrefix(first.Template, "before:") {
		t.Fatalf("first template = %q, want it to start with %q", first.Template, "before:")
	}

	writeConfig(t, cfgPath, "after")
	second := mustCall[PrepareCommitResult](t, s.handlePrepareCommit, args)
	if !strings.HasPrefix(second.Template, "after:") {
		t.Errorf("second template = %q, want it to start with %q", second.Template, "after:")
	}
}

// Given a running server, when the config file changes,
// then the next preview_commit renders with the new template.
func TestPreviewCommit_ReflectsConfigChangesAfterStart(t *testing.T) {
	repo := initRepo(t)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, cfgPath, "before")
	s := NewServer(fileLoader(cfgPath))
	args := map[string]any{"path": repo, "values": `{"summary":"x"}`}

	first := mustCall[PreviewCommitResult](t, s.handlePreviewCommit, args)
	if first.Message != "before: x" {
		t.Fatalf("first message = %q, want %q", first.Message, "before: x")
	}

	writeConfig(t, cfgPath, "after")
	second := mustCall[PreviewCommitResult](t, s.handlePreviewCommit, args)
	if second.Message != "after: x" {
		t.Errorf("second message = %q, want %q", second.Message, "after: x")
	}
}

// Given a server started in one repository,
// when a commit is previewed and approved for another repository,
// then the commit lands in the other repository only.
func TestApproveCommit_CommitsInTheRepositoryGivenByPath(t *testing.T) {
	serverRepo := initRepo(t)
	targetRepo := initRepo(t)
	t.Chdir(serverRepo)
	stageFile(t, targetRepo, "hello.txt")

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, cfgPath, "chore")
	s := NewServer(fileLoader(cfgPath))

	preview := mustCall[PreviewCommitResult](t, s.handlePreviewCommit, map[string]any{
		"path":   targetRepo,
		"values": `{"summary":"add hello file"}`,
	})
	if preview.RepoPath != targetRepo {
		t.Fatalf("preview repo_path = %q, want %q", preview.RepoPath, targetRepo)
	}

	approve := mustCall[ApproveCommitResult](t, s.handleApproveCommit, map[string]any{"id": preview.ID})
	if approve.RepoPath != targetRepo {
		t.Errorf("approve repo_path = %q, want %q", approve.RepoPath, targetRepo)
	}

	if got := git(t, targetRepo, "log", "-1", "--format=%s"); got != "chore: add hello file" {
		t.Errorf("target repo last commit = %q, want %q", got, "chore: add hello file")
	}
	if _, err := exec.Command("git", "-C", serverRepo, "rev-parse", "HEAD").Output(); err == nil {
		t.Errorf("server repo got a commit, want none")
	}
}

// The config is resolved for the repository root,
// even when path points at a directory inside it.
func TestTools_LoadConfigForTheRepositoryRoot(t *testing.T) {
	repo := initRepo(t)
	nested := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, cfgPath, "feat")
	var loadedFor []string
	s := NewServer(func(dir string) (*config.Config, error) {
		loadedFor = append(loadedFor, dir)
		return fileLoader(cfgPath)(dir)
	})

	prepare := mustCall[PrepareCommitResult](t, s.handlePrepareCommit, map[string]any{"path": nested})
	preview := mustCall[PreviewCommitResult](t, s.handlePreviewCommit, map[string]any{"path": nested, "values": `{"summary":"x"}`})

	if prepare.RepoPath != repo || preview.RepoPath != repo {
		t.Errorf("repo_path = %q / %q, want %q", prepare.RepoPath, preview.RepoPath, repo)
	}
	for _, dir := range loadedFor {
		if dir != repo {
			t.Errorf("config loaded for %q, want repo root %q", dir, repo)
		}
	}
}

// prepare_commit and preview_commit refuse to run without a usable path.
func TestTools_RejectMissingOrInvalidPath(t *testing.T) {
	notARepo := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(notARepo))
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, cfgPath, "feat")
	s := NewServer(fileLoader(cfgPath))

	paths := []struct {
		name string
		args map[string]any
		want string
	}{
		{"missing path", map[string]any{}, "missing path parameter"},
		{"relative path", map[string]any{"path": "some/repo"}, "must be an absolute path"},
		{"path outside a repository", map[string]any{"path": notARepo}, "is not in a git repository"},
	}
	tools := []struct {
		name string
		h    handler
	}{
		{"prepare_commit", s.handlePrepareCommit},
		{"preview_commit", s.handlePreviewCommit},
	}
	for _, tool := range tools {
		for _, p := range paths {
			t.Run(tool.name+" with "+p.name, func(t *testing.T) {
				args := map[string]any{"values": `{"summary":"x"}`}
				for k, v := range p.args {
					args[k] = v
				}
				out, isErr := callTool(t, tool.h, args)
				if !isErr {
					t.Fatalf("expected tool error, got %s", out)
				}
				if !strings.Contains(out, p.want) {
					t.Errorf("error %q does not contain %q", out, p.want)
				}
			})
		}
	}
}

// A config that fails to load is reported as a tool error, not a server crash.
func TestTools_ReportConfigLoadErrors(t *testing.T) {
	repo := initRepo(t)
	s := NewServer(func(string) (*config.Config, error) {
		return nil, errors.New("boom")
	})

	tests := []struct {
		name string
		h    handler
		args map[string]any
	}{
		{"prepare_commit", s.handlePrepareCommit, map[string]any{"path": repo}},
		{"preview_commit", s.handlePreviewCommit, map[string]any{"path": repo, "values": `{}`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, isErr := callTool(t, tt.h, tt.args)
			if !isErr {
				t.Fatalf("expected tool error, got %s", out)
			}
			if !strings.Contains(out, "failed to load config: boom") {
				t.Errorf("error %q does not mention the load failure", out)
			}
		})
	}
}

// prepare_commit tells the agent which repository to inspect,
// so it does not run git in its own working directory by mistake.
func TestPrepareCommit_InstructionsTargetTheResolvedRepository(t *testing.T) {
	repo := initRepo(t)
	nested := filepath.Join(repo, "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, cfgPath, "feat")
	s := NewServer(fileLoader(cfgPath))

	got := mustCall[PrepareCommitResult](t, s.handlePrepareCommit, map[string]any{"path": nested})

	for _, want := range []string{
		"Repository: " + repo,
		`git -C "` + repo + `" diff --cached`,
		`git -C "` + repo + `" branch --show-current`,
		"pass this same path to preview_commit",
	} {
		if !strings.Contains(got.Instructions, want) {
			t.Errorf("instructions do not contain %q", want)
		}
	}
	if strings.Contains(got.Instructions, "{{repo_path}}") {
		t.Errorf("instructions still contain the {{repo_path}} placeholder")
	}
}

// The mavis prompt asks the agent to pass the repository path.
func TestMavisPrompt_AsksForTheRepositoryPath(t *testing.T) {
	s := NewServer(fileLoader(""))
	res, err := s.handleMavisPrompt(context.Background(), mcp.GetPromptRequest{})
	if err != nil {
		t.Fatal(err)
	}
	text, ok := res.Messages[0].Content.(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type %T", res.Messages[0].Content)
	}
	if !strings.Contains(text.Text, "absolute path as path to prepare_commit and preview_commit") {
		t.Errorf("prompt %q does not ask for the repository path", text.Text)
	}
}

package mcp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initRepo creates a git repository in a temp dir and returns its resolved root.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	for _, kv := range [][2]string{{"user.name", "mavis test"}, {"user.email", "test@example.com"}, {"commit.gpgsign", "false"}} {
		if out, err := exec.Command("git", "-C", dir, "config", kv[0], kv[1]).CombinedOutput(); err != nil {
			t.Fatalf("git config failed: %v\n%s", err, out)
		}
	}
	// git reports the resolved path (e.g. /private/var on macOS).
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestGetRepoPath_FindsTheRepoContainingTheDirectory(t *testing.T) {
	repo := initRepo(t)
	nested := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		dir  string
	}{
		{"directory at the repo root resolves to the root", repo},
		{"nested directory resolves to the repo root", nested},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRepoPath(context.Background(), tt.dir)
			if err != nil {
				t.Fatalf("getRepoPath(%q) failed: %v", tt.dir, err)
			}
			if got != repo {
				t.Errorf("got %q, want %q", got, repo)
			}
		})
	}
}

// The repo is found from the given directory, not the process working directory.
func TestGetRepoPath_IgnoresProcessWorkingDirectory(t *testing.T) {
	repoA := initRepo(t)
	repoB := initRepo(t)
	t.Chdir(repoA)

	got, err := getRepoPath(context.Background(), repoB)
	if err != nil {
		t.Fatalf("getRepoPath failed: %v", err)
	}
	if got != repoB {
		t.Errorf("got %q, want %q", got, repoB)
	}
}

func TestGetRepoPath_RejectsDirectoriesOutsideARepo(t *testing.T) {
	notARepo := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(notARepo))

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"empty directory is rejected", "", "directory is required"},
		{"relative directory is rejected", "some/repo", "must be an absolute path"},
		{"directory that is not in a repo is rejected", notARepo, "is not in a git repository"},
		{"directory that does not exist is rejected", filepath.Join(notARepo, "missing"), "is not in a git repository"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getRepoPath(context.Background(), tt.dir)
			if err == nil {
				t.Fatalf("getRepoPath(%q) succeeded, want error", tt.dir)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

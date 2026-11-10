package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/kristofferahl/mavis/internal/pkg/commit"
	"github.com/kristofferahl/mavis/internal/pkg/config"
	"github.com/kristofferahl/mavis/internal/pkg/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ConfigLoader loads the mavis config as it applies to dir.
type ConfigLoader func(dir string) (*config.Config, error)

// Server wraps the MCP server with mavis-specific functionality
type Server struct {
	mcpServer  *server.MCPServer
	loadConfig ConfigLoader
	cache      *Cache
}

// NewServer creates a new MCP server for mavis. The config is loaded on
// every tool call so that it reflects the directory the call is made for.
func NewServer(loadConfig ConfigLoader) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer(
			version.Name,
			version.Version,
		),
		loadConfig: loadConfig,
		cache:      NewCache(DefaultTTL),
	}

	s.registerTools()
	s.registerPrompts()
	return s
}

// Serve starts the MCP server over stdio
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	// Tool: prepare_commit
	prepareCommitTool := mcp.NewTool("prepare_commit",
		mcp.WithDescription("Get the commit message template, fields, and instructions for the repository at path. Call this first to understand what values are needed for a commit."),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the git repository to commit in (any directory inside it works)"),
		),
	)
	s.mcpServer.AddTool(prepareCommitTool, s.handlePrepareCommit)

	// Tool: preview_commit
	previewCommitTool := mcp.NewTool("preview_commit",
		mcp.WithDescription("Preview a commit message with the provided field values. Returns the rendered commit message and an approval ID. IMPORTANT: After calling this, you MUST show the message to the user and wait for their explicit approval before calling approve_commit."),
		mcp.WithString("values",
			mcp.Required(),
			mcp.Description("JSON object with field values keyed by field title"),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the git repository to commit in (any directory inside it works)"),
		),
	)
	s.mcpServer.AddTool(previewCommitTool, s.handlePreviewCommit)

	// Tool: approve_commit
	approveCommitTool := mcp.NewTool("approve_commit",
		mcp.WithDescription("Execute a previously previewed commit. ONLY call this after the user has explicitly approved the commit message. Never call this automatically."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("The approval ID returned from preview_commit"),
		),
	)
	s.mcpServer.AddTool(approveCommitTool, s.handleApproveCommit)
}

func (s *Server) registerPrompts() {
	mavisPrompt := mcp.NewPrompt("mavis",
		mcp.WithPromptDescription("Create a commit using mavis"),
	)
	s.mcpServer.AddPrompt(mavisPrompt, s.handleMavisPrompt)
}

func (s *Server) handleMavisPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return mcp.NewGetPromptResult("Create a commit using mavis", []mcp.PromptMessage{
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent("Use the mavis tools to commit staged changes in the repository I am working in. Pass that repository's absolute path as path to prepare_commit and preview_commit. Call prepare_commit first, then preview_commit with appropriate field values, show me the result, and wait for my approval before calling approve_commit."),
		),
	}), nil
}

// resolveRepo reads the required path argument, finds the repository that
// contains it, and loads the config as it applies to that repository's root.
func (s *Server) resolveRepo(ctx context.Context, request mcp.CallToolRequest) (string, *config.Config, *mcp.CallToolResult) {
	path, err := request.RequireString("path")
	if err != nil {
		return "", nil, mcp.NewToolResultError(fmt.Sprintf("missing path parameter: %v", err))
	}

	repoPath, err := getRepoPath(ctx, path)
	if err != nil {
		return "", nil, mcp.NewToolResultError(fmt.Sprintf("failed to get repo path: %v", err))
	}

	cfg, err := s.loadConfig(repoPath)
	if err != nil {
		return "", nil, mcp.NewToolResultError(fmt.Sprintf("failed to load config: %v", err))
	}

	return repoPath, cfg, nil
}

// PrepareCommitResult is the response from prepare_commit
type PrepareCommitResult struct {
	RepoPath     string          `json:"repo_path"`
	Template     string          `json:"template"`
	Fields       []*config.Field `json:"fields"`
	Instructions string          `json:"instructions"`
}

const mcpInstructions = `Generate field values for a git commit message based on STAGED changes only.

Repository: {{repo_path}}
Run every git command against this repository, and pass this same path to preview_commit.

STEP 1 - Gather context (run these commands first):
- git -C "{{repo_path}}" diff --cached         → View the staged changes (this is what you're committing)
- git -C "{{repo_path}}" branch --show-current → Get the current branch name for context

STEP 2 - Analyze the changes:
- Focus on STAGED changes only; ignore unstaged and untracked files
- Consider the branch name: branches like "feat/..." or "fix/..." hint at commit type
- Identify the primary intent: new functionality, bug fix, refactor, docs, chore, etc.
- Determine scope by identifying which area of the codebase is affected

STEP 3 - Generate field values:
- Provide values as a JSON object where keys match field titles exactly
- For "select" fields: use one of the available option key values
- For "confirm" fields: use "true" or "false" (as strings)
- For "input" and "text" fields: use appropriate string values
- Leave optional fields empty ("") if not applicable

Breaking change guidance:
- Mark as breaking if the change removes or renames public APIs, changes function signatures, removes configuration options, or alters expected behavior in ways that require users to update their code`

func (s *Server) handlePrepareCommit(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath, cfg, errResult := s.resolveRepo(ctx, request)
	if errResult != nil {
		return errResult, nil
	}

	instructions := strings.ReplaceAll(mcpInstructions, "{{repo_path}}", repoPath)
	if cfg.AI.CustomPrompt != "" {
		instructions = instructions + "\n\nAdditional guidance:\n" + cfg.AI.CustomPrompt
	}

	result := PrepareCommitResult{
		RepoPath:     repoPath,
		Template:     strings.TrimPrefix(cfg.Template, "\n"),
		Fields:       cfg.Fields,
		Instructions: instructions,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// PreviewCommitResult is the response from preview_commit
type PreviewCommitResult struct {
	ID       string `json:"id"`
	Message  string `json:"message"`
	RepoPath string `json:"repo_path"`
}

func (s *Server) handlePreviewCommit(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	valuesJSON, err := request.RequireString("values")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("missing values parameter: %v", err)), nil
	}

	// Parse the values
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(valuesJSON), &values); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid values JSON: %v", err)), nil
	}

	repoPath, cfg, errResult := s.resolveRepo(ctx, request)
	if errResult != nil {
		return errResult, nil
	}

	// Validate required fields and build template values
	var templateValues []commit.TemplateValue
	for _, field := range cfg.Fields {
		value, ok := values[field.Title]
		if !ok {
			value = nil
		}

		// Check required fields
		if field.Required {
			if value == nil {
				return mcp.NewToolResultError(fmt.Sprintf("missing required field: %s", field.Title)), nil
			}
			if str, ok := value.(string); ok && str == "" {
				return mcp.NewToolResultError(fmt.Sprintf("required field cannot be empty: %s", field.Title)), nil
			}
		}

		// Get template values using the provided value
		if value != nil {
			templateValues = append(templateValues, field.TemplateValuesFrom(value)...)
		} else {
			// Use empty string for missing optional fields
			templateValues = append(templateValues, field.TemplateValuesFrom("")...)
		}
	}

	// Render the commit message
	renderer := commit.NewRenderer(cfg.Template)
	message := renderer.Render(templateValues)

	// Store in cache
	pc := s.cache.Store(repoPath, message)

	result := PreviewCommitResult{
		ID:       pc.ID,
		Message:  message,
		RepoPath: repoPath,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ApproveCommitResult is the response from approve_commit
type ApproveCommitResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	RepoPath string `json:"repo_path"`
}

func (s *Server) handleApproveCommit(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := request.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("missing id parameter: %v", err)), nil
	}

	// Get pending commit
	pc := s.cache.Get(id)
	if pc == nil {
		return mcp.NewToolResultError("commit approval not found or expired, run preview_commit again"), nil
	}

	// Execute git commit
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", pc.Message)
	cmd.Dir = pc.RepoPath

	// Capture output instead of writing to stdout/stderr (which would corrupt MCP stdio protocol)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("git commit failed: %v\n%s", err, string(output))), nil
	}

	// Remove from cache
	s.cache.Remove(id)

	result := ApproveCommitResult{
		Success:  true,
		Message:  pc.Message,
		RepoPath: pc.RepoPath,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

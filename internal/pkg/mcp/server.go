package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kristofferahl/mavis/internal/pkg/commit"
	"github.com/kristofferahl/mavis/internal/pkg/config"
	"github.com/kristofferahl/mavis/internal/pkg/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with mavis-specific functionality
type Server struct {
	mcpServer *server.MCPServer
	config    *config.Config
	cache     *Cache
}

// NewServer creates a new MCP server for mavis
func NewServer(cfg *config.Config) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer(
			version.Name,
			version.Version,
		),
		config: cfg,
		cache:  NewCache(DefaultTTL),
	}

	s.registerTools()
	return s
}

// Serve starts the MCP server over stdio
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	// Tool: prepare_commit
	prepareCommitTool := mcp.NewTool("prepare_commit",
		mcp.WithDescription("Get the commit message template, fields, and instructions. Call this first to understand what values are needed for a commit."),
	)
	s.mcpServer.AddTool(prepareCommitTool, s.handlePrepareCommit)

	// Tool: preview_commit
	previewCommitTool := mcp.NewTool("preview_commit",
		mcp.WithDescription("Preview a commit message with the provided field values. Returns the rendered commit message and an approval ID. IMPORTANT: After calling this, you MUST show the message to the user and wait for their explicit approval before calling approve_commit."),
		mcp.WithString("values",
			mcp.Required(),
			mcp.Description("JSON object with field values keyed by field title"),
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

// PrepareCommitResult is the response from prepare_commit
type PrepareCommitResult struct {
	Template     string          `json:"template"`
	Fields       []*config.Field `json:"fields"`
	Instructions string          `json:"instructions"`
}

const mcpInstructions = `Generate field values for a git commit message based on STAGED changes only.

STEP 1 - Gather context (run these commands first):
- git diff --cached        → View the staged changes (this is what you're committing)
- git branch --show-current → Get the current branch name for context

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
	instructions := mcpInstructions
	if s.config.AI.CustomPrompt != "" {
		instructions = instructions + "\n\nAdditional guidance:\n" + s.config.AI.CustomPrompt
	}

	result := PrepareCommitResult{
		Template:     strings.TrimPrefix(s.config.Template, "\n"),
		Fields:       s.config.Fields,
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

	// Get repo path
	repoPath, err := getRepoPath()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get repo path: %v", err)), nil
	}

	// Validate required fields and build template values
	var templateValues []commit.TemplateValue
	for _, field := range s.config.Fields {
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
	renderer := commit.NewRenderer(s.config.Template)
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("git commit failed: %v", err)), nil
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

// getRepoPath returns the root path of the current git repository
func getRepoPath() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

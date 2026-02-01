# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Run tests
make test

# Build binary to ./dist/
make build

# Build for all platforms (uses goreleaser)
make build-all
```

## Architecture

Mavis is an interactive terminal UI tool for creating structured Git commits with optional AI-powered suggestions.

### Package Structure

- `internal/app/` - CLI entry point using Cobra, handles command-line flags and orchestrates the application flow
- `internal/pkg/config/` - YAML configuration system with support for conditional includes based on working directory
- `internal/pkg/ui/` - Bubble Tea TUI with split-screen layout (form inputs + live commit preview)
- `internal/pkg/commit/` - Template-based commit message renderer using `{{key}}` placeholders
- `internal/pkg/ai/` - AI client abstraction with OpenAI implementation for generating field defaults

### Key Patterns

**Configuration Layering**: The root config (`~/.config/mavis/config.yaml`) supports `include` blocks with `when` conditions that match against the current working directory path, allowing per-project overrides.

**Field System**: Fields define form inputs with formatting rules that map field values to template placeholders. Each field has a `type` (input, text, select, confirm), formatting rules with conditional `when` clauses, and optional default values.

**Template Rendering**: The commit message template uses `{{key}}` placeholders that are replaced by formatted field values. Formatting rules control how each field value transforms into the final output.

**AI Integration**: When AI mode is enabled, the app generates a prompt from the git diff, branch name, and field configuration, then uses the AI client to suggest default values for all fields.

### Dependencies

- **charmbracelet/bubbletea** - TUI framework
- **charmbracelet/huh** - Form components
- **charmbracelet/lipgloss** - Styling
- **spf13/cobra** - CLI framework
- **openai/openai-go** - OpenAI API client

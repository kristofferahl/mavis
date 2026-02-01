package app

import (
	"fmt"
	"os"
	"path"

	"github.com/charmbracelet/log"
	"github.com/kristofferahl/mavis/internal/pkg/config"
	"github.com/kristofferahl/mavis/internal/pkg/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server for AI agent integration",
	Long: `Start a Model Context Protocol (MCP) server that exposes mavis functionality
to AI agents like Claude Code, Kiro, and others.

The server provides three tools:
  - prepare_commit: Get the commit template, fields, and instructions
  - preview_commit: Preview a commit message with provided field values
  - approve_commit: Execute a previously previewed commit

The server runs over stdio and is designed to be used as an MCP server
in your AI agent's configuration.`,
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetReportTimestamp(false)

		configFile, err := mcpConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %w", err)
		}

		// Create config
		c := config.New(configFile)

		// Create config file if it doesn't exist
		if !c.Exists() {
			if err := c.Write(); err != nil {
				return err
			}
		}

		// Read config file
		if err := c.Read(); err != nil {
			return err
		}

		// Apply environment overrides
		if theme := os.Getenv("MAVIS_THEME"); theme != "" {
			c.Theme = theme
		}
		if chip := os.Getenv("MAVIS_CHIP"); chip != "" {
			c.Chip = chip
		}

		// Create and start MCP server
		server := mcp.NewServer(c)
		return server.Serve()
	},
}

func mcpConfigPath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	appConfigDir := path.Join(userConfigDir, "mavis")
	return path.Join(appConfigDir, "config.yaml"), nil
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

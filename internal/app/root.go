package app

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/kristofferahl/mavis/internal/pkg/config"
	"github.com/kristofferahl/mavis/internal/pkg/ui"
	"github.com/kristofferahl/mavis/internal/pkg/version"
	"github.com/spf13/cobra"
)

type RootOptions struct {
	Debug bool
}

var (
	opt RootOptions
)

var rootCmd = &cobra.Command{
	Use:           "mavis",
	Short:         "mavis - unconventional commit",
	SilenceUsage:  true,
	SilenceErrors: false,
	Version:       fmt.Sprintf("%s (commit=%s)", version.Version, version.Commit),
	RunE: func(cmd *cobra.Command, args []string) error {
		if opt.Debug {
			log.SetLevel(log.DebugLevel)
		}

		configFile, err := appConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path, %w", err)
		}

		// create config, store app config path
		c := config.New(configFile)

		// automatically create app config file if it doesn't exist
		if !c.Exists() {
			if err := c.Write(configFile); err != nil {
				return err
			}
		}

		// read app config file
		log.Debug("config", "file", configFile)
		if err := c.Read(configFile); err != nil {
			return err
		}

		p := tea.NewProgram(ui.NewCommitUI(*c))
		model, err := p.Run()
		if err != nil {
			return err
		}
		commitUI, ok := model.(ui.CommitUI)
		if !ok {
			return fmt.Errorf("failed to cast model to CommitUI")
		}

		if *commitUI.Confirm {
			commit := commitUI.Commit
			log.Debug("commit", "string", commit.String(), "lines", commit.Linebreaks())

			args := []string{"commit", "-m", commit.String()}
			log.Debug("git", "args", args)

			cmd := exec.CommandContext(cmd.Context(), "git", args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("git commit failed, %w", err)
			}
		}
		return nil
	},
}

func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetErrPrefix("error: ")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func appConfigPath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir, %w", err)
	}
	appConfigDir := path.Join(userConfigDir, "mavis")
	return path.Join(appConfigDir, "config.yaml"), nil
}

func init() {
	rootCmd.Flags().BoolVarP(&opt.Debug, "debug", "d", false, "run in debug mode")
}

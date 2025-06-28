package app

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/kristofferahl/mavis/internal/pkg/ai"
	"github.com/kristofferahl/mavis/internal/pkg/config"
	"github.com/kristofferahl/mavis/internal/pkg/ui"
	"github.com/kristofferahl/mavis/internal/pkg/version"
	"github.com/spf13/cobra"
)

type RootOptions struct {
	Debug bool
	UseAI bool
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
		log.SetReportTimestamp(false)
		log.SetPrefix(version.Name)
		if opt.Debug {
			log.SetLevel(log.DebugLevel)
		}

		configFile, err := appConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path, %w", err)
		}

		// create config, set the root config path
		c := config.New(configFile)

		// automatically create config file if it doesn't exist
		if !c.Exists() {
			if err := c.Write(); err != nil {
				return err
			}
		}

		// read config file
		if err := c.Read(); err != nil {
			return err
		}

		// env overrides
		theme := os.Getenv("MAVIS_THEME")
		if len(theme) > 0 {
			log.Debug("overriding theme from env", "theme", theme)
			c.Theme = theme
		}
		chip := os.Getenv("MAVIS_CHIP")
		if len(chip) > 0 {
			log.Debug("overriding chip from env", "chip", chip)
			c.Chip = chip
		}

		if !c.UseAI {
			c.UseAI = opt.UseAI
		}

		if c.UseAI {
			log.Debug("AI mode enabled")
			done := ui.Spin("AI mode enabled, generating commit message...")
			gitDiff, err := exec.CommandContext(cmd.Context(), "git", "diff", "--cached").Output()
			if err != nil {
				done(fmt.Errorf("failed to get git diff, %w", err))
				return nil
			}
			err = ai.NewClient().GenerateFieldDefaults(cmd.Context(), c, string(gitDiff))
			if err != nil {
				done(fmt.Errorf("failed to generate defaults, %w", err))
				return nil
			}
			done(nil)
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
	rootCmd.Flags().BoolVarP(&opt.UseAI, "ai", "", false, "use AI to generate commit message")
}

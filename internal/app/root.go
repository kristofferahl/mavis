package app

import (
	"fmt"
	"os"
	"os/exec"

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

		c := config.New()
		c.Theme = os.Getenv("MAVIS_THEME")

		p := tea.NewProgram(ui.NewCommitUI(c))
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
			log.Debug("commit", "string", commit.String())

			args := []string{"commit", "-m", commit.Summary()}
			if commit.HasBody() {
				args = append(args, "-m", commit.Body())
			}

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

func init() {
	rootCmd.Flags().BoolVarP(&opt.Debug, "debug", "d", false, "run in debug mode")
}

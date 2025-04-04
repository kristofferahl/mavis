package app

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/kristofferahl/mavis/internal/pkg/commit"
	"github.com/kristofferahl/mavis/internal/pkg/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
		if !opt.Debug {
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		}

		var err error
		var ok bool
		var commit commit.Commit

		ok = true

		collect := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("what type of commit is it?").
					Value(&commit.Type).
					Options(
						huh.NewOption("feat", "feat").Selected(true),
						huh.NewOption("fix", "fix"),
						huh.NewOption("chore", "chore"),
					),

				huh.NewInput().
					Title("what is the scope of the commit?").
					// A scope MUST consist of a noun describing a section of the codebase
					Description("noun describing a section of the codebase, e.g. (api, ui, etc.)").
					Value(&commit.Scope),

				huh.NewInput().
					Title("summarize the change").
					Value(&commit.Description),

				huh.NewText().
					Title("describe the change in detail").
					Value(&commit.OptionalBody),

				huh.NewConfirm().
					Title("is it a breaking change?").
					Value(&commit.Breaking),
			),
		).WithTheme(huh.ThemeCharm()).WithShowHelp(true)

		err = collect.Run()
		if err != nil {
			return err
		}

		approve := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("commit changes?").
					Description(commit.String()).
					Value(&ok),
			),
		).WithTheme(huh.ThemeCharm()).WithShowHelp(true)

		err = approve.Run()
		if err != nil {
			return err
		}

		if ok {
			fmt.Println(commit.String())

			args := []string{"commit", "-m", commit.Summary()}
			if commit.HasBody() {
				args = append(args, "-m", commit.Body())
			}

			log.Info().Msgf("git %v", args)

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

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		NoColor:    true,
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&opt.Debug, "debug", "d", false, "run in debug mode")
}

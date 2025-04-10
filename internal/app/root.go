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
		c.Chip = os.Getenv("MAVIS_CHIP")

		c.Template = `
{{type}}{{scope}}{{breaking_glyph}}: {{description}}

{{breaking_body}}{{body}}`

		c.Fields = append(c.Fields, &config.Field{
			Type:    "select",
			Title:   "type of commit",
			Default: "feat",
			Formatting: []config.FormattingRule{
				{
					Key:    "type",
					Format: "{{value}}",
				},
			},
			Options: []config.SelectOption{
				{
					Key:   "feat",
					Value: "feat",
				},
				{
					Key:   "fix",
					Value: "fix",
				},
				{
					Key:   "chore",
					Value: "chore",
				},
			},
		})
		c.Fields = append(c.Fields, &config.Field{
			Type:        "input",
			Title:       "scope of the commit",
			Description: "noun describing a section of the codebase",
			Placeholder: "e.g. api, ui, app etc.",
			Formatting: []config.FormattingRule{
				{
					Key:    "scope",
					Format: "({{value}})",
				},
			},
		})
		c.Fields = append(c.Fields, &config.Field{
			Type:        "input",
			Title:       "summary of the change",
			Description: "a short description of the change",
			Placeholder: "e.g. add config file",
			Required:    true,
			Formatting: []config.FormattingRule{
				{
					Key:    "description",
					Format: "{{value}}",
				},
			},
		})
		c.Fields = append(c.Fields, &config.Field{
			Type:        "confirm",
			Title:       "breaking change?",
			Description: "if yes, describe the breaking change in detail",
			Formatting: []config.FormattingRule{
				{
					Key:    "breaking_glyph",
					Format: "!",
					When:   "true",
				},
				{
					Key:    "breaking_glyph",
					Format: "",
					When:   "false",
				},
				{
					Key:    "breaking_body",
					Format: "BREAKING CHANGE: ",
					When:   "true",
				},
				{
					Key:    "breaking_body",
					Format: "",
					When:   "false",
				},
			},
		})
		c.Fields = append(c.Fields, &config.Field{
			Type:        "text",
			Title:       "describe the change in detail (optional)",
			Description: "what is the motivation for this change",
			Formatting: []config.FormattingRule{
				{
					Key:    "body",
					Format: "{{value}}",
				},
			},
		})

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

func init() {
	rootCmd.Flags().BoolVarP(&opt.Debug, "debug", "d", false, "run in debug mode")
}

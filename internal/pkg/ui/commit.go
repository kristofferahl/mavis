package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kristofferahl/mavis/internal/pkg/commit"
	"github.com/kristofferahl/mavis/internal/pkg/config"
	"github.com/kristofferahl/mavis/internal/pkg/version"
)

func NewCommitUI(config config.Config) tea.Model {
	var theme *huh.Theme
	switch config.Theme {
	case "base":
		theme = huh.ThemeBase()
	case "base16":
		theme = huh.ThemeBase16()
	case "catppuccin":
		theme = huh.ThemeCatppuccin()
	case "dracula":
		theme = huh.ThemeDracula()
	case "charm":
		theme = huh.ThemeCharm()
	default:
		theme = huh.ThemeCharm()
	}
	theme.Focused.Card = theme.Focused.Card.PaddingLeft(2)
	theme.Focused.Base = theme.Focused.Base.PaddingLeft(2).BorderStyle(lipgloss.HiddenBorder())

	style := CommitUIStyle{
		Normal:    theme.Focused.Base.GetForeground(),
		Subtle:    theme.Focused.Description.GetForeground(),
		Highlight: theme.Focused.Title.GetForeground(),
		Padding:   2,
	}
	style.Doc = lipgloss.NewStyle().Padding(1, style.Padding, 1, style.Padding)
	style.Base = theme.Focused.Base.Padding(0).Margin(0).Border(lipgloss.HiddenBorder(), false)
	style.Border = lipgloss.NormalBorder()

	commit := &commit.Commit{}
	okay := false

	inputs := []huh.Field{
		huh.NewSelect[string]().
			Title("type of commit").
			Value(&commit.Type).
			Options(
				huh.NewOption("feat", "feat").Selected(true),
				huh.NewOption("fix", "fix"),
				huh.NewOption("chore", "chore"),
			),

		huh.NewInput().
			Title("scope for the commit").
			Description("noun describing a section of the codebase, e.g. (api, ui, etc.)").
			Value(&commit.Scope),

		huh.NewInput().
			Title("summary of the change").
			Description("a short description of the change").
			Value(&commit.Description).
			Validate(func(s string) error {
				if len(s) < 3 {
					return fmt.Errorf("must be at least 3 characters")
				}
				return nil
			}),

		huh.NewText().
			Title("describe the change in detail (optional)").
			Description("e.g. what is the motivation for this change? why was it necessary?").
			Value(&commit.OptionalBody).
			ShowLineNumbers(true).
			Lines(3).
			WithHeight(5),

		// .WithButtonAlignment(lipgloss.Left)
		// https://github.com/charmbracelet/huh/pull/427
		huh.NewConfirm().
			Title("is it a breaking change?").
			Value(&commit.Breaking),

		// .WithButtonAlignment(lipgloss.Left)
		// https://github.com/charmbracelet/huh/pull/427
		huh.NewConfirm().
			Title("commit changes?").
			Value(&okay),
	}

	groups := make([]*huh.Group, len(inputs))

	for i, input := range inputs {
		input.Focus()
		groups[i] = huh.NewGroup(input)
	}

	return CommitUI{
		Commit: commit,
		form: huh.
			NewForm(groups...).
			WithTheme(theme).
			WithShowHelp(true),
		Confirm: &okay,

		width: 0,
		style: style,
		index: 0,
	}
}

type CommitUI struct {
	Commit  *commit.Commit
	Confirm *bool

	quitting bool
	width    int
	style    CommitUIStyle
	form     *huh.Form
	index    int
}

type CommitUIStyle struct {
	Normal    lipgloss.TerminalColor
	Subtle    lipgloss.TerminalColor
	Highlight lipgloss.TerminalColor

	Doc     lipgloss.Style
	Base    lipgloss.Style
	Border  lipgloss.Border
	Padding int
}

func (m CommitUI) Init() tea.Cmd {
	return m.form.Init()
}

func (m CommitUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	var current tea.Model

	current = m.form

	// Process the form
	current, cmd := current.Update(msg)
	form, ok := current.(*huh.Form)
	if ok {
		cmds = append(cmds, cmd)
		if form.State == huh.StateCompleted {
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

func (m CommitUI) View() string {
	s := m.style
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return s.Base.Foreground(s.Highlight).Render("loading...")
	}

	fullWidth := m.width - s.Doc.GetPaddingLeft() - s.Doc.GetPaddingRight()
	form := m.form

	if m.Confirm != nil {
		done := form.State == huh.StateCompleted
		if done {
			return ""
		}
	}

	doc := strings.Builder{}

	// Title
	{
		var (
			titleStyle = s.Base.
				Width(m.width - s.Doc.GetPaddingLeft() - s.Doc.GetPaddingRight()).
				Foreground(s.Highlight).
				BorderStyle(s.Border).
				BorderBottom(true).
				BorderForeground(s.Subtle)
		)
		title := titleStyle.Render(version.Name + " 🐱 " + version.Description + ", v." + version.Version)
		doc.WriteString(title + "\n")
	}

	// Input & Preview
	{
		var (
			width = (fullWidth / 2)
			col   = lipgloss.NewStyle().Width(width)
		)
		inputCol := col.
			Padding(0).
			BorderStyle(s.Border).
			BorderForeground(s.Subtle).
			BorderRight(true)
		previewCol := col.
			Padding(0, s.Padding+1)

		input := inputCol.Render(s.Base.Render(form.WithWidth(width).View()))
		preview := previewCol.Render(m.Commit.String())

		row := lipgloss.JoinHorizontal(lipgloss.Top, input, preview)
		doc.WriteString(row + "\n")
	}

	// Okay, let's render it
	return s.Doc.Render(doc.String()) + "\n"
}

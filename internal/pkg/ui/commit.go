package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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

	style := commitUIStyle{
		Normal:    theme.Focused.Base.GetForeground(),
		Subtle:    theme.Focused.Description.GetForeground(),
		Highlight: theme.Focused.Title.GetForeground(),
		Padding:   2,
	}
	style.Doc = lipgloss.NewStyle().Padding(1, style.Padding, 1, style.Padding)
	style.Base = theme.Focused.Base.Padding(0).Margin(0).Border(lipgloss.HiddenBorder(), false)
	style.Border = lipgloss.NormalBorder()

	commit := commit.NewRenderer(config.Template)
	okay := true

	// Fields
	fields := make([]huh.Field, 0)
	for _, f := range config.Fields {
		switch f.Type {
		case "input":
			v := ""
			if f.Default != nil {
				v = fmt.Sprintf("%v", f.Default)
			}
			i := huh.NewInput().
				Title(f.Title).
				Description(f.Description).
				Placeholder(f.Placeholder).
				Value(&v).
				Validate(func(s string) error {
					if f.Required && len(s) < 1 {
						return fmt.Errorf("must not be empty")
					}
					return nil
				})

			f.SetRef(i)
			fields = append(fields, i)

		case "text":
			v := ""
			if f.Default != nil {
				v = fmt.Sprintf("%v", f.Default)
			}
			i := huh.NewText().
				Title(f.Title).
				Description(f.Description).
				Placeholder(f.Placeholder).
				Value(&v).
				Validate(func(s string) error {
					if f.Required && len(s) < 1 {
						return fmt.Errorf("must not be empty")
					}
					return nil
				}).
				ShowLineNumbers(true).
				Lines(3).
				WithHeight(5)

			f.SetRef(i)
			fields = append(fields, i)

		case "select":
			v := ""
			if f.Default != nil {
				v = fmt.Sprintf("%v", f.Default)
			}
			opts := make([]huh.Option[string], 0)
			for _, opt := range f.Options {
				key := opt.Key
				if len(opt.Key) == 0 {
					key = opt.Value
				}
				o := huh.NewOption(key, opt.Value)
				if opt.Value == v {
					o.Selected(true)
				}
				opts = append(opts, o)
			}
			i := huh.NewSelect[string]().
				Title(f.Title).
				Description(f.Description).
				Options(opts...).
				Value(&v).
				Validate(func(s string) error {
					if f.Required && len(s) < 1 {
						return fmt.Errorf("must not be empty")
					}
					return nil
				})

			f.SetRef(i)
			fields = append(fields, i)

		case "confirm":
			v := false
			if f.Default != nil {
				v = f.Default.(bool)
			}
			i := huh.NewConfirm().
				Title(f.Title).
				Description(f.Description).
				Value(&v)

			f.SetRef(i)
			fields = append(fields, i)
		}
	}

	// .WithButtonAlignment(lipgloss.Left)
	// https://github.com/charmbracelet/huh/pull/427
	fields = append(fields, huh.NewConfirm().
		Title("commit changes?").
		Value(&okay))

	groups := make([]*huh.Group, len(fields))

	for i, input := range fields {
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

		config: config,
		style:  style,
		keys:   keys,
		help:   help.New(),
	}
}

type CommitUI struct {
	Commit  *commit.Renderer
	Confirm *bool

	config config.Config
	style  commitUIStyle
	keys   keyMap
	help   help.Model

	quitting bool
	width    int
	form     *huh.Form
}

type commitUIStyle struct {
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
		m.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Commit):
			q := true
			m.Confirm = &q
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit):
			q := false
			m.Confirm = &q
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
			chipStyle = s.Base.
					Foreground(lipgloss.Color("#FFF")).
					Background(s.Highlight).
					Padding(0, 1).
					MarginRight(1)
		)
		title := titleStyle.Render(version.Name + " 🐱 " + version.Description + ", v" + version.Version)
		chip := ""
		if m.config.Chip != "" {
			chip = chipStyle.Render(m.config.Chip)
		}
		doc.WriteString(chip + title + "\n")
	}

	// Input & Preview
	{
		var (
			width = (fullWidth / 2)
			col   = lipgloss.NewStyle().Width(width)
			data  = make([]commit.TemplateValue, 0)
		)
		for _, field := range m.config.Fields {
			data = append(data, field.TemplateValues()...)
		}
		inputCol := col.
			Padding(0).
			BorderStyle(s.Border).
			BorderForeground(s.Subtle).
			BorderRight(true)
		previewCol := col.
			Padding(0, s.Padding+1)

		input := inputCol.Render(form.WithWidth(width).View())
		preview := previewCol.Render(m.Commit.Render(data))

		row := lipgloss.JoinHorizontal(lipgloss.Top, input, preview)
		doc.WriteString(row + "\n")
	}

	// Help
	helpView := m.help.View(m.keys)
	if helpView != "" {
		doc.WriteString(helpView + "\n")
	}

	// Okay, let's render it
	return s.Doc.Render(doc.String()) + "\n"
}

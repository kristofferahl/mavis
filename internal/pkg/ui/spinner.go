package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/kristofferahl/mavis/internal/pkg/version"
)

type spinnerError error

type spinnerCompleted struct{}

type spinnerModel struct {
	spinner  spinner.Model
	quitting bool
	text     string
	err      error
}

func newSpinner(text string) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return spinnerModel{
		text:    text,
		spinner: s,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerCompleted:
		m.quitting = true
		m.text = m.text + " completed"
		return m, tea.Quit

	case spinnerError:
		m.err = msg
		m.quitting = true
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m spinnerModel) View() string {
	text := m.text
	if m.err != nil {
		text = version.Name + " error: " + m.err.Error()
	}
	str := fmt.Sprintf("\n  %s %s\n", m.spinner.View(), text)
	if m.quitting {
		return str
	}
	return str
}

func Spin(text string) func(err error) {
	p := tea.NewProgram(newSpinner(text))
	go func() {
		if _, err := p.Run(); err != nil {
			log.Error("failed to run spinner, %v", err)
		}
	}()

	return func(err error) {
		if err != nil {
			p.Send(spinnerError(err))
		} else {
			p.Send(spinnerCompleted{})
		}
		p.Wait()
	}
}

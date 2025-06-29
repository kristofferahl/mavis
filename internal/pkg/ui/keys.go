package ui

import "github.com/charmbracelet/bubbles/key"

var keys = keyMap{
	Commit: key.NewBinding(
		key.WithKeys("ctrl+a", "ctrl+s"),
		key.WithHelp("ctrl+a / ctrl+s", "accept preview and commit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc / ctrl+c", "quit"),
	),
}

type keyMap struct {
	Commit key.Binding
	Help   key.Binding
	Quit   key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Commit, k.Help, k.Quit}, // first column
	}
}

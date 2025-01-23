package main

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	filepicker   filepicker.Model
	selectedFile string
	quitting     bool
	err          error

	height  int
	width   int
	padding int
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m model) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case clearErrorMsg:
		m.err = nil
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = path
	}

	if didSelect, _ := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		m.err = errors.New("Only markdown files supported")
		m.selectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}
func (m model) View() string {
	containerStyle := lipgloss.NewStyle().
		Align(lipgloss.Center).Padding(0, m.padding)

	contentStyle := lipgloss.NewStyle().
		Width(m.width - 2*m.padding).
		Height(m.height / 2).
		Align(lipgloss.Left)

	if m.quitting {
		return ""
	}

	var s strings.Builder
	s.WriteString("\n ")
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.selectedFile == "" {
		s.WriteString(" Pick a file:")
	} else {
		// Do some rendering [replace | navigate to pager view]
		s.WriteString(" Selected file: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
	}

	s.WriteString("\n" + m.filepicker.View() + "\n")
	paddedContet := containerStyle.Render(contentStyle.Render(s.String()))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, paddedContet)
}

func main() {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".md"}
	fp.CurrentDirectory, _ = os.Getwd()
	fp.KeyMap = filepicker.KeyMap{
		GoToTop:  key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "first")),
		GoToLast: key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "last")),
		Down:     key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),
		Up:       key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		PageUp:   key.NewBinding(key.WithKeys("K", "pgup"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("J", "pgdown"), key.WithHelp("pgdown", "page down")),
		Back:     key.NewBinding(key.WithKeys("-", "h", "backspace", "left", "esc"), key.WithHelp("h", "back")),
		Open:     key.NewBinding(key.WithKeys("l", "right", "enter"), key.WithHelp("l", "open")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	}

	p := tea.NewProgram(
		model{
			filepicker: fp,
			padding:    5,
		},
	)

	if _, err := p.Run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

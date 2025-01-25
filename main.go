package main

import (
	"bufio"
	"errors"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	picker = "filepicker"
	md     = "markdown"
)

type model struct {
	filepicker   filepicker.Model
	selectedFile string
	quitting     bool
	err          error
	currentPage  string

	viewport viewport.Model

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

		if m.currentPage == md {
			m.viewport.Width = msg.Width - 2*m.padding
			m.viewport.Height = msg.Height - 2*m.padding
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			m.currentPage = picker
			return m, nil
		case "up", "k":
			if m.currentPage == md {
				m.viewport.LineUp(1)
			}
		case "down", "j":
			if m.currentPage == md {
				m.viewport.LineDown(1)
			}
		case "pgup":
			if m.currentPage == md {
				m.viewport.LineUp(m.viewport.Height)
			}
		case "pgdown":
			if m.currentPage == md {
				m.viewport.LineDown(m.viewport.Height)
			}
		}

	case clearErrorMsg:
		m.err = nil
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = path
		m.err = nil
		m.currentPage = md
	}

	if didSelect, _ := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		m.err = errors.New("Only markdown files supported")
		m.selectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(1*time.Second))
	}
	if m.currentPage == md {
		m.viewport = viewport.New(int(float64(m.width)*0.4), m.height)
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, cmd
}
func (m model) View() string {
	switch m.currentPage {
	case picker:
		return m.FilePickerView()
	case md:
		return m.MarkdownView()
	}
	return "you shouldn't be seeing this"
}

func (m model) FilePickerView() string {
	containerStyle := lipgloss.NewStyle().Padding(0, m.padding).Width(int(float64(m.width) * 0.4))

	if m.quitting {
		return ""
	}

	var s strings.Builder
	s.WriteString("\n ")
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else {
		s.WriteString(" Pick a file:")
	}

	s.WriteString("\n" + m.filepicker.View() + "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, containerStyle.Render(s.String()))
}

func (m model) MarkdownView() string {
	containerStyle := lipgloss.NewStyle().Padding(0, m.padding).Width(int(float64(m.width) * 0.4))
	f, err := os.Open(m.selectedFile)
	if err != nil {
		return "An error occured:\n\t" + err.Error()
	}
	defer f.Close()

	var contents strings.Builder
	title := path.Base(m.selectedFile)
	contents.WriteString("\n" + title + "\n\n")
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		contents.WriteString(sc.Text() + "\n")
	}
	if sc.Err() != nil {
		return "An error occured:\n\t" + sc.Err().Error()
	}
	m.viewport.SetContent(containerStyle.Render(contents.String()))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewport.View())
}
func renderMD(content string) string {
	return content
}

func main() {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".md"}
	dir, err := os.Getwd()
	if err != nil {
		slog.Error(err.Error())
	}
	fp.CurrentDirectory = dir
	fp.KeyMap.Back = key.NewBinding(
		key.WithKeys("-", "backspace", "left", "esc"),
		key.WithHelp("h", "back"))

	p := tea.NewProgram(
		model{
			filepicker:  fp,
			padding:     5,
			currentPage: picker,
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

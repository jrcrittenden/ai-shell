package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DialogModel struct {
	viewport    viewport.Model
	Command     string
	Reason      string
	width       int
	height      int
	keymap      DialogKeyMap
	style       lipgloss.Style
}

type DialogKeyMap struct {
	Approve key.Binding
	Reject  key.Binding
	Edit    key.Binding
}

func DefaultDialogKeyMap() DialogKeyMap {
	return DialogKeyMap{
		Approve: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "approve"),
		),
		Reject: key.NewBinding(
			key.WithKeys("n", "N"),
			key.WithHelp("n", "reject"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e", "E"),
			key.WithHelp("e", "edit"),
		),
	}
}

func NewDialog(command, reason string) *DialogModel {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD"))

	return &DialogModel{
		viewport: vp,
		Command:  command,
		Reason:   reason,
		keymap:   DefaultDialogKeyMap(),
		style:    lipgloss.NewStyle().Padding(1, 2),
	}
}

func (m DialogModel) Init() tea.Cmd {
	return nil
}

func (m DialogModel) Update(msg tea.Msg) (DialogModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Add debug info to content
		debug := fmt.Sprintf("DEBUG: Dialog received window size: %dx%d\n", msg.Width, msg.Height)
		m.viewport.SetContent(debug)
		
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4  // Account for padding
		m.viewport.Height = msg.Height - 4 // Account for padding
		m.updateContent()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Approve):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.Reject):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.Edit):
			return m, tea.Quit
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DialogModel) View() string {
	// Create a centered dialog box
	dialog := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 2).
		Width(m.width - 4).  // Leave some margin
		Height(m.height - 4). // Leave some margin
		Render(m.viewport.View())

	// Center the dialog in the terminal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (m *DialogModel) updateContent() {
	// Add debug info
	debug := fmt.Sprintf("DEBUG: Updating dialog content with size: %dx%d\n", m.width, m.height)
	
	// Calculate available height for content
	contentHeight := m.height - 6 // Account for padding and borders

	// Create the content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		debug,
		lipgloss.NewStyle().Bold(true).Render("Command to execute:"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Render(m.Command),
		"",
		lipgloss.NewStyle().Bold(true).Render("Reason:"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#87ceeb")).Render(m.Reason),
		"",
		lipgloss.NewStyle().Bold(true).Render("Options:"),
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.keymap.Approve.Help().Key,
			" - approve, ",
			m.keymap.Reject.Help().Key,
			" - reject, ",
			m.keymap.Edit.Help().Key,
			" - edit",
		),
	)

	// Set viewport content and dimensions
	m.viewport.SetContent(content)
	m.viewport.Width = m.width - 4  // Account for padding
	m.viewport.Height = contentHeight
} 
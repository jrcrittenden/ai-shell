// model.go — core TUI loop  -----------------------------------------------
package main

import (
	"context"
	"fmt"
	//"time"

	"github.com/jrcrittenden/ai-shell/internal/tui"
	"github.com/jrcrittenden/ai-shell/llm"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the application state
type Model struct {
	client     llm.Client
	input      textinput.Model
	output     textarea.Model
	history    []llm.Message
	showDialog bool
	dialog     *tui.DialogModel
	width      int
	height     int
}

// NewModel creates a new model
func NewModel(c llm.Client) Model {
	// Create input
	in := textinput.New()
	in.Placeholder = "Type a message..."
	in.Focus()
	in.Prompt = "> "
	in.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#87ceeb"))
	in.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

	// Create output area
	out := textarea.New()
	out.Placeholder = "AI output will appear here..."
	out.ShowLineNumbers = false
	out.SetWidth(78)  // Slightly smaller width
	out.SetHeight(18) // Slightly smaller height
	out.FocusedStyle.CursorLine = lipgloss.NewStyle()
	out.ShowLineNumbers = false
	out.Prompt = ""
	out.Placeholder = "AI output will appear here..."
	out.FocusedStyle.Prompt = lipgloss.NewStyle()
	out.BlurredStyle.Prompt = lipgloss.NewStyle()
	out.Blur() // Make it non-focusable

	return Model{
		client:     c,
		input:      in,
		output:     out,
		history:    []llm.Message{},
		showDialog: false,
		dialog:     nil,
		width:      80,
		height:     20,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model accordingly
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update dimensions
		m.width = msg.Width
		m.height = msg.Height - 2 // Leave room for input
		m.output.SetWidth(m.width - 2)  // Leave margin on right
		m.output.SetHeight(m.height - 2) // Leave margin on top

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			// Get the current input value
			input := m.input.Value()
			if input == "" {
				return m, nil
			}

			// Add user message to history
			m.history = append(m.history, llm.Message{
				Role:    "user",
				Content: input,
			})

			// Clear the input
			m.input.Reset()

			// Start streaming response
			chunks := m.client.Stream(context.Background(), m.history)
			go func() {
				for chunk := range chunks {
					if chunk.Text != "" {
						m.output.InsertString(chunk.Text)
					}
					if chunk.ToolCall != nil {
						// Create and show dialog
						m.dialog = tui.NewDialog(chunk.ToolCall.Command, chunk.ToolCall.Reason)
						m.showDialog = true
					}
				}
			}()
		}

		// Update the input
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	case llm.Chunk:
		// Handle text content
		if msg.Text != "" {
			m.output.InsertString(msg.Text)
		}

		// Check for tool call
		if msg.ToolCall != nil {
			// Create and show dialog
			m.dialog = tui.NewDialog(msg.ToolCall.Command, msg.ToolCall.Reason)
			m.showDialog = true
		}

		// Add the AI response to history
		if msg.Text != "" {
			m.history = append(m.history, llm.Message{
				Role:    "assistant",
				Content: msg.Text,
			})
		}
	}

	// Update the output area (but don't process key events)
	if _, ok := msg.(tea.KeyMsg); !ok {
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	if m.showDialog && m.dialog != nil {
		// Create a centered dialog box
		dialog := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2).
			Width(m.width - 4).  // Leave some margin
			Height(m.height - 4). // Leave some margin
			Render(m.dialog.View())

		// Center the dialog in the terminal
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
	}

	// Style the output area
	outputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(0, 1).  // Reduced padding
		Width(m.width - 2).  // Leave margin on right
		Height(m.height - 2) // Leave margin on top

	// Style the input area
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#87ceeb")).
		Padding(0, 1).
		Width(m.width - 2)  // Match output width

	// Combine input and output with proper styling
	return fmt.Sprintf("%s\n%s",
		outputStyle.Render(m.output.View()),
		inputStyle.Render(m.input.View()),
	)
}

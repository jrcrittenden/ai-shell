// model.go — core TUI loop  -----------------------------------------------
package main

import (
	"context"
	"fmt"
	//"time"

	"github.com/jrcrittenden/ai-shell/internal/tui"
	"github.com/jrcrittenden/ai-shell/llm"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/rmhubbert/bubbletea-overlay"
)

// streamCmd represents a command that streams chunks from a channel
type streamCmd struct {
	chunks chan llm.Chunk
}

// streamChunks creates a command that reads from the chunks channel
func streamChunks(chunks chan llm.Chunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-chunks
		if !ok {
			return nil
		}
		return chunk
	}
}

/* --------------------------------------------------------------------- */
/*  Modes & custom messages                                              */
/* --------------------------------------------------------------------- */

type Mode int

const (
	ModeAI Mode = iota
	ModeBash
)

func (m Mode) String() string {
	if m == ModeAI {
		return "AI"
	}
	return "Bash"
}

type (
	AIResponseMsg struct {
		PlainText string
		ToolCall  *llm.ToolCall
	}
	ExecOutputMsg string
	ErrMsg        struct{ Err error }
)

/* --------------------------------------------------------------------- */
/*  Keymap                                                               */
/* --------------------------------------------------------------------- */

type keymap struct {
	Toggle key.Binding
	Run    key.Binding
	Quit   key.Binding
}

func defaultKeymap() keymap {
	return keymap{
		Toggle: key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "switch AI↔bash")),
		Run:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send/exec")),
		Quit:   key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
	}
}

/* --------------------------------------------------------------------- */
/*  Model                                                                */
/* --------------------------------------------------------------------- */

// Model represents the application state
type Model struct {
	client     llm.Client
	input      textinput.Model
	output     viewport.Model
	history    []llm.Message
	showDialog bool
	dialog     *tui.DialogModel
	width      int
	height     int
	mode       Mode
	keys       keymap
	aiContent  string
	bashOutput string
	chunkChan  chan llm.Chunk
	overlay    *overlay.Model
}

// appendToOutput adds text to the current output and updates the viewport
func (m *Model) appendToOutput(text string) {
	var content string
	if m.mode == ModeAI {
		if m.aiContent != "" {
			m.aiContent += "\n"
		}
		m.aiContent += text
		content = m.aiContent
	} else {
		if m.bashOutput != "" {
			m.bashOutput += "\n"
		}
		m.bashOutput += text
		content = m.bashOutput
	}
	m.output.SetContent(content)
	m.output.GotoBottom()
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

	// Create output viewport
	vp := viewport.New(78, 15)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#87ceeb")).
		Padding(0, 1)

	// Create base model
	m := Model{
		client:     c,
		input:      in,
		output:     vp,
		history:    []llm.Message{},
		showDialog: false,
		dialog:     nil,
		width:      80,
		height:     20,
		mode:       ModeAI,
		keys:       defaultKeymap(),
		chunkChan:  make(chan llm.Chunk),
	}

	// Initialize overlay with empty dialog
	m.overlay = overlay.New(nil, &m, overlay.Center, overlay.Center, 0, 0)

	return m
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
		m.output.Width = m.width - 2  // Leave margin on right
		m.output.Height = m.height - 4 // Leave more margin on top

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+t":
			// Toggle mode
			if m.mode == ModeAI {
				m.mode = ModeBash
				m.input.Placeholder = "Enter bash command..."
			} else {
				m.mode = ModeAI
				m.input.Placeholder = "Type a message..."
			}
		case "enter":
			// Get the current input value
			input := m.input.Value()
			if input == "" {
				return m, nil
			}

			if m.mode == ModeAI {
				// Add user message to history
				m.history = append(m.history, llm.Message{
					Role:    "user",
					Content: input,
				})

				// Add user input to output
				m.appendToOutput("> " + input)

				// Clear the input
				m.input.Reset()

				// Create a new channel for this stream
				m.chunkChan = make(chan llm.Chunk)

				// Start streaming response
				chunks := m.client.Stream(context.Background(), m.history)
				go func() {
					defer close(m.chunkChan)
					for chunk := range chunks {
						m.chunkChan <- chunk
					}
				}()
				// Add the streaming command
				cmds = append(cmds, streamChunks(m.chunkChan))
			} else {
				// Bash mode - execute command
				// TODO: Implement command execution
				m.appendToOutput("$ " + input)
			}
		}

		// Update the input
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	case llm.Chunk:
		// Handle text content
		if msg.Text != "" {
			m.appendToOutput(msg.Text)
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

		// Continue streaming if we have more chunks
		cmds = append(cmds, streamChunks(m.chunkChan))
	}

	// Update the viewport
	var cmd tea.Cmd
	m.output, cmd = m.output.Update(msg)
	cmds = append(cmds, cmd)

	// Update the overlay
	if m.overlay != nil {
		m.overlay.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

// BaseModel represents the main application view
type BaseModel struct {
	content string
	width   int
	height  int
}

func (m BaseModel) Init() tea.Cmd {
	return nil
}

func (m BaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m BaseModel) View() string {
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(m.content)
}

// DialogModel represents the dialog view
type DialogModel struct {
	content string
	width   int
	height  int
}

func (m DialogModel) Init() tea.Cmd {
	return nil
}

func (m DialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m DialogModel) View() string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 2).
		Width(m.width).
		Height(m.height).
		Render(m.content)
}

// View renders the UI
func (m Model) View() string {
	// Add mode indicator
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87ceeb")).
		Padding(0, 1)

	// Base view with input and output
	baseView := fmt.Sprintf("%s\n%s\n%s",
		modeStyle.Render(fmt.Sprintf("[%s]", m.mode)),
		m.output.View(),
		m.input.View(),
	)

	if m.showDialog && m.dialog != nil {
		// Create dialog content
		dialogContent := fmt.Sprintf("Command: %s\n\nReason: %s",
			m.dialog.Command,
			m.dialog.Reason,
		)

		// Create models for overlay
		baseModel := BaseModel{
			content: baseView,
			width:   m.width,
			height:  m.height,
		}
		dialogModel := DialogModel{
			content: dialogContent,
			width:   m.width / 2,    // Half the terminal width
			height:  m.height / 3,   // One third of the terminal height
		}

		// Create a new overlay with the dialog
		m.overlay = overlay.New(&dialogModel, &baseModel, overlay.Center, overlay.Center, 0, 0)
		return m.overlay.View()
	}

	return baseView
}

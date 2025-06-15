// model.go — core TUI loop  -----------------------------------------------
package main

import (
	"context"
	"fmt"
	//"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	executil "github.com/jrcrittenden/ai-shell/internal/exec"
	"github.com/jrcrittenden/ai-shell/internal/tui"
	"github.com/jrcrittenden/ai-shell/llm"
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
	client             llm.Client
	input              textinput.Model
	output             viewport.Model
	history            []llm.Message
	showDialog         bool
	dialog             *tui.DialogModel
	width              int
	height             int
	mode               Mode
	keys               keymap
	aiContent          string
	bashOutput         string
	chunkChan          chan llm.Chunk
	overlay            *overlay.Model
	baseModel          BaseModel
	dialogModel        DialogModel
	awaitingDenyReason bool
	denyCommand        string
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

// startStream begins streaming from the LLM using the current history.
func (m *Model) startStream() tea.Cmd {
	m.chunkChan = make(chan llm.Chunk)
	chunks := m.client.Stream(context.Background(), m.history)
	go func() {
		defer close(m.chunkChan)
		for chunk := range chunks {
			m.chunkChan <- chunk
		}
	}()
	return streamChunks(m.chunkChan)
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
		client:             c,
		input:              in,
		output:             vp,
		history:            []llm.Message{},
		showDialog:         false,
		dialog:             nil,
		width:              80,
		height:             20,
		mode:               ModeAI,
		keys:               defaultKeymap(),
		chunkChan:          make(chan llm.Chunk),
		awaitingDenyReason: false,
		denyCommand:        "",
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
	case tea.KeyMsg:
		if m.showDialog {
			switch msg.String() {
			case "left", "right":
				dialogModel := m.dialogModel
				updated, cmd := dialogModel.Update(msg)
				m.dialogModel = updated.(DialogModel)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			case "enter":
				dialogModel := m.dialogModel
				updated, cmd := dialogModel.Update(msg)
				m.dialogModel = updated.(DialogModel)
				cmds = append(cmds, cmd)

				if m.dialogModel.selected == 1 {
					// Approve
					m.showDialog = false
					cmdStr := m.dialog.Command
					m.appendToOutput("$ " + cmdStr)
					out, err := executil.RunCommand(context.Background(), cmdStr)
					if out != "" {
						m.appendToOutput(out)
					}
					if err != nil {
						m.appendToOutput(err.Error())
					}
					m.history = append(m.history, llm.Message{Role: "user", Content: fmt.Sprintf("Command `%s` output:\n%s", cmdStr, out)})
					cmds = append(cmds, m.startStream())
				} else if m.dialogModel.selected == 2 {
					// Deny
					m.showDialog = false
					m.awaitingDenyReason = true
					m.denyCommand = m.dialog.Command
					m.input.Placeholder = "Reason for denial..."
					m.input.Focus()
				}
				return m, tea.Batch(cmds...)
			}
		}

		// Handle other keys only if not in dialog
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.awaitingDenyReason {
				reason := m.input.Value()
				m.appendToOutput(fmt.Sprintf("[DENIED] %s\nReason: %s", m.denyCommand, reason))
				m.history = append(m.history, llm.Message{Role: "user", Content: fmt.Sprintf("Denied command `%s`: %s", m.denyCommand, reason)})
				m.input.Reset()
				m.input.Placeholder = "Type a message..."
				m.awaitingDenyReason = false
				m.denyCommand = ""
				cmds = append(cmds, m.startStream())
				return m, tea.Batch(cmds...)
			}
			if m.mode == ModeAI {
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

				// Add user input to output
				m.appendToOutput("> " + input)

				// Clear the input
				m.input.Reset()

				cmds = append(cmds, m.startStream())
			} else {
				// Bash mode - execute command
				// TODO: Implement command execution
				m.appendToOutput("$ " + m.input.Value())
			}
		case "esc":
			if m.mode == ModeBash {
				m.mode = ModeAI
				m.input.SetValue("")
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.output.Width = msg.Width
		m.output.Height = msg.Height - 2 // Leave room for input
		m.input.Width = msg.Width

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

	// Update input
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	cmds = append(cmds, inputCmd)

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
	content  string
	width    int
	height   int
	selected int // 0: none, 1: approve, 2: deny
}

func (m DialogModel) Init() tea.Cmd {
	return nil
}

func (m DialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			if m.selected > 1 {
				m.selected--
			}
		case "right":
			if m.selected < 2 {
				m.selected++
			}
		case "enter":
			// selection handled by parent model
			return m, nil
		}
	}
	return m, nil
}

func (m DialogModel) View() string {
	// Style for buttons
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#87ceeb"))

	// Style for selected button
	selectedStyle := buttonStyle.Copy().
		BorderForeground(lipgloss.Color("#874BFD")).
		Foreground(lipgloss.Color("#874BFD"))

	// Create buttons
	approveBtn := "✓ Approve"
	denyBtn := "✗ Deny"

	// Apply selection styling
	if m.selected == 1 {
		approveBtn = selectedStyle.Render(approveBtn)
	} else {
		approveBtn = buttonStyle.Render(approveBtn)
	}
	if m.selected == 2 {
		denyBtn = selectedStyle.Render(denyBtn)
	} else {
		denyBtn = buttonStyle.Render(denyBtn)
	}

	// Create button row
	buttonRow := lipgloss.JoinHorizontal(
		lipgloss.Center,
		approveBtn,
		lipgloss.NewStyle().Padding(0, 2).Render(""),
		denyBtn,
	)

	// Combine content and buttons
	content := fmt.Sprintf("%s\n\n%s", m.content, buttonRow)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 2).
		Width(m.width).
		Height(m.height).
		Render(content)
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
		// Lazily create and then reuse the overlay so that the
		// dialog's selection state persists across renders.
		// Update dialog and base model content
		dialogContent := fmt.Sprintf("Command: %s\n\nReason: %s",
			m.dialog.Command,
			m.dialog.Reason,
		)

		m.baseModel.content = baseView
		m.baseModel.width = m.width
		m.baseModel.height = m.height

		if m.overlay == nil {
			m.dialogModel = DialogModel{
				content:  dialogContent,
				width:    m.width / 2,
				height:   m.height / 3,
				selected: 1,
			}
			m.overlay = overlay.New(&m.dialogModel, &m.baseModel, overlay.Center, overlay.Center, 0, 0)
		} else {
			m.dialogModel.content = dialogContent
			m.overlay.Foreground = &m.dialogModel
			m.overlay.Background = &m.baseModel
		}

		return m.overlay.View()
	}

	m.overlay = nil
	return baseView
}

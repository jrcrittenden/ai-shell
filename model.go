// model.go — core TUI loop  -----------------------------------------------
package main

import (
	"context"
	"fmt"
	//"os"
	"os/exec"
	"strings"
	//"sync"
	"time"

	"github.com/jrcrittenden/ai-shell/llm"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	//"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

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

type Model struct {
	mode    Mode
	client  llm.Client
	aiView  viewport.Model
	bashView viewport.Model
	localView viewport.Model
	aiContent string
	bashContent string
	localContent string
	input   textinput.Model
	keys    keymap
	program *tea.Program
}

func NewModel(c llm.Client) *Model {
	in := textinput.New()
	in.Focus()
	in.Placeholder = ""
	in.Prompt = "> "
	in.CharLimit = 2048

	// Create separate viewports for each mode
	aiVp := viewport.New(80, 24)
	aiVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD"))

	bashVp := viewport.New(80, 24)
	bashVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#87ceeb"))

	localVp := viewport.New(80, 24)
	localVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#98fb98"))

	return &Model{
		mode:    ModeAI,
		client:  c,
		aiView:  aiVp,
		bashView: bashVp,
		localView: localVp,
		aiContent: "",
		bashContent: "",
		localContent: "",
		input:   in,
		keys:    defaultKeymap(),
	}
}

func (m *Model) Init() tea.Cmd {
	// Initialize all viewports
	cmds := []tea.Cmd{
		textinput.Blink,
		m.aiView.Init(),
		m.bashView.Init(),
		m.localView.Init(),
	}
	return tea.Batch(cmds...)
}

/* --------------------------------------------------------------------- */
/*  Update                                                               */
/* --------------------------------------------------------------------- */

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Set viewport size, leaving room for input and mode badge
		m.aiView.Width = msg.Width
		m.aiView.Height = msg.Height - 4 // Leave room for input, placeholder, mode badge, and border
		m.bashView.Width = msg.Width
		m.bashView.Height = msg.Height - 4 // Leave room for input, placeholder, mode badge, and border
		m.localView.Width = msg.Width
		m.localView.Height = msg.Height - 4 // Leave room for input, placeholder, mode badge, and border

		// Update all viewports
		var cmd tea.Cmd
		m.aiView, cmd = m.aiView.Update(msg)
		cmds = append(cmds, cmd)
		m.bashView, cmd = m.bashView.Update(msg)
		cmds = append(cmds, cmd)
		m.localView, cmd = m.localView.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Toggle):
			m.mode = (m.mode + 1) % 2
			return m, nil

		case key.Matches(msg, m.keys.Run):
			line := strings.TrimSpace(m.input.Value())
			if line == "" {
				return m, nil
			}
			
			// Store the command before clearing input
			cmdStr := line
			m.input.Reset()
			
			if m.mode == ModeBash {
				// Show the command being executed
				m.appendHistory(cmdStr)
				return m, func() tea.Msg {
					cmd := exec.Command("bash", "-c", cmdStr)
					out, err := cmd.CombinedOutput()
					if err != nil {
						return ErrMsg{Err: err}
					}
					return ExecOutputMsg(out)
				}
			}
			return m, askAI(m.client, line)
		}

		// let textinput consume everything else
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	case ExecOutputMsg:
		output := string(msg)
		if output != "" {
			m.appendHistory(output)
		}

	case ErrMsg:
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555"))
		m.appendHistory(red.Render("error: " + msg.Err.Error()))

	case AIResponseMsg:
		if msg.ToolCall != nil {
			m.appendHistory(fmt.Sprintf("AI ➜ `%s`  — %s",
				msg.ToolCall.Command, msg.ToolCall.Reason))
		}
		if msg.PlainText != "" {
			m.appendHistory(msg.PlainText)
		}
		return m, nil
	}

	// Update all viewports with the message
	var cmd tea.Cmd
	m.aiView, cmd = m.aiView.Update(msg)
	cmds = append(cmds, cmd)
	m.bashView, cmd = m.bashView.Update(msg)
	cmds = append(cmds, cmd)
	m.localView, cmd = m.localView.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

/* --------------------------------------------------------------------- */
/*  View                                                                 */
/* --------------------------------------------------------------------- */

func (m *Model) View() string {
	// Get the appropriate viewport based on mode
	var vp viewport.Model
	switch m.mode {
	case ModeAI:
		vp = m.aiView
	case ModeBash:
		vp = m.bashView
	default:
		vp = m.localView
	}

	// Create the mode badge with right alignment
	modeBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#87ceeb")).
		Align(lipgloss.Right).
		Width(vp.Width).
		Render("[" + m.mode.String() + "]")

	// Create a placeholder line
	placeholder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("[placeholder]")

	// Combine everything with proper spacing
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		modeBadge,
		vp.View(),
		m.input.View(),
		placeholder,
	)
}

/* --------------------------------------------------------------------- */
/*  Helpers                                                              */
/* --------------------------------------------------------------------- */

func (m *Model) appendHistory(s string) {
	// Get the appropriate viewport and content based on mode
	var vp *viewport.Model
	var content *string
	switch m.mode {
	case ModeAI:
		vp = &m.aiView
		content = &m.aiContent
	case ModeBash:
		vp = &m.bashView
		content = &m.bashContent
	default:
		vp = &m.localView
		content = &m.localContent
	}

	// Add new content
	if *content != "" {
		*content += "\n"
	}
	*content += s
	
	// Update viewport
	vp.SetContent(*content)
	vp.GotoBottom()
}

/* --------------------------------------------------------------------- */
/*  Cmd wrappers                                                         */
/* --------------------------------------------------------------------- */

func askAI(c llm.Client, userPrompt string) tea.Cmd {
	hist := []llm.Message{{Role: "user", Content: userPrompt}}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		for chunk := range c.Stream(ctx, hist) {
			if chunk.Err != nil {
				return ErrMsg{Err: chunk.Err}
			}
			if chunk.ToolCall != nil {
				return AIResponseMsg{ToolCall: chunk.ToolCall}
			}
			if chunk.Text != "" {
				return AIResponseMsg{PlainText: chunk.Text}
			}
		}
		return nil
	}
}

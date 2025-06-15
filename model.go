// model.go — core TUI loop  -----------------------------------------------
package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jrcrittenden/ai-shell/llm"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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
	history viewport.Model
        logBuf	strings.Builder
	input   textinput.Model
	keys    keymap
}

func NewModel(c llm.Client) Model {
	in := textinput.New()
	in.Focus()
	in.Placeholder = ""
	in.Prompt = "> "
	in.CharLimit = 2048

	vp := viewport.New(0, 0)
	vp.HighPerformanceRendering = true

	return Model{
		mode:    ModeAI,
		client:  c,
		history: vp,
		input:   in,
		keys:    defaultKeymap(),
	}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

/* --------------------------------------------------------------------- */
/*  Update                                                               */
/* --------------------------------------------------------------------- */

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.history.Width, m.history.Height = msg.Width, msg.Height-2

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Toggle):
			m.mode = (m.mode + 1) % 2
			return m, nil

		case key.Matches(msg, m.keys.Run):
			line := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			if line == "" {
				return m, nil
			}
			if m.mode == ModeBash {
				return m, runCmd(line)
			}
			return m, askAI(m.client, line)
		}

		// let textinput consume everything else
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case AIResponseMsg:
		if msg.ToolCall != nil {
			m.appendHistory(fmt.Sprintf("AI ➜ `%s`  — %s",
				msg.ToolCall.Command, msg.ToolCall.Reason))
		}
		if msg.PlainText != "" {
			m.appendHistory(msg.PlainText)
		}

	case ExecOutputMsg:
		m.appendHistory(string(msg))

	case ErrMsg:
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555"))
		m.appendHistory(red.Render("error: " + msg.Err.Error()))
	}
	return m, nil
}

/* --------------------------------------------------------------------- */
/*  View                                                                 */
/* --------------------------------------------------------------------- */

func (m Model) View() string {
	modeBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#87ceeb")).
		Render("[" + m.mode.String() + "]")

	return fmt.Sprintf("%s\n%s\n%s",
		m.history.View(),
		m.input.View(),
		modeBadge,
	)
}

/* --------------------------------------------------------------------- */
/*  Helpers                                                              */
/* --------------------------------------------------------------------- */

func (m *Model) appendHistory(s string) {
	m.logBuf.WriteString(s)
	m.logBuf.WriteByte('\n')
        m.history.SetContent(m.logBuf.String())
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

func runCmd(cmdLine string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("bash", "-c", cmdLine).CombinedOutput()
		if err != nil {
			return ErrMsg{Err: err}
		}
		return ExecOutputMsg(out)
	}
}

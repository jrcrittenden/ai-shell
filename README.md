# AI Shell Scaffold

Tiny Bubble Tea TUI that lets you flip between a normal bash prompt **and** an AI prompt.
The AI can suggest commands, which you can approve or reject.  
Back‑end is swappable:

* **OpenAI‑compatible** (`--backend openai`) – api.openai.com, LocalAI, vLLM, Groq…  
* **Local Operator** (`--backend localop`) – any running `local-operator serve` instance.

## Quick start

```bash
git clone https://github.com/you/ai-shell.git
cd ai-shell
go run . --backend openai --model gpt-4o
```

or

```bash
local-operator serve &
go run . --backend localop --model default
```

Keybinds:

| Key          | Action              |
|--------------|---------------------|
| `Ctrl+T`     | Toggle AI ↔ Bash    |
| `Enter`      | Send prompt / run   |
| `Ctrl+C` `q` | Quit                |

## Files

* `main.go` – flags + Bubble Tea program boot
* `model.go` – core TUI logic
* `llm/` – backend‑agnostic LLM interface, OpenAI & Local Operator drivers
* `go.mod` – module + deps


---

Original Specification

1. Core Tech Stack
Layer	Lib / Tool	Why it fits
TUI framework	bubbletea v2 beta	Event loop + Elm-style Model/Update/View; built-in support for subsystems (tabs, modals). v2 brings cursor control and saner Init semantics. 
github.com
github.com
Styling / Layout	lipgloss, bubbles	Themeable widgets (textinput, viewport, keymap) with almost zero boilerplate.
Quick one-off prompts	gum	Great for transient confirmations without writing extra Bubble Tea code. 
charm.sh
AI backend	go-openai (or any OpenAI-compatible endpoint)	Same HTTP shape works for OpenAI, Azure OpenAI, LocalAI, or your own MCP bridge.
Command sandbox	os/exec + creack/pty	Spawn real shells or single commands; capture stdout/stderr in real time.
Inspiration	Charm’s Mods CLI (AI chat that can pipe and exec)	Shows the pattern works; we’ll generalise it for interactive loops. 
charm.sh

2. High-Level Architecture
text
Copy
Edit
┌──────────────────────── Terminal UI (Bubble Tea) ────────────────────────┐
│                                                                          │
│   ┌────── Viewport (scrollback) ─────────────┐                            │
│   │                                          │                            │
│   │   … chat history, command output …       │                            │
│   └──────────────────────────────────────────┘                            │
│   ┌───────────── Input Line ──────────────┐ ┌────── Mode Indicator ─────┐ │
│   │  > user types here …                  │ │ [AI]  /  [Bash]           │ │
│   └────────────────────────────────────────┘ └───────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────┘
               ▲                              ▲
               │ key events / confirm         │ streaming output
               │                              ▼
         ┌───────────Core State/Model────────────┐
         │ currentMode (AI|Bash)                 │
         │ pendingCmd   *string                  │
         │ chatHistory  []Message                │
         │ …                                     │
         └────────────────────────────────────────┘
               ▲                     ▲
               │ approve/deny        │ spawn/pty
               │                     ▼
       ┌────────────── AI Worker ──────────────┐
       │ 1. send prompt                        │
       │ 2. parse tool_call JSON               │
       │ 3. push suggested cmd → Model         │
       └────────────────────────────────────────┘
3. Interaction Loop (“human-in-the-loop” agent)
User toggles mode – Ctrl-A ↔ Ctrl-B (AI / Bash).

AI mode

Text goes to LLM.

In your system prompt, instruct the model to always reply with either:

plain text or

{"tool":"bash","command":"…","reason":"…"}

When a JSON tool call arrives, the TUI:

Renders the command + reason in a modal.

Waits for [y]es / [n]o / [e]dit (use gum confirm for free candy-floss UX).

Approval path

Yes → spawn via exec.CommandContext, stream output back into viewport.

Edit → load command into input field (now in Bash mode) for tweaks.

No → send “Command rejected, try another approach” back to AI context.

Loop until the user switches back or exits.

Because Bubble Tea is message-driven, wiring this is mostly passing custom Msg types (AIResponseMsg, ExecOutputMsg, ApprovalMsg) through the Update function.

4. Key Components & Suggested Files
File	Purpose
main.go	CLI flags (e.g., --endpoint, --model), bootstrap Bubble Tea program.
model.go	Core state struct, Init, Update, View. Keep it tight.
ai.go	Goroutine that streams ChatCompletions; returns AIResponseMsg.
exec.go	Lightweight PTY wrapper; broadcasts ExecOutputMsg lines.
commands.go	Marshal/unmarshal the JSON tool schema; central place to whitelist allowed commands.
theme.go	All Lipgloss colours/styles so user can tweak a single file.
config.yml	Endpoint keys, default prompt templates, key-bindings.

Tip: write ai_test.go with a fake OpenAI server so your CI doesn’t burn tokens.

5. Security & Safety Nets
Whitelist or regex-filter commands before even showing them.

Dry-run flag that prints but never executes (great for demos).

Per-command timeout via context.WithTimeout.

Optionally launch everything inside a toolbox container or chroot for guard-rails.

6. Stretch Goals
Streaming: Pipe tokens from the AI to the viewport for that hacker-movie vibe.

Profiles: Quick-switch between local LLM, OpenAI, and a remote MCP chain.

Replay/Logging: Save every (prompt, response, command, output) tuple for later RAG or auditing.

Voice mode: Feed Mic → Whisper → same loop (Charm’s stack won’t fight you).

SSH-anywhere: Wrap with wish so the whole TUI runs when a user SSHs into a special port.

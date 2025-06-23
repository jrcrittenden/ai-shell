# Development Plan

## Completed
- [x] Reviewed repository structure and existing LLM backends.
- [x] Drafted plan to support Codex CLI and Claude code.
- [x] Added runtime backend switching with F1–F4 keys and navigation bar.

## Todo
- [ ] Add AI-assisted typeahead suggestions with a small distilled model.
- [ ] Improve markdown editing capabilities.

## Architectural Notes
The application uses a pluggable backend interface (`llm.Client`) which streams
`Chunk` structs to the TUI. New backends should conform to this interface.
Codex CLI can be invoked as an external command and wrapped in a client that
parses its output into `Chunk` messages. Claude would likely require an HTTP
client similar to the OpenAI implementation. The CLI clients run subprocesses
with `COLUMNS` and `LINES` environment variables set to fit inside the TUI
panes, passing conversation prompts via standard input and streaming standard
output lines back to the model.
Backends are now stored in a map and can be switched at runtime; a navigation bar
shows F-key assignments and a footer lists Ctrl-based shortcuts.

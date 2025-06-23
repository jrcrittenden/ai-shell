# Agent Guidelines

- Always format Go code with `gofmt -w` on changed files.
- Run `go test ./...` and ensure all tests pass after modifying code.
- When editing documentation only, tests are not required.
- Keep `plan.md` updated with checkmarks for completed items and record any
  architectural decisions that influence future work.
- Document new functions and exported types using Go doc comments.
- CLI-based backends (Codex and Claude) should execute the given binary using
  `exec.CommandContext` and set `COLUMNS` and `LINES` environment variables to
  small values so output fits within the TUI.
- Familiarity tips: use `grep -n` and `sed -n` for quick context, and keep tests
  fast by focusing on unit-level coverage.

_Last updated: 2025-06-22_

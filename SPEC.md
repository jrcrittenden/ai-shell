# AI‑Shell **Coding Agent** Specification  — v0.3 (2025‑06‑15)

> **Mission** — Guide an autonomous or semi‑autonomous LLM coder (Cline, Claude‑3, GPT‑Engineer, etc.) in extending and hardening the **ai‑shell** project while staying within predefined safety and quality constraints.
> 
> _This spec describes the engineering roadmap, coding standards, and project context. Implementation‑level tool‑call schemas and runtime interaction rules are documented elsewhere._

---

## 0  Project Overview  📜

`ai‑shell` is a **terminal‑native productivity tool** that fuses three interchangeable interaction modes:

1. **LLM Chat** — direct conversation with a centrally hosted LLM (e.g., OpenAI‑shape endpoint).
2. **Local Operator Chat** — conversation with a self‑hosted agent mesh exposing an SSE `/chat` endpoint.
3. **Standard Shell** — an interactive command‑line session tethered to the user’s preferred shell (bash, zsh, fish, PowerShell, etc.).

Core properties:

- **Human‑in‑the‑Loop Safety** — AI suggestions are surfaced for approval before any command runs.
- **Backend Agnostic** — plug‑in drivers for OpenAI‑compatible APIs and Local Operator out of the box.
- **Portable** — single static Go binary for macOS, Linux, Windows (WSL fallback).
- **Extensible** — upcoming plugin SPI for Git, Docker, AWS‑CLI, k8s, etc.

Current MVP has basic UI, generic shell execution, and backend abstraction but lacks streaming, approval UI, config loader, and plugin architecture.

---

## 1  Operating Environment

|Item|Spec|
|---|---|
|**Go**|1.22.x|
|**Module path**|`github.com/jrcrittenden/ai-shell`|
|**Key Deps**|Bubble Tea v0.25, Bubbles v0.15, Lipgloss v0.9, go‑openai v1.20|
|**OS targets**|Darwin/amd64+arm64, Linux/amd64+arm64, Windows/amd64 (WSL)|
|**CI**|GitHub Actions + Makefile|

---

## 2  Milestones & Acceptance Tests  🏁

|   |   |   |
|---|---|---|
|Phase|Description|Exit Criteria|
|**M0**|Compile‑clean scaffold|`go vet ./... && go test ./...` = 0|
|**M1**|**Viewport streaming** via `viewport.Append()`|Demo shows real‑time token flow|
|**M2**|**Approval modal** (dialog component)|Y/N/E keys trigger correct events|
|**M3**|**LLM streaming** (OpenAI chunks → viewport)|`TestStreamChunks` passes|
|**M4**|**Config loader** (`~/.config/ai-shell/config.yml` + env overrides)|`TestLoadConfig` passes|
|**M5**|**Unit coverage ≥ 80 %**|`go test -cover` ≥ 0.80|
|**M6**|**Plugin architecture draft**|SPI doc merged + dummy plugin loads|
|**M7**|**Dockerised release pipeline**|`make release` publishes multi‑arch image + GitHub tag|

_New milestones must be proposed and approved before work begins._

---

## 3  Canonical Repository Layout  🗂️

```
ai‑shell/
├── cmd/                 # future sub‑commands (daemon, plugin host, etc.)
├── internal/
│   ├── tui/             # Bubble Tea widgets (dialogs, spinners, keymaps)
│   ├── exec/            # PTY/sandbox helpers
│   ├── config/          # Config load + validation
│   └── plugins/         # SPI + plugin loader (planned M6)
├── llm/                 # backend abstraction layer
├── model.go             # core TUI model (update/view)
├── main.go              # CLI entry point
├── go.mod / go.sum
└── README.md
```

Structural changes require a migration note and README update.

---

## 4  Coding Standards  🛠️

1. **Formatting** — `go fmt ./...` & `goimports` must leave zero diff.
2. **Contexts** — All blocking ops accept `ctx context.Context`.
3. **Logging** — `log/slog` (`Info` default); avoid `fmt.Println`.
4. **Error wrapping** — `%w` via `fmt.Errorf` / `errors.Join`; no silent ignores.
5. **Tests** — Parallel by default; use `t.Setenv` and temp dirs.
6. **Commits** — Conventional Commits (`feat:`, `fix:`, `refactor:`, `docs:`).=

---

## 5  CI / CD Pipeline  ⚙️

- `make vet`         — `go vet ./...`
- `make test`        — `go test ./... -cover`
- `make release`     — builds `ai-shell` for darwin/linux/windows amd64+arm64, creates GitHub release, pushes Docker image.

Github Actions skeleton provided in repository (`.github/workflows/ci.yml`).

---

## 6  Security & Resource Rules  🛡️

- No outbound network calls in tests; mock via `httptest`.
- Max CPU per `go test` package 2 s (`-timeout=2s`).
- Shell commands executed through sandboxed PTY helper.
- Writes limited to repo path, `$HOME/.cache/ai-shell`, or `/tmp`.
- `go:embed` limited to ≤ 100 kB per file.

---

## 7  Glossary  📖

|   |   |
|---|---|
|Term|Meaning|
|_Local Operator_|Agent framework exposing `/chat` SSE endpoint.|
|_LLM Driver_|Code that implements `llm.Client` interface.|
|_SPI_|Service Provider Interface for plugin|

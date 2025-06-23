# Outstanding Tasks

1. **Implement Codex CLI Backend**
   - [x] Wrap the CLI in an `llm.Client` implementation.
   - [x] Stream stdout lines as `Chunk.Text`.

2. **Add Claude Backend**
   - [x] Implement CLI client using the `claude` binary.

3. **CLI Integration**
   - [x] Extend `main.go` to support `--backend codex` and `--backend claude` options.

4. **Testing**
   - [x] Add unit tests for the new clients using test binaries.

5. **Documentation**
   - [x] Update the README with setup instructions for the new backends.

6. **Runtime Backend Switching**
   - [x] Add navigation menu with F1–F4 keys to switch between OpenAI, LocalOp, Codex and Claude.
   - [x] Display Ctrl-key shortcuts at the bottom of the window.

7. **Future Enhancements**
   - [ ] Implement AI-assisted typeahead suggestions using a small distilled model.
   - [ ] Improve markdown editing workflow within the TUI.

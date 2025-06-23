package llm

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
)

// cliClient implements Client by invoking an external command and
// streaming its stdout.
type cliClient struct {
	path string
}

// newCLIClient returns a Client that runs the specified executable.
func newCLIClient(path string) Client {
	return &cliClient{path: path}
}

func (c *cliClient) Stream(ctx context.Context, hist []Message) <-chan Chunk {
	out := make(chan Chunk, 8)
	go func() {
		defer close(out)

		if c.path == "" {
			out <- Chunk{Err: io.EOF}
			return
		}

		cmd := exec.CommandContext(ctx, c.path)
		cmd.Env = append(os.Environ(), "COLUMNS=80", "LINES=20")

		stdin, err := cmd.StdinPipe()
		if err != nil {
			out <- Chunk{Err: err}
			return
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			out <- Chunk{Err: err}
			return
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			out <- Chunk{Err: err}
			return
		}

		var prompt string
		if len(hist) > 0 {
			prompt = hist[len(hist)-1].Content
		}
		io.WriteString(stdin, prompt)
		stdin.Close()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			out <- Chunk{Text: scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			out <- Chunk{Err: err}
		}
		cmd.Wait()
		out <- Chunk{Done: true}
	}()
	return out
}

// NewCodexCLI wraps the `codex` command line tool as a Client.
func NewCodexCLI(path string) Client { return newCLIClient(path) }

// NewClaudeCode wraps the `claude` command line tool as a Client.
func NewClaudeCode(path string) Client { return newCLIClient(path) }

package exec

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// Shell represents an interactive shell session
type Shell struct {
	cmd    *exec.Cmd
	ptmx   *os.File
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

// NewShell creates a new interactive shell session
func NewShell(ctx context.Context, shell string) (*Shell, error) {
	if shell == "" {
		shell = "bash"
	}

	cmd := exec.CommandContext(ctx, shell)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &Shell{
		cmd:  cmd,
		ptmx: ptmx,
		done: make(chan struct{}),
	}, nil
}

// Write writes data to the shell
func (s *Shell) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, io.EOF
	}
	return s.ptmx.Write(p)
}

// Read reads data from the shell
func (s *Shell) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, io.EOF
	}
	return s.ptmx.Read(p)
}

// Close closes the shell session
func (s *Shell) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.done)
	return s.ptmx.Close()
}

// Resize resizes the PTY
func (s *Shell) Resize(width, height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return io.EOF
	}
	return pty.Setsize(s.ptmx, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	})
}

// Wait waits for the shell to exit
func (s *Shell) Wait() error {
	return s.cmd.Wait()
}

// Done returns a channel that is closed when the shell is closed
func (s *Shell) Done() <-chan struct{} {
	return s.done
} 
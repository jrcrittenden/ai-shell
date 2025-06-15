package exec

import (
	"context"
	"os/exec"
)

// RunCommand executes a command using the system shell and returns combined stdout and stderr.
func RunCommand(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

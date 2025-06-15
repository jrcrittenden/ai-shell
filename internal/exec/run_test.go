package exec

import (
	"context"
	"strings"
	"testing"
)

func TestRunCommand(t *testing.T) {
	out, err := RunCommand(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("RunCommand error: %v", err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("unexpected output: %q", out)
	}
}

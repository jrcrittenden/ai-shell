package llm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIClient(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "echo.sh")
	script := "#!/bin/sh\nread line\necho $line processed"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	client := newCLIClient(scriptPath)
	hist := []Message{{Role: "user", Content: "test"}}
	chunks := collectChunks(client.Stream(context.Background(), hist))
	if len(chunks) == 0 {
		t.Fatalf("no chunks returned")
	}
	if chunks[0].Text != "test processed" {
		t.Fatalf("unexpected output: %q", chunks[0].Text)
	}
}

// collectChunks is copied from mock_test.go
func collectChunks(ch <-chan Chunk) []Chunk {
	var out []Chunk
	for c := range ch {
		out = append(out, c)
	}
	return out
}

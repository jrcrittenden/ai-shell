package llm

import (
	"context"
	"strings"
	"testing"
)

// gatherChunks reads from the channel until it closes and returns the collected chunks.
func gatherChunks(ch <-chan Chunk) []Chunk {
	var out []Chunk
	for c := range ch {
		out = append(out, c)
	}
	return out
}

func TestMockOpenAIFirstInteraction(t *testing.T) {
	c := NewMockOpenAI()
	hist := []Message{{Role: "user", Content: "hi"}}

	chunks := gatherChunks(c.Stream(context.Background(), hist))
	if len(chunks) == 0 {
		t.Fatalf("no chunks returned")
	}

	if !chunks[len(chunks)-1].Done {
		t.Errorf("expected last chunk to be marked Done")
	}

	var found bool
	for _, ch := range chunks {
		if ch.Text != "" && containsAvailable(ch.Text) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected available command listing in output")
	}
}

func containsAvailable(s string) bool {
	return strings.Contains(s, "Available commands")
}

func TestMockOpenAIToolCall(t *testing.T) {
	c := NewMockOpenAI()
	// first call to prime the client
	gatherChunks(c.Stream(context.Background(), []Message{{Role: "user", Content: "hi"}}))

	hist := []Message{{Role: "user", Content: "cmd"}}
	chunks := gatherChunks(c.Stream(context.Background(), hist))

	var tool *ToolCall
	for _, ch := range chunks {
		if ch.ToolCall != nil {
			tool = ch.ToolCall
			break
		}
	}
	if tool == nil {
		t.Fatalf("expected tool call chunk")
	}
	if tool.Command != "ls -alh" {
		t.Errorf("unexpected command %q", tool.Command)
	}
}

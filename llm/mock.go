package llm

import (
	"context"
	"fmt"
	"strings"
)

// mockOpenAI implements Client and returns deterministic responses for tests.
type mockOpenAI struct {
	firstInteraction bool
}

// NewMockOpenAI creates a new in-memory client that emulates an OpenAI chat
// endpoint. It is primarily intended for testing.
func NewMockOpenAI() Client {
	return &mockOpenAI{
		firstInteraction: true,
	}
}

// Stream sends a response for the supplied chat history. It interprets a small
// set of text commands and may emit ToolCall chunks.
func (m *mockOpenAI) Stream(ctx context.Context, hist []Message) <-chan Chunk {
	out := make(chan Chunk, 8)

	go func() {
		defer close(out)

		// Get the last message from history
		var lastMsg string
		if len(hist) > 0 {
			lastMsg = strings.ToLower(strings.TrimSpace(hist[len(hist)-1].Content))
			out <- Chunk{Text: fmt.Sprintf("[DEBUG] Processing: %q\n", lastMsg)}
		}

		// First interaction: show available commands
		if m.firstInteraction {
			m.firstInteraction = false
			out <- Chunk{Text: `Available commands:
- hello or greet: Get a friendly greeting
- cmd or command: List directory contents with ls -alh`}
			out <- Chunk{Done: true}
			return
		}

		// Handle commands
		switch lastMsg {
		case "hello", "greet":
			out <- Chunk{Text: "Hello!\n"}
		case "cmd", "command":
			out <- Chunk{Text: "[DEBUG] Command matched: cmd/command\n"}

			// Create and send the tool call
			toolCall := &ToolCall{
				Command: "ls -alh",
				Reason:  "Listing directory contents",
			}
			out <- Chunk{Text: fmt.Sprintf("[DEBUG] Tool call: %s (%s)\n", toolCall.Command, toolCall.Reason)}

			// Send the tool call in a separate chunk
			toolCallChunk := Chunk{ToolCall: toolCall}
			out <- toolCallChunk
		default:
			out <- Chunk{Text: fmt.Sprintf("I don't understand that command: %q. Try 'hello' or 'cmd'.\n", lastMsg)}
		}

		out <- Chunk{Done: true}
	}()

	return out
}

package llm

import "context"

type mockOpenAI struct {
	responses []string
	index    int
}

func NewMockOpenAI() Client {
	return &mockOpenAI{
		responses: []string{
			"Hello! I'm a mock AI assistant.",
			"Let me help you list the directory contents.",
		},
		index: 0,
	}
}

func (m *mockOpenAI) Stream(ctx context.Context, hist []Message) <-chan Chunk {
	out := make(chan Chunk, 8)

	go func() {
		defer close(out)

		// Get the next response
		response := m.responses[m.index]
		m.index = (m.index + 1) % len(m.responses)

		// If it's the second response, send a tool call
		if m.index == 0 {
			out <- Chunk{Text: response}
		} else {
			out <- Chunk{ToolCall: &ToolCall{
				Command: "ls -alh",
				Reason:  "Listing directory contents",
			}}
		}

		out <- Chunk{Done: true}
	}()

	return out
} 
package llm

import "context"

// ToolCall represents a command suggestion produced by the language model.
type ToolCall struct {
	// Command holds the command line to be executed.
	Command string
	// Reason explains why the model suggested this command.
	Reason string
}

// Chunk represents one streamed message from the model.
type Chunk struct {
	// Text contains plain text output from the model.
	Text string
	// ToolCall specifies an optional command the model wants to run.
	ToolCall *ToolCall
	// Done signals the end of the response stream.
	Done bool
	// Err contains any error produced while streaming.
	Err error
}

// Message represents a single entry in the chat history.
type Message struct {
	// Role of the sender (e.g. "user" or "assistant").
	Role string
	// Content holds the text of the message.
	Content string
}

// Client defines the common interface for language model backends.
//
// Stream should return a channel that produces chunks of the model's response
// for the provided chat history.
type Client interface {
	Stream(ctx context.Context, history []Message) <-chan Chunk
}

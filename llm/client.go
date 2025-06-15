package llm

import "context"

type ToolCall struct {
    Command string
    Reason  string
}

type Chunk struct {
    Text     string
    ToolCall *ToolCall
    Done     bool
    Err      error
}

type Message struct {
    Role    string
    Content string
}

type Client interface {
    Stream(ctx context.Context, history []Message) <-chan Chunk
}
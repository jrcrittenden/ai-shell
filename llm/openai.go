package llm

import (
	"context"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

// openAI implements Client using the OpenAI-compatible API.
type openAI struct {
	c     *openai.Client
	model string
}

// NewOpenAI returns a Client that connects to an OpenAI-compatible endpoint.
// baseURL may be left empty for the default OpenAI URL.
func NewOpenAI(apiKey, baseURL, model string) Client {
	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL // works for LocalAI, vLLM, Groq, etc.
	}
	return &openAI{
		c:     openai.NewClientWithConfig(cfg),
		model: model,
	}
}

func (o *openAI) Stream(ctx context.Context, hist []Message) <-chan Chunk {
	out := make(chan Chunk, 8)

	go func() {
		defer close(out)

		// Convert our slice to OpenAI’s type.
		msgs := make([]openai.ChatCompletionMessage, len(hist))
		for i, m := range hist {
			msgs[i] = openai.ChatCompletionMessage{Role: m.Role, Content: m.Content}
		}

		req := openai.ChatCompletionRequest{
			Model:  o.model,
			Stream: true,
			// Temperature / tools / etc. go here.
			Messages: msgs,
		}

		stream, err := o.c.CreateChatCompletionStream(ctx, req)
		if err != nil {
			out <- Chunk{Err: err}
			return
		}
		defer stream.Close()

		for {
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				out <- Chunk{Err: err}
				return
			}
			if len(resp.Choices) == 0 {
				continue
			}

			delta := resp.Choices[0].Delta
			if delta.Content != "" {
				out <- Chunk{Text: delta.Content}
			}
			if len(delta.ToolCalls) > 0 {
				tc := delta.ToolCalls[0]
				out <- Chunk{ToolCall: &ToolCall{
					Command: string(tc.Function.Arguments),
					Reason:  "tool call via OpenAI",
				}}
			}
		}
		out <- Chunk{Done: true}
	}()

	return out
}

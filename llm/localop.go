package llm

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "net/http"
)

type localOp struct {
    url   string
    agent string
}

func NewLocalOperator(endpoint, agent string) Client {
    return &localOp{url: endpoint, agent: agent}
}

func (l *localOp) Stream(ctx context.Context, hist []Message) <-chan Chunk {
    out := make(chan Chunk, 8)
    go func() {
        defer close(out)

        reqBody, _ := json.Marshal(map[string]interface{}{
            "agent":   l.agent,
            "stream":  true,
            "message": hist[len(hist)-1].Content,
        })
        req, _ := http.NewRequestWithContext(ctx, "POST", l.url, bytes.NewReader(reqBody))
        req.Header.Set("Content-Type", "application/json")

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            out <- Chunk{Err: err}
            return
        }
        defer resp.Body.Close()

        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            if bytes.HasPrefix([]byte(line), []byte("data: ")) {
                data := bytes.TrimPrefix([]byte(line), []byte("data: "))
                var payload map[string]string
                if err := json.Unmarshal(data, &payload); err == nil {
                    if cmd, ok := payload["command"]; ok {
                        out <- Chunk{ToolCall: &ToolCall{Command: cmd, Reason: payload["reason"]}}
                    } else if text, ok := payload["content"]; ok {
                        out <- Chunk{Text: text}
                    }
                }
            }
        }
        if scanner.Err() != nil {
            out <- Chunk{Err: scanner.Err()}
        }
        out <- Chunk{Done: true}
    }()
    return out
}
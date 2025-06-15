package main

import (
    //"context"
    "flag"
    "log"
    "os"
    "time"

    "github.com/jrcrittenden/ai-shell/llm"
    tea "github.com/charmbracelet/bubbletea"
)

var (
    backend = flag.String("backend", "openai", "openai | localop")
    model   = flag.String("model", "gpt-4o", "model or agent name")
    url     = flag.String("url", "", "override base URL")
    apiKey     = flag.String("key", os.Getenv("OPENAI_API_KEY"), "OpenAI key (if needed)")
)

func makeClient() llm.Client {
    switch *backend {
    case "localop":
        if *url == "" {
            *url = "http://localhost:8080/chat"
        }
        return llm.NewLocalOperator(*url, *model)
    default:
        return llm.NewOpenAI(*apiKey, *url, *model)
    }
}

func main() {
    flag.Parse()
    client := makeClient()

    m := NewModel(client)
    p := tea.NewProgram(m)
    m.program = p
    if _, err := p.Run(); err != nil {
        log.Fatalf("error running program: %v", err)
    }
    // give some time for shutdown cleanup
    time.Sleep(200 * time.Millisecond)
}

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/jrcrittenden/ai-shell/llm"
)

var (
	backend = flag.String("backend", "openai", "Backend to use (openai, localop, mock)")
	apiKey  = flag.String("api-key", "", "OpenAI API key")
	url     = flag.String("url", "", "URL for local operator")
	model   = flag.String("model", "gpt-4", "Model to use")
)

func main() {
	flag.Parse()

	// Create the client
	client := makeClient()

	// Create the model
	m := NewModel(client)

	// Create the program
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func makeClient() llm.Client {
	switch *backend {
	case "localop":
		if *url == "" {
			*url = "http://localhost:8080/chat"
		}
		return llm.NewLocalOperator(*url, *model)
	case "mock":
		return llm.NewMockOpenAI()
	default:
		return llm.NewOpenAI(*apiKey, *url, *model)
	}
}

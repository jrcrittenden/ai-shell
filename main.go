package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/jrcrittenden/ai-shell/llm"
)

var (
	backend = flag.String("backend", "openai", "Backend to use (openai, localop, codex, claude, mock)")
	apiKey  = flag.String("api-key", "", "OpenAI API key")
	url     = flag.String("url", "", "URL for local operator")
	model   = flag.String("model", "gpt-4", "Model to use")
)

func main() {
	flag.Parse()

	// Create the clients for runtime switching
	clients := makeClients()

	// Create the model with the requested backend active
	m := NewModel(clients, *backend)

	// Create the program
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func makeClients() map[string]llm.Client {
	clients := map[string]llm.Client{
		"openai":  llm.NewOpenAI(*apiKey, *url, *model),
		"localop": llm.NewLocalOperator(defaultURL(), *model),
		"codex":   llm.NewCodexCLI("codex"),
		"claude":  llm.NewClaudeCode("claude"),
		"mock":    llm.NewMockOpenAI(),
	}
	return clients
}

func defaultURL() string {
	if *url == "" {
		return "http://localhost:8080/chat"
	}
	return *url
}

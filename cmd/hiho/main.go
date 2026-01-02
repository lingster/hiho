package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"hiho/internal/tmux"
	"hiho/internal/ui"
)

func main() {
	manager := tmux.NewManager()
	model := ui.NewModel(manager)

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		log.Fatalf("failed to start TUI: %v", err)
	}
}

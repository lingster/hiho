package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"hiho/internal/config"
	"hiho/internal/tmux"
	"hiho/internal/ui"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Create tmux manager
	manager := tmux.NewManager()

	// Create UI model with config
	model := ui.NewModel(manager, cfg)

	// Create program with alt screen and mouse support
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("failed to start TUI: %v", err)
	}
}

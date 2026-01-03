module hiho

go 1.24.3

replace github.com/charmbracelet/bubbletea => ./third_party/github.com/charmbracelet/bubbletea

replace github.com/charmbracelet/bubbles => ./third_party/github.com/charmbracelet/bubbles

replace github.com/charmbracelet/lipgloss => ./third_party/github.com/charmbracelet/lipgloss

require (
	github.com/charmbracelet/bubbles v0.0.0
	github.com/charmbracelet/bubbletea v0.0.0
	github.com/charmbracelet/lipgloss v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/term v0.27.0 // indirect
)

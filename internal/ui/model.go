package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hiho/internal/tmux"
)

type viewMode int

const (
	conversationView viewMode = iota
	sessionView
)

// Message represents a single conversation item.
type Message struct {
	Role    string
	Content string
}

// Model drives the TUI.
type Model struct {
	manager        tmux.SessionManager
	messages       []Message
	currentSession string
	sessionLog     string
	mode           viewMode
	input          textinput.Model
	viewport       viewport.Model
	width          int
	height         int
}

// NewModel constructs the UI model.
func NewModel(manager tmux.SessionManager) Model {
	input := textinput.New()
	input.Placeholder = "/new <cmd> or type a note"
	input.Prompt = "> "
	input.Focus()

	vp := viewport.New(0, 0)
	return Model{
		manager:  manager,
		mode:     conversationView,
		input:    input,
		viewport: vp,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if value != "" {
				if err := m.handleSubmit(value); err != nil {
					m.appendMessage("error", err.Error())
				}
				m.input.Reset()
				m.refreshViewport()
			}
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 5
		m.refreshViewport()
	}

	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	title := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("Session: %s • View: %s", m.displaySession(), m.modeLabel()),
	)
	body := m.viewport.View()
	prompt := lipgloss.NewStyle().
		MarginTop(1).
		Render(m.input.View())

	return lipgloss.JoinVertical(lipgloss.Left, title, body, prompt)
}

func (m *Model) handleSubmit(input string) error {
	if strings.HasPrefix(input, "/") {
		if err := m.handleCommand(input); err != nil {
			return err
		}
	} else {
		m.appendMessage("user", input)
	}
	return nil
}

func (m *Model) handleCommand(input string) error {
	parts := strings.SplitN(strings.TrimPrefix(input, "/"), " ", 2)
	command := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	switch command {
	case "new":
		if arg == "" {
			return fmt.Errorf("usage: /new <command>")
		}
		session, err := m.manager.NewSession(arg)
		if err != nil {
			return err
		}
		m.currentSession = session.Name
		return m.captureCurrentSession()
	case "next":
		session, err := m.manager.Next(m.currentSession)
		if err != nil {
			return err
		}
		m.currentSession = session.Name
		return m.captureCurrentSession()
	case "prev":
		session, err := m.manager.Prev(m.currentSession)
		if err != nil {
			return err
		}
		m.currentSession = session.Name
		return m.captureCurrentSession()
	case "switch":
		if arg == "" {
			return fmt.Errorf("usage: /switch <session>")
		}
		session, err := m.manager.Switch(arg)
		if err != nil {
			return err
		}
		m.currentSession = session.Name
		return m.captureCurrentSession()
	case "sessions":
		sessions, err := m.manager.List()
		if err != nil {
			return err
		}
		names := make([]string, 0, len(sessions))
		for _, session := range sessions {
			names = append(names, session.Name)
		}
		m.appendMessage("sessions", strings.Join(names, ", "))
	case "view":
		switch arg {
		case "session":
			m.mode = sessionView
		default:
			m.mode = conversationView
		}
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
	return nil
}

func (m *Model) captureCurrentSession() error {
	if m.currentSession == "" {
		return tmux.ErrSessionNotFound
	}
	output, err := m.manager.Capture(m.currentSession)
	if err != nil {
		return err
	}
	m.sessionLog = output
	m.appendMessage(m.currentSession, output)
	m.refreshViewport()
	return nil
}

func (m *Model) appendMessage(role, content string) {
	m.messages = append(m.messages, Message{Role: role, Content: content})
	m.refreshViewport()
}

func (m *Model) refreshViewport() {
	content := m.renderBody()
	m.viewport.SetContent(content)
}

func (m *Model) renderBody() string {
	if m.mode == sessionView {
		if m.currentSession == "" {
			return "No active session."
		}
		header := lipgloss.NewStyle().Bold(true).Render(m.currentSession)
		return lipgloss.JoinVertical(lipgloss.Left, header, strings.TrimSpace(m.sessionLog))
	}

	var builder strings.Builder
	for _, message := range m.messages {
		role := lipgloss.NewStyle().Bold(true).Render(message.Role + ":")
		builder.WriteString(role)
		builder.WriteString(" ")
		builder.WriteString(strings.TrimSpace(message.Content))
		builder.WriteString("\n")
	}
	return strings.TrimSuffix(builder.String(), "\n")
}

func (m Model) displaySession() string {
	if m.currentSession == "" {
		return "none"
	}
	return m.currentSession
}

func (m Model) modeLabel() string {
	if m.mode == sessionView {
		return "session"
	}
	return "conversation"
}

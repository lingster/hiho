package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hiho/internal/config"
	"hiho/internal/tmux"
)

type tabType int

const (
	tabConversation tabType = iota
	tabTmux
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusMain
	focusInput
)

// Message represents a single conversation item.
type Message struct {
	Role    string
	Content string
}

// Model drives the TUI.
type Model struct {
	manager        tmux.SessionManager
	config         config.Config
	messages       []Message
	currentSession string
	sessionLog     string
	activeTab      tabType
	focus          focusArea
	input          textinput.Model
	viewport       viewport.Model
	width          int
	height         int
	sessions       []tmux.Session // cached session list
	sessionIndex   int            // selected session in sidebar
}

// NewModel constructs the UI model.
func NewModel(manager tmux.SessionManager, cfg config.Config) Model {
	input := textinput.New()
	input.Placeholder = "/new <cmd> or type a note"
	input.Prompt = "> "
	input.Focus()

	vp := viewport.New(0, 0)
	return Model{
		manager:   manager,
		config:    cfg,
		activeTab: tabConversation,
		focus:     focusInput,
		input:     input,
		viewport:  vp,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// sidebarWidth calculates the sidebar width (1/3 of total).
func (m Model) sidebarWidth() int {
	return m.width / 3
}

// mainWidth calculates the main panel width (2/3 of total).
func (m Model) mainWidth() int {
	return m.width - m.sidebarWidth()
}

// bodyHeight calculates the height for sidebar and main panels.
func (m Model) bodyHeight() int {
	return m.height - 4 // Reserve 4 rows for input panel
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Check configurable keybindings first
		switch key {
		case m.config.KeyBindings.Quit:
			return m, tea.Quit
		case m.config.KeyBindings.ToggleTab:
			m.toggleTab()
			m.refreshViewport()
			return m, nil
		case m.config.KeyBindings.NextSession:
			if err := m.navigateSession(1); err != nil {
				m.appendMessage("error", err.Error())
			}
			return m, nil
		case m.config.KeyBindings.PrevSession:
			if err := m.navigateSession(-1); err != nil {
				m.appendMessage("error", err.Error())
			}
			return m, nil
		case m.config.KeyBindings.CycleWindows:
			// Cycle focus between sidebar, main, input
			switch m.focus {
			case focusSidebar:
				m.focus = focusMain
			case focusMain:
				m.focus = focusInput
				m.input.Focus()
			case focusInput:
				m.focus = focusSidebar
				m.input.Blur()
			}
			return m, nil
		}

		// Handle focus-specific keys
		switch m.focus {
		case focusSidebar:
			switch key {
			case m.config.KeyBindings.SessionUp, "up", "k":
				m.selectPrevSession()
				return m, nil
			case m.config.KeyBindings.SessionDown, "down", "j":
				m.selectNextSession()
				return m, nil
			case "enter":
				m.activateSelectedSession()
				return m, nil
			}
		case focusInput:
			switch key {
			case "enter":
				value := strings.TrimSpace(m.input.Value())
				if value != "" {
					if err := m.handleSubmit(value); err != nil {
						m.appendMessage("error", err.Error())
					}
					m.input.Reset()
					m.refreshViewport()
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}

		// Legacy key handling for backward compatibility
		switch key {
		case "alt+h":
			if err := m.navigateSession(-1); err != nil {
				m.appendMessage("error", err.Error())
			}
		case "alt+l":
			if err := m.navigateSession(1); err != nil {
				m.appendMessage("error", err.Error())
			}
		case "alt+j":
			if err := m.navigateSession(-1); err != nil {
				m.appendMessage("error", err.Error())
			}
		case "alt+k":
			if err := m.navigateSession(1); err != nil {
				m.appendMessage("error", err.Error())
			}
		}

	case tea.MouseMsg:
		m.handleMouse(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport dimensions for the main panel
		m.viewport.Width = m.mainWidth() - 4  // Account for borders
		m.viewport.Height = m.bodyHeight() - 4 // Account for borders and tab bar
		m.refreshSessions()
		m.refreshViewport()
	}

	return m, nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) {
	if msg.Type != tea.MouseLeft {
		return
	}

	sidebarW := m.sidebarWidth()
	bodyH := m.bodyHeight()

	// Click in sidebar?
	if msg.X < sidebarW && msg.Y > 0 && msg.Y < bodyH {
		// Header row is at Y=1 (inside border), sessions start at Y=2
		sessionIdx := msg.Y - 2
		if sessionIdx >= 0 && sessionIdx < len(m.sessions) {
			m.sessionIndex = sessionIdx
			m.activateSelectedSession()
			m.focus = focusSidebar
		}
		return
	}

	// Click in main panel tab bar?
	if msg.X >= sidebarW && msg.Y >= 1 && msg.Y <= 1 {
		// Rough detection: first half = Conversation, second half = Tmux
		tabX := msg.X - sidebarW
		mainW := m.mainWidth()
		if tabX < mainW/2 {
			m.activeTab = tabConversation
		} else {
			m.activeTab = tabTmux
		}
		m.refreshViewport()
		return
	}

	// Click in input area?
	if msg.Y >= bodyH {
		m.focus = focusInput
		m.input.Focus()
		return
	}

	// Click in main content area
	if msg.X >= sidebarW && msg.Y > 1 && msg.Y < bodyH {
		m.focus = focusMain
		m.input.Blur()
	}
}

func (m *Model) refreshSessions() {
	sessions, err := m.manager.ListHiho()
	if err == nil {
		m.sessions = sessions
	}
}

func (m *Model) selectPrevSession() {
	if len(m.sessions) == 0 {
		m.refreshSessions()
	}
	if len(m.sessions) > 0 && m.sessionIndex > 0 {
		m.sessionIndex--
	}
}

func (m *Model) selectNextSession() {
	if len(m.sessions) == 0 {
		m.refreshSessions()
	}
	if len(m.sessions) > 0 && m.sessionIndex < len(m.sessions)-1 {
		m.sessionIndex++
	}
}

func (m *Model) activateSelectedSession() {
	if len(m.sessions) == 0 {
		m.refreshSessions()
	}
	if m.sessionIndex >= 0 && m.sessionIndex < len(m.sessions) {
		m.currentSession = m.sessions[m.sessionIndex].Name
		m.captureCurrentSession()
		m.activeTab = tabTmux
		m.refreshViewport()
	}
}

func (m *Model) toggleTab() {
	if m.activeTab == tabConversation {
		m.activeTab = tabTmux
	} else {
		m.activeTab = tabConversation
	}
}

func (m *Model) navigateSession(delta int) error {
	m.refreshSessions()
	if len(m.sessions) == 0 {
		return fmt.Errorf("no hiho sessions available")
	}

	if m.currentSession == "" {
		m.sessionIndex = 0
		m.currentSession = m.sessions[0].Name
		return m.captureCurrentSession()
	}

	// Find current session index
	for i, s := range m.sessions {
		if s.Name == m.currentSession {
			m.sessionIndex = i
			break
		}
	}

	// Navigate
	newIndex := m.sessionIndex + delta
	if newIndex < 0 {
		newIndex = len(m.sessions) - 1
	} else if newIndex >= len(m.sessions) {
		newIndex = 0
	}

	m.sessionIndex = newIndex
	m.currentSession = m.sessions[newIndex].Name
	return m.captureCurrentSession()
}

// View renders the TUI with 3-panel layout.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Render the three panels
	sidebar := m.renderSidebar()
	mainPanel := m.renderMainPanel()

	// Join sidebar and main panel horizontally
	topSection := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainPanel)

	// Render input panel
	inputPanel := m.renderInputPanel()

	return lipgloss.JoinVertical(lipgloss.Left, topSection, inputPanel)
}

func (m Model) renderSidebar() string {
	w := m.sidebarWidth() - 2 // Account for border
	h := m.bodyHeight() - 2   // Account for border

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true)
	content.WriteString(titleStyle.Render("Sessions"))
	content.WriteString("\n")

	// Session list
	if len(m.sessions) == 0 {
		content.WriteString("No sessions\n")
		content.WriteString("Use /new <cmd>")
	} else {
		for i, session := range m.sessions {
			var line string
			isSelected := i == m.sessionIndex
			isCurrent := session.Name == m.currentSession

			prefix := "  "
			if isCurrent {
				prefix = "> "
			}

			name := session.Name
			// Truncate if too long
			maxLen := w - 4
			if len(name) > maxLen && maxLen > 3 {
				name = name[:maxLen-3] + "..."
			}

			line = prefix + name

			if isSelected && m.focus == focusSidebar {
				// Highlighted with inverted colors
				line = lipgloss.NewStyle().Reverse(true).Render(line)
			} else if isCurrent {
				// Current session in bold
				line = lipgloss.NewStyle().Bold(true).Render(line)
			}

			content.WriteString(line)
			if i < len(m.sessions)-1 {
				content.WriteString("\n")
			}
		}
	}

	// Apply border and fixed dimensions
	style := lipgloss.NewStyle().
		Border(true).
		Width(w).
		Height(h)

	return style.Render(content.String())
}

func (m Model) renderMainPanel() string {
	w := m.mainWidth() - 2   // Account for border
	h := m.bodyHeight() - 2  // Account for border

	var content strings.Builder

	// Tab bar
	tabBar := m.renderTabBar()
	content.WriteString(tabBar)
	content.WriteString("\n")

	// Main content (viewport)
	body := m.viewport.View()
	content.WriteString(body)

	// Apply border and fixed dimensions
	style := lipgloss.NewStyle().
		Border(true).
		Width(w).
		Height(h)

	return style.Render(content.String())
}

func (m Model) renderTabBar() string {
	activeStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1)

	var convTab, tmuxTab string
	if m.activeTab == tabConversation {
		convTab = activeStyle.Render("Conversation")
		tmuxTab = inactiveStyle.Render("Tmux Window")
	} else {
		convTab = inactiveStyle.Render("Conversation")
		tmuxTab = activeStyle.Render("Tmux Window")
	}

	sessionInfo := ""
	if m.currentSession != "" {
		sessionInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(
			fmt.Sprintf(" • %s", m.currentSession),
		)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, convTab, " ", tmuxTab, sessionInfo)
}

func (m Model) renderInputPanel() string {
	w := m.width - 2 // Account for border

	var content strings.Builder

	// Input line
	content.WriteString(m.input.View())
	content.WriteString("\n")

	// Help line
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpText := fmt.Sprintf("Tab: toggle view • %s: cycle focus • ↑↓: navigate • Ctrl+C: quit",
		m.config.KeyBindings.CycleWindows)
	content.WriteString(helpStyle.Render(helpText))

	// Apply border
	style := lipgloss.NewStyle().
		Border(true).
		Width(w)

	return style.Render(content.String())
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
	case "help":
		m.appendMessage("info", commandHelp)
	case "new":
		if arg == "" {
			return fmt.Errorf("usage: /new <command>")
		}
		session, err := m.manager.NewSession(arg)
		if err != nil {
			return err
		}
		m.currentSession = session.Name
		m.activeTab = tabTmux
		m.refreshSessions()
		return m.captureCurrentSession()
	case "next":
		session, err := m.manager.Next(m.currentSession)
		if err != nil {
			return err
		}
		m.currentSession = session.Name
		m.refreshSessions()
		return m.captureCurrentSession()
	case "prev":
		session, err := m.manager.Prev(m.currentSession)
		if err != nil {
			return err
		}
		m.currentSession = session.Name
		m.refreshSessions()
		return m.captureCurrentSession()
	case "switch":
		if arg == "" {
			if m.activeTab == tabTmux {
				return m.navigateSession(1)
			}
			return fmt.Errorf("usage: /switch <session> (or use without arg in Tmux tab to cycle)")
		}
		session, err := m.manager.Switch(arg)
		if err != nil {
			return err
		}
		m.currentSession = session.Name
		m.refreshSessions()
		return m.captureCurrentSession()
	case "list":
		m.refreshSessions()
		if len(m.sessions) == 0 {
			m.appendMessage("info", "No hiho sessions found")
			return nil
		}
		names := make([]string, 0, len(m.sessions))
		for _, session := range m.sessions {
			names = append(names, session.Name)
		}
		m.appendMessage("sessions", strings.Join(names, "\n"))
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
	case "closeall":
		if err := m.manager.KillAllHiho(); err != nil {
			return err
		}
		if strings.HasPrefix(m.currentSession, "hiho-") {
			m.currentSession = ""
			m.sessionLog = ""
		}
		m.refreshSessions()
		m.appendMessage("info", "All hiho sessions closed")
	case "view":
		switch arg {
		case "session", "tmux":
			m.activeTab = tabTmux
		default:
			m.activeTab = tabConversation
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
	if m.activeTab == tabTmux {
		if m.currentSession == "" {
			return "No active session. Use /new <command> to create one."
		}
		header := lipgloss.NewStyle().Bold(true).Render(m.currentSession)
		return lipgloss.JoinVertical(lipgloss.Left, header, strings.TrimSpace(m.sessionLog))
	}

	// Conversation view
	if len(m.messages) == 0 {
		return "Welcome to hiho!\n" + commandHelp
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

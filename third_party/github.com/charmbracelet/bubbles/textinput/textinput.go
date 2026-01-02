package textinput

import tea "github.com/charmbracelet/bubbletea"

// Model holds text input state.
type Model struct {
	ValueStr    string
	Placeholder string
	Prompt      string
	focused     bool
}

// New constructs a Model.
func New() Model {
	return Model{Prompt: "> "}
}

// Focus enables editing.
func (m *Model) Focus() {
	m.focused = true
}

// Update applies key messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "backspace":
		if len(m.ValueStr) > 0 {
			m.ValueStr = m.ValueStr[:len(m.ValueStr)-1]
		}
	case "enter", "ctrl+c":
		// handled upstream
	default:
		if m.focused {
			m.ValueStr += key.String()
		}
	}
	return m, nil
}

// View renders the input.
func (m Model) View() string {
	value := m.ValueStr
	if value == "" && m.Placeholder != "" {
		value = m.Placeholder
	}
	return m.Prompt + value
}

// Value returns current text.
func (m Model) Value() string {
	return m.ValueStr
}

// Reset clears the input.
func (m *Model) Reset() {
	m.ValueStr = ""
}

// Blink is retained for compatibility.
var Blink tea.Cmd = nil

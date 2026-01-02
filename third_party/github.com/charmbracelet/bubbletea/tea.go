package bubbletea

import (
	"bufio"
	"fmt"
	"os"
)

// Msg represents a message handled by the model.
type Msg interface{}

// Cmd produces a Msg.
type Cmd func() Msg

// Model is the application state.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}

// Program executes the Bubble Tea loop.
type Program struct {
	model Model
}

// ProgramOption configures a Program.
type ProgramOption func(*Program)

// NewProgram builds a Program.
func NewProgram(model Model, _ ...ProgramOption) *Program {
	return &Program{model: model}
}

// WithAltScreen is retained for API compatibility.
func WithAltScreen() ProgramOption { return func(*Program) {} }

// Run executes a simplified event loop reading runes from stdin.
func (p *Program) Run() (Model, error) {
	m := p.model
	if cmd := m.Init(); cmd != nil {
		if msg := cmd(); msg != nil {
			m, _ = m.Update(msg)
		}
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\033[H\033[2J") // clear screen for a lightweight refresh
		fmt.Println(m.View())

		r, _, err := reader.ReadRune()
		if err != nil {
			return m, err
		}

		msg := KeyMsg{Type: string(r)}
		if r == '\n' {
			msg = KeyMsg{Type: "enter"}
		} else if r == 3 { // Ctrl+C
			msg = KeyMsg{Type: "ctrl+c"}
		}

		var cmd Cmd
		m, cmd = m.Update(msg)

		if cmd != nil {
			if _, ok := cmd().(quitMsg); ok {
				return m, nil
			}
		}
	}
}

// Batch runs commands and returns the last non-nil result.
func Batch(cmds ...Cmd) Cmd {
	return func() Msg {
		var msg Msg
		for _, cmd := range cmds {
			if cmd == nil {
				continue
			}
			msg = cmd()
		}
		return msg
	}
}

// KeyMsg represents a key press.
type KeyMsg struct {
	Type string
}

// String returns the key type.
func (k KeyMsg) String() string {
	return k.Type
}

// WindowSizeMsg reports the terminal dimensions.
type WindowSizeMsg struct {
	Width  int
	Height int
}

type quitMsg struct{}

// Quit stops the program.
var Quit Cmd = func() Msg { return quitMsg{} }

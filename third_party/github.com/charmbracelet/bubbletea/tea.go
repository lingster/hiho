package bubbletea

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
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
	model        Model
	altScreen    bool
	mouseEnabled bool
}

// ProgramOption configures a Program.
type ProgramOption func(*Program)

// NewProgram builds a Program.
func NewProgram(model Model, opts ...ProgramOption) *Program {
	p := &Program{model: model}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithAltScreen enables the alternate screen buffer.
func WithAltScreen() ProgramOption {
	return func(p *Program) { p.altScreen = true }
}

// WithMouseCellMotion enables mouse support with cell motion tracking.
func WithMouseCellMotion() ProgramOption {
	return func(p *Program) { p.mouseEnabled = true }
}

// Run executes the event loop with proper terminal handling.
func (p *Program) Run() (Model, error) {
	// Save terminal state and enter raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return p.model, fmt.Errorf("failed to enter raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Enter alternate screen if requested
	if p.altScreen {
		fmt.Print("\033[?1049h") // Enter alt screen
		defer fmt.Print("\033[?1049l") // Exit alt screen
	}

	// Enable mouse if requested
	if p.mouseEnabled {
		fmt.Print("\033[?1000h") // Enable mouse click tracking
		fmt.Print("\033[?1006h") // Enable SGR extended mouse mode
		defer fmt.Print("\033[?1000l")
		defer fmt.Print("\033[?1006l")
	}

	// Hide cursor during operation
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	m := p.model

	// Channel for messages (input, resize, etc.)
	msgCh := make(chan Msg, 10)
	done := make(chan struct{})
	defer close(done)

	// Handle window resize signals in separate goroutine
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGWINCH)
		defer signal.Stop(sigCh)
		for {
			select {
			case <-sigCh:
				w, h, _ := term.GetSize(int(os.Stdout.Fd()))
				select {
				case msgCh <- WindowSizeMsg{Width: w, Height: h}:
				case <-done:
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Read input in separate goroutine
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			msgs := parseInput(buf[:n])
			for _, msg := range msgs {
				select {
				case msgCh <- msg:
				case <-done:
					return
				}
			}
		}
	}()

	// Get initial window size
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	m, _ = m.Update(WindowSizeMsg{Width: width, Height: height})

	// Run init command
	if cmd := m.Init(); cmd != nil {
		if msg := cmd(); msg != nil {
			m, _ = m.Update(msg)
		}
	}

	// Main event loop
	for {
		// Clear screen and render
		fmt.Print("\033[H\033[2J")
		fmt.Print(m.View())

		// Wait for message
		msg := <-msgCh

		var cmd Cmd
		m, cmd = m.Update(msg)
		if cmd != nil {
			if _, ok := cmd().(quitMsg); ok {
				return m, nil
			}
		}
	}
}

// parseInput converts raw input bytes into messages.
func parseInput(buf []byte) []Msg {
	var msgs []Msg

	for i := 0; i < len(buf); {
		// Check for escape sequence
		if buf[i] == 0x1b {
			// SGR mouse sequence: ESC [ < Cb ; Cx ; Cy M/m
			if i+2 < len(buf) && buf[i+1] == '[' && buf[i+2] == '<' {
				msg, consumed := parseSGRMouse(buf[i:])
				if consumed > 0 {
					msgs = append(msgs, msg)
					i += consumed
					continue
				}
			}

			// CSI sequence: ESC [ ...
			if i+1 < len(buf) && buf[i+1] == '[' {
				msg, consumed := parseCSI(buf[i:])
				if consumed > 0 {
					msgs = append(msgs, msg)
					i += consumed
					continue
				}
			}

			// Alt+key: ESC followed by another character
			if i+1 < len(buf) && buf[i+1] != '[' && buf[i+1] != 'O' {
				key := string(buf[i+1])
				msgs = append(msgs, KeyMsg{Type: "alt+" + key})
				i += 2
				continue
			}

			// Standalone ESC
			msgs = append(msgs, KeyMsg{Type: "esc"})
			i++
			continue
		}

		// Control characters
		switch buf[i] {
		case 0x09:
			msgs = append(msgs, KeyMsg{Type: "tab"})
		case 0x0a, 0x0d:
			msgs = append(msgs, KeyMsg{Type: "enter"})
		case 0x01:
			msgs = append(msgs, KeyMsg{Type: "ctrl+a"})
		case 0x02:
			msgs = append(msgs, KeyMsg{Type: "ctrl+b"})
		case 0x03:
			msgs = append(msgs, KeyMsg{Type: "ctrl+c"})
		case 0x04:
			msgs = append(msgs, KeyMsg{Type: "ctrl+d"})
		case 0x05:
			msgs = append(msgs, KeyMsg{Type: "ctrl+e"})
		case 0x06:
			msgs = append(msgs, KeyMsg{Type: "ctrl+f"})
		case 0x0b:
			msgs = append(msgs, KeyMsg{Type: "ctrl+k"})
		case 0x0c:
			msgs = append(msgs, KeyMsg{Type: "ctrl+l"})
		case 0x0e:
			msgs = append(msgs, KeyMsg{Type: "ctrl+n"})
		case 0x0f:
			msgs = append(msgs, KeyMsg{Type: "ctrl+o"})
		case 0x10:
			msgs = append(msgs, KeyMsg{Type: "ctrl+p"})
		case 0x15:
			msgs = append(msgs, KeyMsg{Type: "ctrl+u"})
		case 0x17:
			msgs = append(msgs, KeyMsg{Type: "ctrl+w"})
		case 0x7f:
			msgs = append(msgs, KeyMsg{Type: "backspace"})
		default:
			// Regular character
			if buf[i] >= 0x20 && buf[i] < 0x7f {
				msgs = append(msgs, KeyMsg{Type: string(buf[i])})
			}
		}
		i++
	}

	return msgs
}

// parseCSI parses CSI escape sequences (ESC [ ...).
func parseCSI(buf []byte) (Msg, int) {
	if len(buf) < 3 || buf[0] != 0x1b || buf[1] != '[' {
		return nil, 0
	}

	// Arrow keys and other simple sequences
	switch buf[2] {
	case 'A':
		return KeyMsg{Type: "up"}, 3
	case 'B':
		return KeyMsg{Type: "down"}, 3
	case 'C':
		return KeyMsg{Type: "right"}, 3
	case 'D':
		return KeyMsg{Type: "left"}, 3
	case 'H':
		return KeyMsg{Type: "home"}, 3
	case 'F':
		return KeyMsg{Type: "end"}, 3
	case 'Z':
		return KeyMsg{Type: "shift+tab"}, 3
	}

	// Modified keys: ESC [ 1 ; mod X
	if len(buf) >= 6 && buf[2] == '1' && buf[3] == ';' {
		mod := buf[4]
		key := buf[5]
		prefix := ""
		switch mod {
		case '2':
			prefix = "shift+"
		case '3':
			prefix = "alt+"
		case '4':
			prefix = "shift+alt+"
		case '5':
			prefix = "ctrl+"
		case '6':
			prefix = "shift+ctrl+"
		case '7':
			prefix = "alt+ctrl+"
		case '8':
			prefix = "shift+alt+ctrl+"
		}
		switch key {
		case 'A':
			return KeyMsg{Type: prefix + "up"}, 6
		case 'B':
			return KeyMsg{Type: prefix + "down"}, 6
		case 'C':
			return KeyMsg{Type: prefix + "right"}, 6
		case 'D':
			return KeyMsg{Type: prefix + "left"}, 6
		case 'H':
			return KeyMsg{Type: prefix + "home"}, 6
		case 'F':
			return KeyMsg{Type: prefix + "end"}, 6
		}
	}

	// Page up/down, delete, insert: ESC [ N ~
	if len(buf) >= 4 && buf[3] == '~' {
		switch buf[2] {
		case '2':
			return KeyMsg{Type: "insert"}, 4
		case '3':
			return KeyMsg{Type: "delete"}, 4
		case '5':
			return KeyMsg{Type: "pgup"}, 4
		case '6':
			return KeyMsg{Type: "pgdown"}, 4
		}
	}

	return KeyMsg{Type: "unknown"}, 3
}

// parseSGRMouse parses SGR extended mouse sequences (ESC [ < Cb ; Cx ; Cy M/m).
func parseSGRMouse(buf []byte) (Msg, int) {
	if len(buf) < 4 || buf[0] != 0x1b || buf[1] != '[' || buf[2] != '<' {
		return nil, 0
	}

	// Parse: <Cb;Cx;CyM or <Cb;Cx;Cym
	var cb, cx, cy int
	var endChar byte
	i := 3

	// Parse button code
	for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
		cb = cb*10 + int(buf[i]-'0')
		i++
	}
	if i >= len(buf) || buf[i] != ';' {
		return nil, 0
	}
	i++

	// Parse X coordinate
	for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
		cx = cx*10 + int(buf[i]-'0')
		i++
	}
	if i >= len(buf) || buf[i] != ';' {
		return nil, 0
	}
	i++

	// Parse Y coordinate
	for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
		cy = cy*10 + int(buf[i]-'0')
		i++
	}
	if i >= len(buf) {
		return nil, 0
	}
	endChar = buf[i]
	i++

	if endChar != 'M' && endChar != 'm' {
		return nil, 0
	}

	// Determine event type
	eventType := MouseLeft
	if endChar == 'm' {
		eventType = MouseRelease
	} else {
		switch cb & 0x03 {
		case 0:
			eventType = MouseLeft
		case 1:
			eventType = MouseMiddle
		case 2:
			eventType = MouseRight
		case 3:
			eventType = MouseRelease
		}
		if cb&32 != 0 {
			eventType = MouseMotion
		}
		if cb&64 != 0 {
			if cb&1 == 0 {
				eventType = MouseWheelUp
			} else {
				eventType = MouseWheelDown
			}
		}
	}

	return MouseMsg{
		X:    cx - 1, // Convert to 0-based
		Y:    cy - 1,
		Type: eventType,
	}, i
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

// MouseEventType represents different mouse events.
type MouseEventType int

const (
	MouseLeft MouseEventType = iota
	MouseRight
	MouseMiddle
	MouseRelease
	MouseMotion
	MouseWheelUp
	MouseWheelDown
)

// MouseMsg represents a mouse event.
type MouseMsg struct {
	X    int
	Y    int
	Type MouseEventType
}

type quitMsg struct{}

// Quit stops the program.
var Quit Cmd = func() Msg { return quitMsg{} }

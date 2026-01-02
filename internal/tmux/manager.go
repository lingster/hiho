package tmux

import (
	"errors"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SessionManager describes tmux operations used by the TUI.
type SessionManager interface {
	NewSession(cmd string) (Session, error)
	Capture(name string) (string, error)
	List() ([]Session, error)
	Switch(name string) (Session, error)
	Next(current string) (Session, error)
	Prev(current string) (Session, error)
	Kill(name string) error
}

// Session represents a tmux session.
type Session struct {
	Name string
}

// Manager orchestrates tmux sessions.
type Manager struct {
	mu sync.Mutex
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ErrSessionNotFound indicates the requested session could not be located.
var ErrSessionNotFound = errors.New("session not found")

// NewManager constructs a Manager.
func NewManager() *Manager {
	return &Manager{}
}

// NewSession starts a detached tmux session and runs the provided command.
func (m *Manager) NewSession(cmd string) (Session, error) {
	name := m.uniqueName()

	if err := m.run("tmux", "new-session", "-d", "-s", name, "bash"); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	command := fmt.Sprintf("set -o pipefail; %s", cmd)
	if err := m.run("tmux", "send-keys", "-t", name, command, "C-m"); err != nil {
		return Session{}, fmt.Errorf("send command: %w", err)
	}

	return Session{Name: name}, nil
}

// Capture returns the visible pane output for a session.
func (m *Manager) Capture(name string) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-p", "-t", name, "-S", "-200").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("capture output: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// List returns all tmux sessions.
func (m *Manager) List() ([]Session, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#S").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var sessions []Session
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		sessions = append(sessions, Session{Name: strings.TrimSpace(line)})
	}
	return sessions, nil
}

// Switch updates the active session reference if it exists.
func (m *Manager) Switch(name string) (Session, error) {
	sessions, err := m.List()
	if err != nil {
		return Session{}, err
	}
	for _, session := range sessions {
		if session.Name == name {
			return session, nil
		}
	}
	return Session{}, ErrSessionNotFound
}

// Next cycles to the next session after the provided name.
func (m *Manager) Next(current string) (Session, error) {
	return m.selectRelative(current, 1)
}

// Prev cycles to the previous session before the provided name.
func (m *Manager) Prev(current string) (Session, error) {
	return m.selectRelative(current, -1)
}

// Kill terminates the named session.
func (m *Manager) Kill(name string) error {
	if err := m.run("tmux", "kill-session", "-t", name); err != nil {
		return fmt.Errorf("kill session: %w", err)
	}
	return nil
}

func (m *Manager) selectRelative(current string, delta int) (Session, error) {
	sessions, err := m.List()
	if err != nil {
		return Session{}, err
	}
	if len(sessions) == 0 {
		return Session{}, ErrSessionNotFound
	}
	index := -1
	for i, session := range sessions {
		if session.Name == current {
			index = i
			break
		}
	}
	if index == -1 {
		return Session{}, ErrSessionNotFound
	}
	next := (index + delta + len(sessions)) % len(sessions)
	return sessions[next], nil
}

func (m *Manager) run(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *Manager) uniqueName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf("hiho-%d-%04d", time.Now().UnixNano(), rand.Intn(10000))
}

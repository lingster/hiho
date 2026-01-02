package ui

import (
	"testing"

	"hiho/internal/tmux"
)

type stubManager struct {
	created      []string
	sessions     []string
	outputByName map[string]string
	currentIndex int
}

func (s *stubManager) NewSession(cmd string) (tmux.Session, error) {
	s.created = append(s.created, cmd)
	name := s.nextName()
	s.sessions = append(s.sessions, name)
	return tmux.Session{Name: name}, nil
}

func (s *stubManager) Capture(name string) (string, error) {
	return s.outputByName[name], nil
}

func (s *stubManager) List() ([]tmux.Session, error) {
	var result []tmux.Session
	for _, name := range s.sessions {
		result = append(result, tmux.Session{Name: name})
	}
	return result, nil
}

func (s *stubManager) Switch(name string) (tmux.Session, error) {
	for i, session := range s.sessions {
		if session == name {
			s.currentIndex = i
			return tmux.Session{Name: name}, nil
		}
	}
	return tmux.Session{}, tmux.ErrSessionNotFound
}

func (s *stubManager) Next(current string) (tmux.Session, error) {
	if len(s.sessions) == 0 {
		return tmux.Session{}, tmux.ErrSessionNotFound
	}
	s.currentIndex = (s.currentIndex + 1) % len(s.sessions)
	return tmux.Session{Name: s.sessions[s.currentIndex]}, nil
}

func (s *stubManager) Prev(current string) (tmux.Session, error) {
	if len(s.sessions) == 0 {
		return tmux.Session{}, tmux.ErrSessionNotFound
	}
	s.currentIndex = (s.currentIndex - 1 + len(s.sessions)) % len(s.sessions)
	return tmux.Session{Name: s.sessions[s.currentIndex]}, nil
}

func (s *stubManager) Kill(name string) error { return nil }

func (s *stubManager) nextName() string {
	return "session" + string('A'+rune(len(s.sessions)))
}

func TestNewCommandCreatesSessionAndCapturesOutput(t *testing.T) {
	manager := &stubManager{
		outputByName: map[string]string{
			"sessionA": "hello world\n",
		},
	}

	model := NewModel(manager)

	if err := model.handleSubmit("/new echo hello world"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	if len(manager.created) != 1 {
		t.Fatalf("expected one session creation, got %d", len(manager.created))
	}
	if manager.created[0] != "echo hello world" {
		t.Fatalf("unexpected command recorded: %q", manager.created[0])
	}
	if model.currentSession != "sessionA" {
		t.Fatalf("expected current session to be sessionA, got %q", model.currentSession)
	}
	if len(model.messages) == 0 {
		t.Fatalf("expected a message to be recorded")
	}
	if got := model.messages[len(model.messages)-1].Content; got != "hello world\n" {
		t.Fatalf("unexpected message content: %q", got)
	}
}

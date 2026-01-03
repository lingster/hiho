package ui

import (
	"strings"
	"testing"

	"hiho/internal/config"
	"hiho/internal/tmux"
)

// testConfig returns a default config for testing.
func testConfig() config.Config {
	return config.DefaultConfig()
}

type stubManager struct {
	created      []string
	sessions     []string
	outputByName map[string]string
	currentIndex int
	killed       []string
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

func (s *stubManager) ListHiho() ([]tmux.Session, error) {
	var result []tmux.Session
	for _, name := range s.sessions {
		if strings.HasPrefix(name, "hiho-") {
			result = append(result, tmux.Session{Name: name})
		}
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

func (s *stubManager) Kill(name string) error {
	s.killed = append(s.killed, name)
	// Remove from sessions
	for i, session := range s.sessions {
		if session == name {
			s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)
			break
		}
	}
	return nil
}

func (s *stubManager) KillAllHiho() error {
	var remaining []string
	for _, name := range s.sessions {
		if strings.HasPrefix(name, "hiho-") {
			s.killed = append(s.killed, name)
		} else {
			remaining = append(remaining, name)
		}
	}
	s.sessions = remaining
	return nil
}

func (s *stubManager) nextName() string {
	return "hiho-123-" + string('0'+rune(len(s.sessions)))
}

func TestNewCommandCreatesSessionAndCapturesOutput(t *testing.T) {
	manager := &stubManager{
		outputByName: map[string]string{
			"hiho-123-0": "hello world\n",
		},
	}

	model := NewModel(manager, testConfig())

	if err := model.handleSubmit("/new echo hello world"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	if len(manager.created) != 1 {
		t.Fatalf("expected one session creation, got %d", len(manager.created))
	}
	if manager.created[0] != "echo hello world" {
		t.Fatalf("unexpected command recorded: %q", manager.created[0])
	}
	if model.currentSession != "hiho-123-0" {
		t.Fatalf("expected current session to be hiho-123-0, got %q", model.currentSession)
	}
	if len(model.messages) == 0 {
		t.Fatalf("expected a message to be recorded")
	}
	if got := model.messages[len(model.messages)-1].Content; got != "hello world\n" {
		t.Fatalf("unexpected message content: %q", got)
	}
	// /new should switch to tmux tab
	if model.activeTab != tabTmux {
		t.Fatalf("expected activeTab to be tabTmux after /new")
	}
}

func TestListCommandShowsHihoSessions(t *testing.T) {
	manager := &stubManager{
		sessions: []string{"hiho-123-0", "hiho-123-1", "other-session"},
	}

	model := NewModel(manager, testConfig())

	if err := model.handleSubmit("/list"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	if len(model.messages) != 1 {
		t.Fatalf("expected one message, got %d", len(model.messages))
	}
	msg := model.messages[0]
	if msg.Role != "sessions" {
		t.Fatalf("expected role 'sessions', got %q", msg.Role)
	}
	// Should only list hiho sessions
	if !strings.Contains(msg.Content, "hiho-123-0") {
		t.Fatalf("expected hiho-123-0 in message, got %q", msg.Content)
	}
	if !strings.Contains(msg.Content, "hiho-123-1") {
		t.Fatalf("expected hiho-123-1 in message, got %q", msg.Content)
	}
	if strings.Contains(msg.Content, "other-session") {
		t.Fatalf("should not contain other-session, got %q", msg.Content)
	}
}

func TestListCommandShowsInfoWhenNoSessions(t *testing.T) {
	manager := &stubManager{
		sessions: []string{"other-session"},
	}

	model := NewModel(manager, testConfig())

	if err := model.handleSubmit("/list"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	if len(model.messages) != 1 {
		t.Fatalf("expected one message, got %d", len(model.messages))
	}
	if model.messages[0].Role != "info" {
		t.Fatalf("expected role 'info', got %q", model.messages[0].Role)
	}
}

func TestCloseAllKillsHihoSessions(t *testing.T) {
	manager := &stubManager{
		sessions: []string{"hiho-123-0", "hiho-123-1", "other-session"},
	}

	model := NewModel(manager, testConfig())
	model.currentSession = "hiho-123-0"

	if err := model.handleSubmit("/closeall"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	// Should have killed the hiho sessions
	if len(manager.killed) != 2 {
		t.Fatalf("expected 2 sessions killed, got %d", len(manager.killed))
	}
	// Current session should be cleared
	if model.currentSession != "" {
		t.Fatalf("expected currentSession to be empty, got %q", model.currentSession)
	}
	// Other session should remain
	if len(manager.sessions) != 1 || manager.sessions[0] != "other-session" {
		t.Fatalf("expected only other-session to remain, got %v", manager.sessions)
	}
}

func TestSwitchWithoutArgCyclesInTmuxTab(t *testing.T) {
	manager := &stubManager{
		sessions:     []string{"hiho-123-0", "hiho-123-1"},
		outputByName: map[string]string{"hiho-123-0": "out0", "hiho-123-1": "out1"},
	}

	model := NewModel(manager, testConfig())
	model.activeTab = tabTmux
	model.currentSession = "hiho-123-0"

	if err := model.handleSubmit("/switch"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	if model.currentSession != "hiho-123-1" {
		t.Fatalf("expected currentSession to be hiho-123-1, got %q", model.currentSession)
	}
}

func TestSwitchWithoutArgErrorsInConversationTab(t *testing.T) {
	manager := &stubManager{
		sessions: []string{"hiho-123-0", "hiho-123-1"},
	}

	model := NewModel(manager, testConfig())
	model.activeTab = tabConversation

	err := model.handleSubmit("/switch")
	if err == nil {
		t.Fatalf("expected error for /switch without arg in conversation tab")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected usage error, got %q", err.Error())
	}
}

func TestSwitchWithArgSwitchesToSession(t *testing.T) {
	manager := &stubManager{
		sessions:     []string{"hiho-123-0", "hiho-123-1"},
		outputByName: map[string]string{"hiho-123-0": "out0", "hiho-123-1": "out1"},
	}

	model := NewModel(manager, testConfig())

	if err := model.handleSubmit("/switch hiho-123-1"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	if model.currentSession != "hiho-123-1" {
		t.Fatalf("expected currentSession to be hiho-123-1, got %q", model.currentSession)
	}
}

func TestTabToggle(t *testing.T) {
	manager := &stubManager{}
	model := NewModel(manager, testConfig())

	if model.activeTab != tabConversation {
		t.Fatalf("expected initial tab to be conversation")
	}

	model.toggleTab()
	if model.activeTab != tabTmux {
		t.Fatalf("expected tab to be tmux after toggle")
	}

	model.toggleTab()
	if model.activeTab != tabConversation {
		t.Fatalf("expected tab to be conversation after second toggle")
	}
}

func TestNavigateSessionCyclesForward(t *testing.T) {
	manager := &stubManager{
		sessions:     []string{"hiho-123-0", "hiho-123-1", "hiho-123-2"},
		outputByName: map[string]string{"hiho-123-0": "out0", "hiho-123-1": "out1", "hiho-123-2": "out2"},
	}

	model := NewModel(manager, testConfig())
	model.currentSession = "hiho-123-0"

	if err := model.navigateSession(1); err != nil {
		t.Fatalf("navigateSession error: %v", err)
	}
	if model.currentSession != "hiho-123-1" {
		t.Fatalf("expected hiho-123-1, got %q", model.currentSession)
	}
}

func TestNavigateSessionCyclesBackward(t *testing.T) {
	manager := &stubManager{
		sessions:     []string{"hiho-123-0", "hiho-123-1", "hiho-123-2"},
		outputByName: map[string]string{"hiho-123-0": "out0", "hiho-123-1": "out1", "hiho-123-2": "out2"},
		currentIndex: 1,
	}

	model := NewModel(manager, testConfig())
	model.currentSession = "hiho-123-1"

	if err := model.navigateSession(-1); err != nil {
		t.Fatalf("navigateSession error: %v", err)
	}
	if model.currentSession != "hiho-123-0" {
		t.Fatalf("expected hiho-123-0, got %q", model.currentSession)
	}
}

func TestNavigateSessionSelectsFirstWhenNoCurrent(t *testing.T) {
	manager := &stubManager{
		sessions:     []string{"hiho-123-0", "hiho-123-1"},
		outputByName: map[string]string{"hiho-123-0": "out0", "hiho-123-1": "out1"},
	}

	model := NewModel(manager, testConfig())
	// No current session set

	if err := model.navigateSession(1); err != nil {
		t.Fatalf("navigateSession error: %v", err)
	}
	if model.currentSession != "hiho-123-0" {
		t.Fatalf("expected hiho-123-0, got %q", model.currentSession)
	}
}

func TestViewCommandSwitchesTabs(t *testing.T) {
	manager := &stubManager{}
	model := NewModel(manager, testConfig())

	if err := model.handleSubmit("/view tmux"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}
	if model.activeTab != tabTmux {
		t.Fatalf("expected tabTmux after /view tmux")
	}

	if err := model.handleSubmit("/view conversation"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}
	if model.activeTab != tabConversation {
		t.Fatalf("expected tabConversation after /view conversation")
	}

	if err := model.handleSubmit("/view session"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}
	if model.activeTab != tabTmux {
		t.Fatalf("expected tabTmux after /view session")
	}
}

func TestNextCommand(t *testing.T) {
	manager := &stubManager{
		sessions:     []string{"hiho-123-0", "hiho-123-1"},
		outputByName: map[string]string{"hiho-123-0": "out0", "hiho-123-1": "out1"},
	}

	model := NewModel(manager, testConfig())
	model.currentSession = "hiho-123-0"

	if err := model.handleSubmit("/next"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}
	if model.currentSession != "hiho-123-1" {
		t.Fatalf("expected hiho-123-1, got %q", model.currentSession)
	}
}

func TestPrevCommand(t *testing.T) {
	manager := &stubManager{
		sessions:     []string{"hiho-123-0", "hiho-123-1"},
		outputByName: map[string]string{"hiho-123-0": "out0", "hiho-123-1": "out1"},
		currentIndex: 1,
	}

	model := NewModel(manager, testConfig())
	model.currentSession = "hiho-123-1"

	if err := model.handleSubmit("/prev"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}
	if model.currentSession != "hiho-123-0" {
		t.Fatalf("expected hiho-123-0, got %q", model.currentSession)
	}
}

func TestSessionsCommandListsAll(t *testing.T) {
	manager := &stubManager{
		sessions: []string{"hiho-123-0", "other-session"},
	}

	model := NewModel(manager, testConfig())

	if err := model.handleSubmit("/sessions"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	if len(model.messages) != 1 {
		t.Fatalf("expected one message, got %d", len(model.messages))
	}
	// /sessions should list ALL sessions (unlike /list)
	if !strings.Contains(model.messages[0].Content, "other-session") {
		t.Fatalf("expected other-session in message, got %q", model.messages[0].Content)
	}
}

func TestHelpCommandShowsCommands(t *testing.T) {
	manager := &stubManager{}
	model := NewModel(manager, testConfig())

	if err := model.handleSubmit("/help"); err != nil {
		t.Fatalf("handleSubmit error: %v", err)
	}

	if len(model.messages) != 1 {
		t.Fatalf("expected one message, got %d", len(model.messages))
	}
	if model.messages[0].Role != "info" {
		t.Fatalf("expected role 'info', got %q", model.messages[0].Role)
	}
	if !strings.Contains(model.messages[0].Content, "/new <cmd>") {
		t.Fatalf("expected /new in help content, got %q", model.messages[0].Content)
	}
	if !strings.Contains(model.messages[0].Content, "/view conversation") {
		t.Fatalf("expected /view conversation in help content, got %q", model.messages[0].Content)
	}
}

func TestUnknownCommandReturnsError(t *testing.T) {
	manager := &stubManager{}
	model := NewModel(manager, testConfig())

	err := model.handleSubmit("/unknown")
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected 'unknown command' error, got %q", err.Error())
	}
}

func TestNewCommandWithoutArgReturnsError(t *testing.T) {
	manager := &stubManager{}
	model := NewModel(manager, testConfig())

	err := model.handleSubmit("/new")
	if err == nil {
		t.Fatalf("expected error for /new without arg")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected usage error, got %q", err.Error())
	}
}

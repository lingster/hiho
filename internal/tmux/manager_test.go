package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestNewSessionRunsCommand(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux binary not available")
	}

	manager := NewManager()

	session, err := manager.NewSession("echo hello world")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer manager.Kill(session.Name)

	output, err := manager.Capture(session.Name)
	if err != nil {
		t.Fatalf("failed to capture output: %v", err)
	}

	if !strings.Contains(output, "hello world") {
		t.Fatalf("expected output to contain greeting, got: %q", output)
	}
}

func TestSessionNamingFormat(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux binary not available")
	}

	manager := NewManager()
	pid := os.Getpid()

	session1, err := manager.NewSession("true")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer manager.Kill(session1.Name)

	session2, err := manager.NewSession("true")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer manager.Kill(session2.Name)

	// Check naming format: hiho-<pid>-<counter>
	expected1 := fmt.Sprintf("hiho-%d-0", pid)
	expected2 := fmt.Sprintf("hiho-%d-1", pid)

	if session1.Name != expected1 {
		t.Fatalf("expected session name %q, got %q", expected1, session1.Name)
	}
	if session2.Name != expected2 {
		t.Fatalf("expected session name %q, got %q", expected2, session2.Name)
	}
}

func TestListHihoFiltersCorrectly(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux binary not available")
	}

	manager := NewManager()

	// Create a hiho session
	session, err := manager.NewSession("true")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer manager.Kill(session.Name)

	// ListHiho should return our session
	hihoSessions, err := manager.ListHiho()
	if err != nil {
		t.Fatalf("ListHiho error: %v", err)
	}

	found := false
	for _, s := range hihoSessions {
		if s.Name == session.Name {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find %q in ListHiho results", session.Name)
	}

	// All returned sessions should have hiho- prefix
	for _, s := range hihoSessions {
		if !strings.HasPrefix(s.Name, "hiho-") {
			t.Fatalf("ListHiho returned non-hiho session: %q", s.Name)
		}
	}
}

func TestKillAllHiho(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux binary not available")
	}

	manager := NewManager()

	// Create two hiho sessions
	session1, err := manager.NewSession("true")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	session2, err := manager.NewSession("true")
	if err != nil {
		manager.Kill(session1.Name)
		t.Fatalf("failed to create session: %v", err)
	}

	// Kill all hiho sessions
	if err := manager.KillAllHiho(); err != nil {
		t.Fatalf("KillAllHiho error: %v", err)
	}

	// Verify sessions are gone
	hihoSessions, err := manager.ListHiho()
	if err != nil {
		t.Fatalf("ListHiho error: %v", err)
	}

	for _, s := range hihoSessions {
		if s.Name == session1.Name || s.Name == session2.Name {
			t.Fatalf("session %q should have been killed", s.Name)
		}
	}
}

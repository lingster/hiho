package tmux

import (
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

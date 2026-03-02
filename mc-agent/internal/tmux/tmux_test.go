package tmux_test

import (
	"strings"
	"testing"

	"mission-control/mc-agent/internal/tmux"
)

// These are integration tests — they require tmux installed and running.
// They create real tmux sessions and clean up after themselves.

const testPrefix = "mc-test-"

func cleanupSessions(t *testing.T, mgr *tmux.Manager) {
	t.Helper()
	sessions, _ := mgr.List(testPrefix)
	for _, s := range sessions {
		_ = mgr.Kill(s.Name)
	}
}

// Create a session and verify it shows up in List.
func TestCreate_sessionAppearsInList(t *testing.T) {
	mgr := tmux.NewManager()
	defer cleanupSessions(t, mgr)

	name := testPrefix + "create-1"
	err := mgr.Create(name, "")
	if err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}

	sessions, err := mgr.List(testPrefix)
	if err != nil {
		t.Fatalf("List(%q): %v", testPrefix, err)
	}

	found := false
	for _, s := range sessions {
		if s.Name == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("session %q not found in list: %v", name, sessions)
	}
}

// Create a session with an initial shell command.
func TestCreate_withCommand(t *testing.T) {
	mgr := tmux.NewManager()
	defer cleanupSessions(t, mgr)

	name := testPrefix + "cmd-1"
	err := mgr.Create(name, "echo hello")
	if err != nil {
		t.Fatalf("Create(%q, cmd): %v", name, err)
	}

	sessions, err := mgr.List(testPrefix)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := false
	for _, s := range sessions {
		if s.Name == name {
			found = true
		}
	}
	if !found {
		t.Errorf("session %q not found after Create with command", name)
	}
}

// Creating a session with an already-taken name should error.
func TestCreate_duplicateName_returnsError(t *testing.T) {
	mgr := tmux.NewManager()
	defer cleanupSessions(t, mgr)

	name := testPrefix + "dup-1"
	err := mgr.Create(name, "")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	err = mgr.Create(name, "")
	if err == nil {
		t.Error("expected error creating duplicate session, got nil")
	}
}

// Kill a session and verify it disappears from List.
func TestKill_sessionDisappearsFromList(t *testing.T) {
	mgr := tmux.NewManager()
	defer cleanupSessions(t, mgr)

	name := testPrefix + "kill-1"
	err := mgr.Create(name, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = mgr.Kill(name)
	if err != nil {
		t.Fatalf("Kill(%q): %v", name, err)
	}

	sessions, err := mgr.List(testPrefix)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, s := range sessions {
		if s.Name == name {
			t.Errorf("session %q still present after Kill", name)
		}
	}
}

// Killing a session that doesn't exist should error.
func TestKill_nonexistentSession_returnsError(t *testing.T) {
	mgr := tmux.NewManager()

	err := mgr.Kill(testPrefix + "nonexistent-xyz")
	if err == nil {
		t.Error("expected error killing nonexistent session, got nil")
	}
}

// Capture reads pane content; a session started with "echo X" should contain X.
func TestCapture_containsCommandOutput(t *testing.T) {
	mgr := tmux.NewManager()
	defer cleanupSessions(t, mgr)

	name := testPrefix + "capture-1"
	// Start a session that echoes a known string
	err := mgr.Create(name, "echo CAPTURE_TEST_MARKER")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	output, err := mgr.Capture(name)
	if err != nil {
		t.Fatalf("Capture(%q): %v", name, err)
	}

	if !strings.Contains(output, "CAPTURE_TEST_MARKER") {
		t.Errorf("Capture output missing expected marker.\nGot: %q", output)
	}
}

// Capturing a nonexistent session should error.
func TestCapture_nonexistentSession_returnsError(t *testing.T) {
	mgr := tmux.NewManager()

	_, err := mgr.Capture(testPrefix + "nonexistent-xyz")
	if err == nil {
		t.Error("expected error capturing nonexistent session, got nil")
	}
}

// SendKeys injects text into a pane; verify it appears in captured output.
func TestSendKeys_textAppearsInCapture(t *testing.T) {
	mgr := tmux.NewManager()
	defer cleanupSessions(t, mgr)

	name := testPrefix + "send-1"
	err := mgr.Create(name, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Send a command that writes a marker
	err = mgr.SendKeys(name, "echo SENDKEYS_MARKER")
	if err != nil {
		t.Fatalf("SendKeys: %v", err)
	}

	output, err := mgr.Capture(name)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	// The echo command itself should appear in the pane (typed text)
	if !strings.Contains(output, "SENDKEYS_MARKER") {
		t.Errorf("Capture after SendKeys missing marker.\nGot: %q", output)
	}
}

// Sending keys to a nonexistent session should error.
func TestSendKeys_nonexistentSession_returnsError(t *testing.T) {
	mgr := tmux.NewManager()

	err := mgr.SendKeys(testPrefix+"nonexistent-xyz", "echo hi")
	if err == nil {
		t.Error("expected error sending keys to nonexistent session, got nil")
	}
}

// List only returns sessions whose names match the given prefix.
func TestList_filtersByPrefix(t *testing.T) {
	mgr := tmux.NewManager()
	defer cleanupSessions(t, mgr)

	// Create two sessions with our prefix and verify List filters correctly
	name1 := testPrefix + "filter-a"
	name2 := testPrefix + "filter-b"

	if err := mgr.Create(name1, ""); err != nil {
		t.Fatalf("Create(%q): %v", name1, err)
	}
	if err := mgr.Create(name2, ""); err != nil {
		t.Fatalf("Create(%q): %v", name2, err)
	}

	sessions, err := mgr.List(testPrefix)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	names := make(map[string]bool)
	for _, s := range sessions {
		names[s.Name] = true
		if !strings.HasPrefix(s.Name, testPrefix) {
			t.Errorf("List returned session %q which doesn't match prefix %q", s.Name, testPrefix)
		}
	}

	if !names[name1] {
		t.Errorf("missing %q in filtered list", name1)
	}
	if !names[name2] {
		t.Errorf("missing %q in filtered list", name2)
	}
}

// List with empty prefix returns all sessions without erroring.
func TestList_emptyPrefix_returnsAll(t *testing.T) {
	mgr := tmux.NewManager()

	// Empty prefix should return all sessions (or empty slice), not error
	_, err := mgr.List("")
	if err != nil {
		t.Fatalf("List with empty prefix should not error: %v", err)
	}
}

// Session struct should have CreatedAt and Windows populated.
func TestList_sessionHasCreatedAtAndWindows(t *testing.T) {
	mgr := tmux.NewManager()
	defer cleanupSessions(t, mgr)

	name := testPrefix + "fields-1"
	if err := mgr.Create(name, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	sessions, err := mgr.List(testPrefix)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, s := range sessions {
		if s.Name == name {
			if s.CreatedAt.IsZero() {
				t.Error("session CreatedAt should not be zero")
			}
			if s.Windows < 1 {
				t.Error("session should have at least 1 window")
			}
			return
		}
	}
	t.Errorf("session %q not found", name)
}

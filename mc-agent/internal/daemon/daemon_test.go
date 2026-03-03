package daemon_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"mission-control/mc-agent/internal/daemon"
)

// fakeTmux is an in-memory fake of the tmux session manager.
type fakeTmux struct {
	sessions map[string]daemon.ManagedSession
}

func newFakeTmux() *fakeTmux {
	return &fakeTmux{sessions: make(map[string]daemon.ManagedSession)}
}

func (f *fakeTmux) Create(name string, command string) error {
	if _, exists := f.sessions[name]; exists {
		return fmt.Errorf("duplicate session: %s", name)
	}
	f.sessions[name] = daemon.ManagedSession{
		Name:      name,
		Windows:   1,
		CreatedAt: time.Now(),
	}
	return nil
}

func (f *fakeTmux) List(prefix string) ([]daemon.ManagedSession, error) {
	var result []daemon.ManagedSession
	for _, s := range f.sessions {
		if prefix == "" || strings.HasPrefix(s.Name, prefix) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (f *fakeTmux) Kill(name string) error {
	if _, exists := f.sessions[name]; !exists {
		return fmt.Errorf("session not found: %s", name)
	}
	delete(f.sessions, name)
	return nil
}

func (f *fakeTmux) Capture(name string) (string, error) {
	if _, exists := f.sessions[name]; !exists {
		return "", fmt.Errorf("session not found: %s", name)
	}
	return "", nil
}

func (f *fakeTmux) SendKeys(name string, keys string) error {
	if _, exists := f.sessions[name]; !exists {
		return fmt.Errorf("session not found: %s", name)
	}
	return nil
}

// CreateSession returns a tracked session with a ca-<project>-<id> tmux name.
func TestCreateSession_returnsSessionWithCorrectName(t *testing.T) {
	d := daemon.New(newFakeTmux())

	sess, err := d.CreateSession("mission-control")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if sess.Project != "mission-control" {
		t.Errorf("Project = %q, want %q", sess.Project, "mission-control")
	}
	if !strings.HasPrefix(sess.Name, "ca-mission-control-") {
		t.Errorf("Name = %q, want prefix %q", sess.Name, "ca-mission-control-")
	}
	if sess.ID == "" {
		t.Error("ID should not be empty")
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

// CreateSession creates a real tmux session via the manager.
func TestCreateSession_createsTmuxSession(t *testing.T) {
	fake := newFakeTmux()
	d := daemon.New(fake)

	sess, err := d.CreateSession("api-server")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if _, exists := fake.sessions[sess.Name]; !exists {
		t.Errorf("tmux session %q was not created", sess.Name)
	}
}

// Created sessions appear in ListSessions.
func TestCreateSession_appearsInListSessions(t *testing.T) {
	d := daemon.New(newFakeTmux())

	sess, err := d.CreateSession("frontend")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessions := d.ListSessions()
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions, want 1", len(sessions))
	}
	if sessions[0].ID != sess.ID {
		t.Errorf("listed session ID = %q, want %q", sessions[0].ID, sess.ID)
	}
}

// Multiple sessions are all tracked independently.
func TestCreateSession_multipleSessions(t *testing.T) {
	d := daemon.New(newFakeTmux())

	_, err := d.CreateSession("project-a")
	if err != nil {
		t.Fatalf("CreateSession(a): %v", err)
	}
	_, err = d.CreateSession("project-b")
	if err != nil {
		t.Fatalf("CreateSession(b): %v", err)
	}

	sessions := d.ListSessions()
	if len(sessions) != 2 {
		t.Fatalf("ListSessions returned %d sessions, want 2", len(sessions))
	}

	projects := map[string]bool{}
	for _, s := range sessions {
		projects[s.Project] = true
	}
	if !projects["project-a"] || !projects["project-b"] {
		t.Errorf("expected both projects, got %v", projects)
	}
}

// Each created session gets a unique ID.
func TestCreateSession_uniqueIDs(t *testing.T) {
	d := daemon.New(newFakeTmux())

	s1, _ := d.CreateSession("proj")
	s2, _ := d.CreateSession("proj")

	if s1.ID == s2.ID {
		t.Errorf("two sessions for same project got same ID: %q", s1.ID)
	}
	if s1.Name == s2.Name {
		t.Errorf("two sessions for same project got same tmux name: %q", s1.Name)
	}
}

// GetSession returns a session by ID.
func TestGetSession_findsExistingSession(t *testing.T) {
	d := daemon.New(newFakeTmux())

	created, _ := d.CreateSession("proj")

	sess, ok := d.GetSession(created.ID)
	if !ok {
		t.Fatal("GetSession returned ok=false for existing session")
	}
	if sess.ID != created.ID {
		t.Errorf("ID = %q, want %q", sess.ID, created.ID)
	}
}

// GetSession returns false for unknown IDs.
func TestGetSession_unknownID_returnsFalse(t *testing.T) {
	d := daemon.New(newFakeTmux())

	_, ok := d.GetSession("nonexistent")
	if ok {
		t.Error("GetSession returned ok=true for nonexistent session")
	}
}

// KillSession removes it from the registry and kills the tmux session.
func TestKillSession_removesFromRegistryAndTmux(t *testing.T) {
	fake := newFakeTmux()
	d := daemon.New(fake)

	sess, _ := d.CreateSession("proj")

	err := d.KillSession(sess.ID)
	if err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// Gone from daemon registry
	if _, ok := d.GetSession(sess.ID); ok {
		t.Error("session still in registry after KillSession")
	}

	// Gone from tmux
	if _, exists := fake.sessions[sess.Name]; exists {
		t.Error("tmux session still exists after KillSession")
	}
}

// KillSession with unknown ID returns error.
func TestKillSession_unknownID_returnsError(t *testing.T) {
	d := daemon.New(newFakeTmux())

	err := d.KillSession("nonexistent")
	if err == nil {
		t.Error("expected error killing unknown session, got nil")
	}
}

// Killing one session doesn't affect others.
func TestKillSession_doesNotAffectOtherSessions(t *testing.T) {
	d := daemon.New(newFakeTmux())

	s1, _ := d.CreateSession("proj-a")
	s2, _ := d.CreateSession("proj-b")

	if err := d.KillSession(s1.ID); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	sessions := d.ListSessions()
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions, want 1", len(sessions))
	}
	if sessions[0].ID != s2.ID {
		t.Errorf("remaining session ID = %q, want %q", sessions[0].ID, s2.ID)
	}
}

// Sync discovers existing ca-* tmux sessions not in the registry.
func TestSync_discoversExistingTmuxSessions(t *testing.T) {
	fake := newFakeTmux()
	// Pre-create a tmux session as if it existed before the daemon started
	fake.sessions["ca-legacy-proj-ab12"] = daemon.ManagedSession{
		Name:      "ca-legacy-proj-ab12",
		Windows:   1,
		CreatedAt: time.Now(),
	}

	d := daemon.New(fake)

	err := d.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	sessions := d.ListSessions()
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions after Sync, want 1", len(sessions))
	}
	if sessions[0].Name != "ca-legacy-proj-ab12" {
		t.Errorf("synced session Name = %q, want %q", sessions[0].Name, "ca-legacy-proj-ab12")
	}
}

// Sync doesn't duplicate sessions already in the registry.
func TestSync_doesNotDuplicateExistingSessions(t *testing.T) {
	fake := newFakeTmux()
	d := daemon.New(fake)

	_, err := d.CreateSession("proj")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	err = d.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	sessions := d.ListSessions()
	if len(sessions) != 1 {
		t.Errorf("ListSessions returned %d sessions after Sync, want 1", len(sessions))
	}
}

// Sync ignores non-cc tmux sessions (e.g. term-* or user sessions).
func TestSync_ignoresNonCcSessions(t *testing.T) {
	fake := newFakeTmux()
	fake.sessions["term-ab12"] = daemon.ManagedSession{
		Name:      "term-ab12",
		Windows:   1,
		CreatedAt: time.Now(),
	}
	fake.sessions["my-other-session"] = daemon.ManagedSession{
		Name:      "my-other-session",
		Windows:   1,
		CreatedAt: time.Now(),
	}

	d := daemon.New(fake)

	err := d.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	sessions := d.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("ListSessions returned %d sessions, want 0 (non-cc sessions should be ignored)", len(sessions))
	}
}

package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const sessionPrefix = "ca-"

// SessionManager is the interface the daemon uses to interact with tmux.
// Matches the tmux.Manager API so the real implementation satisfies it directly.
type SessionManager interface {
	Create(name string, command string) error
	List(prefix string) ([]ManagedSession, error)
	Kill(name string) error
	Capture(name string) (string, error)
	SendKeys(name string, keys string) error
}

// ManagedSession represents a tmux session as seen by the daemon.
type ManagedSession struct {
	Name      string
	Windows   int
	CreatedAt time.Time
}

// Session is a tracked Claude Code session in the daemon's registry.
type Session struct {
	ID        string
	Project   string
	Name      string // tmux session name (ca-<project>-<id>)
	CreatedAt time.Time
}

// Daemon is the core mc-agent process that manages sessions.
type Daemon struct {
	mgr      SessionManager
	sessions map[string]*Session // keyed by ID
}

// New creates a new Daemon backed by the given session manager.
func New(mgr SessionManager) *Daemon {
	return &Daemon{
		mgr:      mgr,
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new Claude Code tmux session for the given project.
func (d *Daemon) CreateSession(project string) (*Session, error) {
	id := shortID()
	name := sessionPrefix + project + "-" + id

	if err := d.mgr.Create(name, ""); err != nil {
		return nil, fmt.Errorf("create tmux session: %w", err)
	}

	sess := &Session{
		ID:        id,
		Project:   project,
		Name:      name,
		CreatedAt: time.Now(),
	}
	d.sessions[id] = sess
	return sess, nil
}

// ListSessions returns all tracked sessions.
func (d *Daemon) ListSessions() []*Session {
	result := make([]*Session, 0, len(d.sessions))
	for _, s := range d.sessions {
		result = append(result, s)
	}
	return result
}

// KillSession kills a tracked session by ID, removing it from both the registry and tmux.
func (d *Daemon) KillSession(id string) error {
	sess, ok := d.sessions[id]
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	if err := d.mgr.Kill(sess.Name); err != nil {
		return fmt.Errorf("kill tmux session: %w", err)
	}

	delete(d.sessions, id)
	return nil
}

// GetSession returns a tracked session by ID.
func (d *Daemon) GetSession(id string) (*Session, bool) {
	sess, ok := d.sessions[id]
	return sess, ok
}

// Sync discovers existing ca-* tmux sessions and adds any that aren't
// already in the registry. This supports the agent-as-observer model:
// sessions persist independently, and the daemon picks them up on restart.
func (d *Daemon) Sync() error {
	tmuxSessions, err := d.mgr.List(sessionPrefix)
	if err != nil {
		return fmt.Errorf("list tmux sessions: %w", err)
	}

	// Build a set of tmux names already tracked
	tracked := make(map[string]bool, len(d.sessions))
	for _, s := range d.sessions {
		tracked[s.Name] = true
	}

	for _, ts := range tmuxSessions {
		if tracked[ts.Name] {
			continue
		}

		// Parse project and id from ca-<project>-<id>
		project, id := parseCCName(ts.Name)

		d.sessions[id] = &Session{
			ID:        id,
			Project:   project,
			Name:      ts.Name,
			CreatedAt: ts.CreatedAt,
		}
	}
	return nil
}

// parseCCName extracts project and id from a ca-<project>-<id> session name.
// The id is the last segment after the final hyphen.
func parseCCName(name string) (project, id string) {
	without := strings.TrimPrefix(name, sessionPrefix)
	lastDash := strings.LastIndex(without, "-")
	if lastDash == -1 {
		return without, without
	}
	return without[:lastDash], without[lastDash+1:]
}

func shortID() string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

package tmux

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Session represents a tmux session.
type Session struct {
	Name      string
	Windows   int
	CreatedAt time.Time
}

// Manager executes tmux operations via the tmux CLI.
type Manager struct{}

// NewManager returns a new Manager.
func NewManager() *Manager {
	return &Manager{}
}

// Create starts a new detached tmux session with the given name.
// If command is non-empty, it runs as the initial shell command.
func (m *Manager) Create(name string, command string) error {
	args := []string{"new-session", "-d", "-s", name}
	if command != "" {
		args = append(args, command)
	}
	return m.run(args...)
}

// List returns all tmux sessions whose names start with prefix.
// An empty prefix returns all sessions.
func (m *Manager) List(prefix string) ([]Session, error) {
	out, err := m.output("list-sessions", "-F", "#{session_name}\t#{session_windows}\t#{session_created}")
	if err != nil {
		// "no server running" or "no sessions" — not an error, just empty
		if strings.Contains(err.Error(), "no server running") || strings.Contains(err.Error(), "no sessions") {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}

		sName := parts[0]
		if prefix != "" && !strings.HasPrefix(sName, prefix) {
			continue
		}

		windows, _ := strconv.Atoi(parts[1])
		created, _ := strconv.ParseInt(parts[2], 10, 64)

		sessions = append(sessions, Session{
			Name:      sName,
			Windows:   windows,
			CreatedAt: time.Unix(created, 0),
		})
	}
	return sessions, nil
}

// Kill destroys a tmux session by name.
func (m *Manager) Kill(name string) error {
	return m.run("kill-session", "-t", name)
}

// Capture returns the visible content of the first pane in the session.
func (m *Manager) Capture(name string) (string, error) {
	return m.output("capture-pane", "-t", name, "-p")
}

// SendKeys types text into the first pane of the session, followed by Enter.
func (m *Manager) SendKeys(name string, keys string) error {
	return m.run("send-keys", "-t", name, keys, "Enter")
}

// run executes a tmux command and returns any error.
func (m *Manager) run(args ...string) error {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}

// output executes a tmux command and returns its stdout.
func (m *Manager) output(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

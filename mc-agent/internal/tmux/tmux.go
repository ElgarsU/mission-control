package tmux

import "time"

// Session represents a tmux session.
type Session struct {
	Name      string
	Windows   int
	CreatedAt time.Time
}

// Manager executes tmux operations.
type Manager struct{}

// NewManager returns a new Manager.
func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Create(name string, command string) error {
	panic("not implemented")
}

func (m *Manager) List(prefix string) ([]Session, error) {
	panic("not implemented")
}

func (m *Manager) Kill(name string) error {
	panic("not implemented")
}

func (m *Manager) Capture(name string) (string, error) {
	panic("not implemented")
}

func (m *Manager) SendKeys(name string, keys string) error {
	panic("not implemented")
}

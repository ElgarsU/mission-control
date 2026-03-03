package daemon

import "mission-control/mc-agent/internal/tmux"

// TmuxAdapter wraps tmux.Manager to satisfy the SessionManager interface.
type TmuxAdapter struct {
	mgr *tmux.Manager
}

// NewTmuxAdapter creates an adapter around a real tmux.Manager.
func NewTmuxAdapter(mgr *tmux.Manager) *TmuxAdapter {
	return &TmuxAdapter{mgr: mgr}
}

func (a *TmuxAdapter) Create(name string, command string) error {
	return a.mgr.Create(name, command)
}

func (a *TmuxAdapter) List(prefix string) ([]ManagedSession, error) {
	sessions, err := a.mgr.List(prefix)
	if err != nil {
		return nil, err
	}
	result := make([]ManagedSession, len(sessions))
	for i, s := range sessions {
		result[i] = ManagedSession{
			Name:      s.Name,
			Windows:   s.Windows,
			CreatedAt: s.CreatedAt,
		}
	}
	return result, nil
}

func (a *TmuxAdapter) Kill(name string) error {
	return a.mgr.Kill(name)
}

func (a *TmuxAdapter) Capture(name string) (string, error) {
	return a.mgr.Capture(name)
}

func (a *TmuxAdapter) SendKeys(name string, keys string) error {
	return a.mgr.SendKeys(name, keys)
}

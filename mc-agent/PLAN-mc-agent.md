# mc-agent — Implementation Plan

> Local daemon for Mission Control. Manages Coding Agent tmux sessions, connects to relay, exposes menu bar + TUI.

## Build Phases

### Phase 1: Foundation (current)
- **tmux package** (`internal/tmux/`): create, list, kill sessions via `os/exec`
- **Daemon skeleton** (`internal/daemon/`): session registry, command dispatch
- **WebSocket client** (`internal/ws/`): connect to relay, send/receive JSON messages
- **CLI wiring**: `mc-agent start` boots daemon, connects WS, begins managing sessions
- No monitoring, no TUI, no menu bar yet

### Phase 2: Discord Integration
- Wire up message types: `session.created`, `session.closed`, `session.list`
- Handle inbound: `session.create`, `session.kill`, `session.list_req`, `session.input`
- `session.input` → `tmux send-keys` injection
- Security boundary enforcement: reject terminal session commands from WebSocket

### Phase 3: Monitoring
- **Monitor package** (`internal/monitor/`): poll `tmux capture-pane`, detect "waiting" patterns (Claude Code should have some hooks, review documentation)
- `session.output` streaming to relay (batched)
- `session.attention` alerts (question, approval, error detection)
- Output mode support (quiet/full/summary) per session

### Phase 4: TUI
- **TUI package** (`internal/tui/`): Bubble Tea interactive UI
- **IPC package** (`internal/ipc/`): unix socket server in daemon, client in TUI
- `mc-agent tui` connects to daemon via unix socket
- Session list, attach, create, kill from TUI

### Phase 5: Polish
- **Menu bar** (`internal/menubar/`): systray integration, built into daemon process
- `mc-agent start --headless` flag for server use (no systray)
- Reconnection handling (WS auto-reconnect with backoff)
- launchd plist management (auto-install, toggle)
- Logging and observability

## Package Responsibilities

| Package | Role |
|---------|------|
| `cmd/mc-agent/` | CLI entry point (Cobra). Subcommand routing only — no business logic. |
| `internal/daemon/` | Core daemon loop. Session registry, command dispatch, lifecycle management. |
| `internal/tmux/` | Thin wrapper over tmux CLI. Create, list, kill, capture-pane, send-keys. |
| `internal/monitor/` | Polls tmux output, detects Coding Agent "waiting" patterns, emits events. |
| `internal/ws/` | WebSocket client. Connect to relay, marshal/unmarshal protocol messages. |
| `internal/ipc/` | Unix socket server (daemon side) + client (TUI/CLI side). Local command transport. |
| `internal/tui/` | Bubble Tea TUI. Connects to daemon via IPC. |
| `internal/menubar/` | systray menu bar. Runs in daemon process, calls daemon API directly. |
| `internal/config/` | Config file loading, defaults, validation. |

## Key Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/spf13/cobra` | CLI framework — subcommand routing, help generation |
| `github.com/gorilla/websocket` | WebSocket client for relay connection |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/getlantern/systray` | macOS menu bar integration |

## Design Decisions

### Daemon model
Single long-running process started via `mc-agent start`. Manages all sessions, holds the WebSocket connection, runs the menu bar (unless `--headless`). 
Frontends (TUI, menu bar, relay) send commands to the daemon — they don't manage sessions directly.

### IPC via unix socket
The daemon listens on a unix socket (`~/.mc-agent.sock`). TUI and CLI commands connect as clients. 
JSON-based request/response protocol matching the relay message types. 
Unix socket chosen over TCP because: no port conflicts, automatic file-permission-based access control, no network exposure.

### Security boundary enforcement
The daemon is the enforcement point — not the relay. Commands arriving over WebSocket (relay) are restricted to Coding Agent session operations only. 
Terminal sessions (`term-*`) can only be created via local IPC. The daemon validates command origin (local vs relay) before executing.

### Agent-as-observer
tmux sessions are independent of the agent. If the agent crashes or restarts, sessions keep running. 
On restart, the agent discovers existing `ca-*` sessions and resumes monitoring. This means no critical state is lost on agent failure.

## What's Deferred

These are explicitly out of scope until their respective phases:
- Output pattern detection and attention alerts (Phase 3)
- TUI frontend (Phase 4)
- Menu bar / systray (Phase 5)
- launchd integration (Phase 5)
- Config file support (Phase 5)
- Multiple machine support (future, if ever)

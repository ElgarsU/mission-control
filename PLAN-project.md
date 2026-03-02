# Project Implementation

## Project Structure

```
mission-control/
├── ARCH.md                  # Architecture document
├── mc-agent/                # Go — single binary (MacBook)
│   ├── cmd/
│   │   └── mc-agent/
│   │       └── main.go      # Entry point (cobra): start, tui (no args = help)
│   ├── internal/
│   │   ├── daemon/          # Core daemon: session management, monitoring loop
│   │   ├── tmux/            # tmux operations (create, capture-pane, send-keys, kill)
│   │   ├── monitor/         # Output monitoring & pattern detection
│   │   ├── ws/              # WebSocket client (relay connection)
│   │   ├── ipc/             # Unix socket server + client (daemon ↔ frontends)
│   │   ├── tui/             # Interactive TUI frontend
│   │   ├── menubar/         # systray menu bar frontend
│   │   └── config/
│   ├── go.mod
│   └── go.sum
├── mc-relay/                # TypeScript relay + Discord bot (VPS)
│   ├── src/
│   │   ├── bot/             # Discord bot logic, slash commands
│   │   ├── relay/           # WebSocket server, message routing
│   │   ├── sessions/        # Session state management
│   │   └── index.ts
│   ├── package.json
│   └── tsconfig.json
├── infra/                   # Infrastructure configs
│   ├── wireguard/           # WG configs for MacBook, VPS, Phone
│   ├── systemd/             # systemd units for VPS services
│   ├── launchd/             # launchd plist for MacBook agent
│   └── ssh/                 # SSH configs
└── scripts/                 # Helper scripts
    └── ...                  #
```

## Phases

### Phase 1: Foundation
- WireGuard setup between MacBook and VPS
- Basic mc-agent: tmux session create/list/kill
- Basic mc-relay: WebSocket server accepting agent connections

### Phase 2: Discord
- Bot setup with discord.js
- Slash commands (`/cc start`, `/cc stop`, `/cc list`)
- Channel management (auto-create, auto-close)
- Message bridging (Discord ↔ tmux)

### Phase 3: Monitoring
- Output pattern detection for Claude "waiting" states
- Attention alerts with @user pings
- Channel output modes (quiet/full/summary)

### Phase 4: Terminal
- SSH chain setup (Phone → VPS → MacBook)
- `mc-agent tui` subcommand

### Phase 5: Polish
- Menu bar app with systray
- Reconnection handling and error recovery
- Logging and observability

---

## Implementation Notes

Gotchas and constraints to keep in mind during development.

### mc-agent: command filtering by transport

The daemon accepts commands from two transports: unix socket (local) and WebSocket (relay). Commands must be filtered based on origin:

- **Unix socket (trusted):** All commands allowed — Claude sessions, terminal sessions, kill, list, attach.
- **WebSocket (untrusted):** Claude session commands only — `session.create`, `session.input`, `session.kill`, `session.list_req`, `session.mode`. 

Reject anything else. Never allow terminal session creation or arbitrary exec over WebSocket.
This is the enforcement point for the security boundary. Do not rely on the relay to self-restrict — the daemon must enforce it.

### mc-agent: lazy initialization

`mc-agent` is a single binary with two subcommands (`start`, `tui`). Running `mc-agent` with no args prints help. Go's `init()` functions run regardless of which subcommand is invoked. Do not initialize systray, WebSocket connections, or tmux monitoring at import time — keep all of that behind explicit startup in the respective subcommand handler. Use a CLI framework like cobra that naturally isolates subcommand initialization.

---

## Open Questions

1. **Output pattern detection:** How does Claude Code signal it's waiting? Critical for `quiet` mode. 
2. **Discord rate limits:** 5 messages per 5 seconds per channel. `full` mode needs smart batching (edit-in-place, then new message on pause).
3. **Reconnection:** What happens when MacBook goes to sleep or WireGuard disconnects? Agent should auto-reconnect. Relay should show sessions as "disconnected" in Discord.
4. **Multiple initial prompts:** Should `/cc start` support piping in a multi-line prompt from Discord?
5. ~~**Security:** Is shared-secret + WireGuard sufficient, or do we want mTLS on the WebSocket?~~ Resolved: WireGuard is the auth layer. No token or mTLS needed.
6. **Multiple machines in future?** Design for one machine now, but worth considering naming conventions.

# Mission Control - Architecture

> A system for managing Claude Code sessions locally (via menu bar app) or remotely via Discord.
> System provides option to ssh to development machine and gain terminal access.

## High-Level Architecture

```
+──────────────+       WireGuard       +──────────────+   WireGuard (optional for Discord access) +──────────────+
│   MacBook    │◄─────────────────────►│     VPS      │◄──────────────────────►│    Phone     │
│ (Dev Machine)│     10.0.0.2          │ (Jump Server)│      10.0.0.3          │  (Remote)    │
│              │                       │  10.0.0.1    │                        │              │
│ - tmux       │                       │ - mc-relay   │                        │ - Discord App│
│ - Claude Code│◄─────────────────────►│   (TS/Node)  │                        │ - Terminal   │
│ - mc-agent   │  WebSocket over WG    │ - sshd       │                        │  (WireGuard) │
│   (Go)       │                       │              │                        │              │
+──────────────+                       +──────────────+                        +──────────────+
```

**Three components:** mc-agent (MacBook), mc-relay (VPS), Phone (Discord app + terminal app).

**Key flows:**
- Start session from laptop OR Discord → tmux session created → Discord channel auto-created
- Claude needs attention → agent detects pattern → bot pings you in Discord
- You reply in Discord → message injected into tmux pane
- Direct terminal access: Phone → SSH → VPS → SSH → MacBook → `mc-agent tui`

## Technology Stack

### Components

| Component  |        Tech         | Rationale |
|------------|---------------------|-----------|
| `mc-agent` |       **Go**        | Single binary: daemon + menu bar (`systray`) + TUI (`mc-agent tui`). Manages tmux, monitors output, WebSocket client. |
| `mc-relay` | **TypeScript/Node** | Discord bot (discord.js) + WebSocket server. Bridges Discord ↔ agent. |

## Component Details

### mc-agent (MacBook) — Go

A single Go binary that runs as a **daemon** and accepts commands from multiple frontends.

#### Architecture: Daemon + Frontends

```
┌────────────────────────────────────────────┐
│             mc-agent daemon                 │
│                                             │
│  - tmux session management                  │
│  - Claude output monitoring                 │
│  - WebSocket client (relay connection)      │
│  - Unix socket server (local IPC)           │
│                                             │
│       ▲            ▲            ▲           │
│       │            │            │           │
│   Menu Bar        TUI        Relay         │
│  (systray)     (mc-agent   (WebSocket      │
│                 tui)       from VPS)        │
└────────────────────────────────────────────┘
```

**The daemon** is the core — it runs persistently (launched at login via launchd), maintains the WebSocket connection to relay, and monitors all tmux sessions. It exposes a unix socket for local commands.

**Frontends** are just different ways to talk to the daemon:

| Frontend     | How it works                                                                                                   | Used by              |
|--------------|----------------------------------------------------------------------------------------------------------------|----------------------|
| **Menu bar** | Built into the daemon process via `systray`. Buttons send commands to the daemon's internal API directly.       | User on local machine  |
| **TUI**      | `mc-agent tui`. Interactive wrapper over the daemon's unix socket. Lists sessions, attach, start new sessions.  | User from phone via SSH or local terminal |
| **Relay**    | `session.create`, `session.input`, `session.kill` messages over WebSocket.                                      | Discord slash commands |

All frontends talk to the same daemon. The daemon does all the actual work.

**Starting the daemon:**
- `mc-agent start` — starts daemon with menu bar
- `mc-agent start --headless` — starts daemon without menu bar (for Mac Mini / server use)
- `mc-agent` (no args) — prints help

**Launch at Login:**
- Enabled by default. mc-agent manages its own launchd plist (`~/Library/LaunchAgents/`).
- Toggle in menu bar UI: Settings → "Launch at Login" checkbox.
- In headless mode, managed via `mc-agent tui` settings or config file.
- First run auto-installs the plist. Toggling off removes it.

**Responsibilities:**
- Manage tmux sessions (create, list, attach, kill)
- Launch Claude Code instances inside tmux sessions
- Start plain terminal sessions (tmux sessions without Claude, for file browsing etc.)
- Monitor Claude Code output for "waiting for input" patterns (question marks, prompts, tool approval requests)
- Forward Claude output/state to VPS relay
- Receive and inject input from VPS relay into the correct tmux session
- Report session health/status

**Key details:**
- Single static binary, runs as a launchd daemon
- Each Claude Code instance gets its own tmux session named `cc-<project>-<short-id>` (e.g., `cc-mission-control-a1b2`)
- Plain terminal sessions use naming `term-<short-id>` — not exposed to Discord
- Monitors tmux pane content via `tmux capture-pane` polling
- Connects to VPS relay via WebSocket over WireGuard tunnel

**Agent-as-observer model:** tmux sessions persist independently of agent. Agent monitors, injects input, manages lifecycle but does NOT own the process. If the agent has a bug or needs updating, Claude sessions keep running — you just temporarily lose Discord integration.

**Agent's role with tmux:**
- **Create** sessions (`tmux new-session -d -s cc-project-id`)
- **Monitor** sessions (`tmux capture-pane` to read output)
- **Inject input** (`tmux send-keys` to type into sessions)
- **List/discover** sessions (`tmux list-sessions` filtered by `cc-` prefix)
- **Kill** sessions on explicit user request only

**Agent behavior table:**

| Event | tmux sessions | mc-agent |
|-------|--------------|----------|
| Agent toggled OFF (menu bar) | Keep running. Claude continues working. | Stops monitoring, disconnects WS. You lose Discord visibility. |
| Agent toggled ON | Reconnects. Discovers existing sessions. Resumes monitoring. | Re-syncs session list to Discord. |
| Laptop sleeps | Suspended (resume on wake) | Suspended (resume on wake) |
| Agent crashes | Keep running | Needs restart. Sessions still intact. |
| Kill via menu bar/TUI/Discord | Agent sends `tmux kill-session` | Agent cleans up + notifies Discord |
| Claude finishes & exits | tmux session auto-closes | Detects exit, notifies Discord |

You can still use tmux directly — `tmux attach -t cc-mission-control-a1b2` works while the agent watches from the side, forwarding to Discord in parallel.

### mc-relay (VPS) — TypeScript/Node

**Responsibilities:**
- Run the Discord bot (discord.js)
- Bridge Discord messages ↔ MacBook agent commands
- Manage Discord channel ↔ Claude session mapping

Note: The VPS also runs `sshd` as an SSH jump host for terminal access, but that's an OS-level service — not part of mc-relay.

### Phone — Interaction Layer

**Discord App (primary for Claude interaction):**
- Read Claude output in channel
- Reply to Claude questions by typing in channel
- Start new sessions via slash commands
- Get push notifications when Claude needs attention

**Terminal App (for direct shell access):**
- SSH to MacBook (directly via WireGuard, or via VPS jump host) → `mc-agent tui`

```
=== Mission Control ===
Active sessions:
  1) cc-mission-control-a1b2  [Claude waiting]
  2) cc-api-server-c3d4       [Running]
  3) term-f7g8                [Terminal]

Actions:
  a) Attach to session
  s) New Claude Code session
  t) New terminal session
  q) Quit
```

`mc-agent tui` is a frontend for the daemon — it connects via unix socket and sends the same commands as the CLI or menu bar. The MacBook's SSH config launches it automatically for remote sessions (e.g., via `ForceCommand` or shell profile).

Terminal sessions (`term-*`) are plain tmux shells, not exposed to Discord. Only available via TUI and menu bar — never via relay/Discord (see Security Boundary).

---

## Menu Bar Agent UX

Built into the daemon process via `systray` — not a separate app.

**Status colors:**
- **Green** = connected to VPS, all sessions healthy
- **Yellow** = one or more sessions need attention
- **Red** = disconnected from VPS
- **Gray** = agent off

**Click** = toggle agent on/off

**Dropdown menu:**
```
● Connected (2 sessions)
──────────────────────────────────
  cc-mission-control-a1b2  ⚡ waiting
    [Attach]  [Kill]
  cc-api-server-c3d4       ● running
    [Attach]  [Kill]
──────────────────────────────────
  [+ New Claude Session]
──────────────────────────────────
  Settings
  Quit
```

- **Attach:** Opens a new Terminal.app window and runs `tmux attach -t <session>`. Agent continues monitoring in parallel.
- **Kill:** Sends `tmux kill-session`, notifies Discord to mark channel as closed.
- **Mode:** Menu bar by default, `--headless` flag for server use (e.g., Mac Mini without display).

The toggle starts/stops the WebSocket connection and session monitoring. Tmux sessions themselves persist independently.

---

## Discord Bot Design

### Server Structure

```
My Dev Server
├── #command-center        (slash commands, bot announcements)
├── 📂 Active Sessions
│   ├── #cc-mission-control-a1b2
│   └── #cc-api-server-c3d4
└── 📂 Closed Sessions
    └── #closed-cc-frontend-e5f6
```

### Slash Commands //TODO

| Command | Description |
|---------|-------------|
| `/cc start <project> [--dir path] [--mode quiet\|full] [--prompt "..."]` | Start new Claude Code session |
| `/cc stop <session>` | Kill a session |
| `/cc list` | List active sessions |
| `/cc mode <quiet\|full\|summary>` | Switch output mode in current channel |
| `/cc send <text>` | Explicitly send input (alternative to just typing) |
| `/cc input on\|off` | Toggle input mode per channel |

### Output Formatting — Hybrid Approach

**Cleaned markdown** for regular Claude output:

```
🤖 Claude:
I'll fix the authentication bug in `src/auth.ts`. Let me read the file first.

📄 Reading src/auth.ts...

🤖 Claude:
Found the issue - the token expiry check is using `<` instead of `<=`.
I'll update line 42.
```

**Structured embeds** for attention alerts, tool approvals, and errors:

```
┌─── 🤖 Claude Response ─── (blue sidebar)
│ I'll fix the authentication bug in src/auth.ts.
│ Let me read the file first.
└────────────────────────────

┌─── 🔧 Tool: Read File ─── (gray sidebar)
│ src/auth.ts
└────────────────────────────

┌─── ⚠️ Approval Required ─── (yellow sidebar)
│ Edit: src/auth.ts (line 42)
│
│ Reply to approve or deny
└────────────────────────────
```

Embeds give visual hierarchy — you can instantly spot attention items (yellow) vs normal output (blue) vs tool use (gray). Embeds have a 4096 char limit per embed (vs 2000 for regular messages), and on mobile the colored sidebars make scanning easy.

### Input Mode — Toggle

`/cc input on|off` per channel:
- **ON:** Messages go to Claude as input
- **OFF:** Messages are just notes/discussion (not forwarded)

### Session Close Behavior — Auto-close

When Claude exits, the tmux session is killed. The Discord channel gets renamed to `closed-cc-...` and locked (moved to Closed Sessions category). Channels are NOT deleted — they are kept but marked closed.

---

## Channel Output Modes

Each Discord channel has a configurable output mode, toggled via `/cc mode <mode>`:

| Mode | Behavior | Use Case |
|------|----------|----------|
| `quiet` | Only posts when Claude needs attention (questions, approvals, errors, task completion) | Fire-and-forget tasks |
| `full` | Streams all Claude output (batched every 2-3s, edits last message until pause, then posts new) | Active monitoring |
| `summary` | Posts periodic AI-summarized progress + attention alerts | Long-running tasks |

Default mode is configurable per-session at creation: `/cc start myproject --mode quiet`

Can be switched mid-session: `/cc mode full`

---

## Network Layer

### WireGuard Topology

```
MacBook (10.0.0.2) ◄──► VPS (10.0.0.1) ◄──► Phone (10.0.0.3, optional)
```

- **MacBook ↔ VPS:** Always-on tunnel. Agent and relay communicate over this.
- **Phone ↔ VPS:** Optional. Adds security for terminal access. Without it, phone SSH connects to VPS public IP directly (SSH keys required).
- **VPS firewall:** Only WireGuard (51820/udp) + SSH (22/tcp) exposed. WebSocket runs internally over WG.

### SSH Terminal Access

Two options depending on whether the phone has WireGuard:

```
With WireGuard:    Phone → SSH → MacBook (10.0.0.2) → mc-agent tui
Without WireGuard: Phone → SSH → VPS (public IP) → SSH → MacBook (10.0.0.2) → mc-agent tui
```

The VPS jump is handled by standard SSH (`ProxyJump` in `~/.ssh/config`). This is plain sshd — no mc-relay involvement.

---

## Communication Protocol (Agent ↔ Relay)

**Transport:** WebSocket over WireGuard (bidirectional, streaming-friendly)

**Authentication:** Shared secret token in WS handshake header (sufficient since WireGuard already encrypts the tunnel)

### Security Boundary

**Discord can only control Claude Code sessions — never arbitrary shell access.**

The relay/agent protocol is deliberately restricted:
- `session.create` can ONLY start a Claude Code process inside a tmux session. There is no message type or command to start a raw shell/terminal.
- `session.input` delivers text exclusively to a Claude Code session's stdin. The agent must validate the target session is a Claude Code session before injecting input.
- `session.kill` can only terminate existing Claude Code sessions.
- There is no `exec`, `shell`, or `run` command in the protocol. The agent must reject any unrecognized message types.

Plain terminal sessions (`term-*`) can only be created via local frontends (TUI, menu bar, CLI) — never via the relay/Discord. This is an intentional separation: Discord is the Claude control plane, SSH + TUI is the escape hatch for raw shell access.

### Message Types

```
Agent → Relay:
  session.created    { session_id, project, created_at }
  session.output     { session_id, content, timestamp }
  session.attention  { session_id, type: "question"|"approval"|"error", context }
  session.closed     { session_id }
  session.list       { sessions: [...] }

Relay → Agent:
  session.input      { session_id, content }
  session.create     { project, initial_prompt?, working_dir?, mode? }
  session.kill       { session_id }
  session.list_req   {}
  session.mode       { session_id, mode: "quiet"|"full"|"summary" }
```

`session.create` starts **Claude Code only** — the agent hardcodes the launched process. There is no parameter to specify an arbitrary binary.

---

## Session Lifecycle Flows

### Starting from Laptop

1. User clicks "+ New Claude Session" in menu bar (or runs `mc-agent tui` in terminal)
2. Daemon creates tmux session `cc-mission-control-a1b2`, launches Claude Code
3. Daemon sends `session.created` to relay
4. Relay/bot creates Discord channel `#cc-mission-control-a1b2`
5. Output streams to Discord per channel mode

### Starting from Discord

1. User types `/cc start mission-control --mode full` in Discord
2. Bot sends `session.create` to daemon via relay
3. Daemon creates tmux session + Claude Code
4. Bot creates channel, streaming begins

### Starting from TUI (Phone via SSH)

1. User SSHs to MacBook (via VPS jump), `mc-agent tui` launches automatically
2. User selects "New Claude Code session" or "New terminal session"
3. TUI sends command to daemon via unix socket
4. Daemon creates tmux session (Claude or plain shell)
5. TUI attaches to the new session
6. If Claude session: daemon notifies relay, Discord channel created

### Claude Needs Attention

1. Agent detects Claude waiting (output pattern matching on `tmux capture-pane`)
2. Sends `session.attention` to relay
3. Bot posts alert + @user ping in channel
4. User replies in Discord
5. Bot sends `session.input` → agent injects via `tmux send-keys`

### Closing a Session

1. Claude exits naturally OR user runs `/cc stop`
2. Agent cleans up tmux session (auto-close), sends `session.closed`
3. Bot renames channel to `closed-cc-...` and locks it (moved to Closed Sessions category)

---

## Project Structure

```
mission-control/
├── arch.md                  # This architecture document
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
    └── mc                   # CLI wrapper (calls mc-agent)
```

---

## Implementation Phases

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
- **WebSocket (untrusted):** Claude session commands only — `session.create`, `session.input`, `session.kill`, `session.list_req`, `session.mode`. Reject anything else. Never allow terminal session creation or arbitrary exec over WebSocket.

This is the enforcement point for the security boundary. Do not rely on the relay to self-restrict — the daemon must enforce it.

### mc-agent: lazy initialization

`mc-agent` is a single binary with two subcommands (`start`, `tui`). Running `mc-agent` with no args prints help. Go's `init()` functions run regardless of which subcommand is invoked. Do not initialize systray, WebSocket connections, or tmux monitoring at import time — keep all of that behind explicit startup in the respective subcommand handler. Use a CLI framework like cobra that naturally isolates subcommand initialization.

---

## Open Questions

1. **Output pattern detection:** How does Claude Code signal it's waiting? Need to reverse-engineer the terminal output patterns (spinner stops, prompt appears, question text). Critical for `quiet` mode.
2. **Discord rate limits:** 5 messages per 5 seconds per channel. `full` mode needs smart batching (edit-in-place, then new message on pause).
3. **Reconnection:** What happens when MacBook goes to sleep or WireGuard disconnects? Agent should auto-reconnect. Relay should show sessions as "disconnected" in Discord.
4. **Multiple initial prompts:** Should `/cc start` support piping in a multi-line prompt from Discord?
5. **Security:** Is shared-secret + WireGuard sufficient, or do we want mTLS on the WebSocket?
6. **Multiple machines in future?** Design for one machine now, but worth considering naming conventions.

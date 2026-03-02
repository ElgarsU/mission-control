# Mission Control - Architecture

> A system for managing Coding Agent sessions locally (via menu bar app) or remotely via Discord.
> System provides an option to ssh to development machine and gain terminal access.

## High-Level Architecture

```
+──────────────+       WireGuard       +──────────────+                       +──────────────+
│   MacBook    │◄─────────────────────►│     VPS      │◄─────────────────────►│    Phone     │
│ (Dev Machine)│                       │ (Jump Server)│                       │  (Remote)    │
│   10.0.0.2   │                       │  10.0.0.1    │                       │              │
│ - mc-agent   │       WebSocket       │ - mc-relay   │                       │ - Discord App│
│ -            │◄─────────────────────►│    (Node)    │◄─────────────────────►│ - Terminal   │
│ -            │         SSH           │ -            │          SSH          │              │
+──────────────+                       +──────────────+                       +──────────────+
```

**Key flows & functionality:**
- Start/terminate Coding Agent session from laptop OR Discord: tmux session created / **mc-agent** and Discord channel auto-created / **mc-relay**
- Coding Agent needs attention → **mc-agent** detects a pattern → **mc-relay** pings you in Discord
- You reply in Discord (via **mc-relay**) → the message is injected into tmux pane with **mc-agent**
- Direct terminal access: Phone → SSH → VPS → SSH → MacBook → `mc-agent tui`
- Agent-as-observer model - tmux sessions persist independently of **mc-agent**, **mc-agent** monitors, injects input, manages lifecycle but does NOT own the process. 

------------------------------------------------------------------------------------------------------------------------------------

## Communication Protocols

### Agent ↔ Relay
**Transport:** WebSocket over WireGuard (bidirectional, streaming-friendly)
**Authentication:** WireGuard — only authorized WG peers can reach the relay. No application-layer auth needed.

### Phone (Discord) ↔ Relay
**Transport:** Discord API (HTTPS + WebSocket, managed by discord.js)
**Authentication:** Discord OAuth / bot token

### Phone (Terminal) ↔ VPS (SSH) ↔ MacBook (mc-agent tui)
**Transport:** SSH (Phone → VPS over public internet, VPS → MacBook over WireGuard)
**Authentication:** SSH key auth on both hops

------------------------------------------------------------------------------------------------------------------------------------

## Components

### mc-agent (MacBook - backend & orchestrator)

A single Go binary that runs as a daemon and accepts commands from multiple frontends.
Central backend & orchestrator.

```
┌────────────────────────────────────────────┐
│             mc-agent daemon                 │
│                                             │
│  - tmux session management                  │
│  - Coding Agent output monitoring           │
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

**Daemon**
Core service runs persistently (launched at login via launchd), maintains the WebSocket connection to relay, and monitors all tmux sessions. 
It exposes a unix socket for local commands.

**Daemon commands:**
- `mc-agent start` — starts daemon with menu bar
- `mc-agent start --headless` — starts daemon without menu bar (for Mac Mini / server use)
- `mc-agent` (no args) — prints help

**Responsibilities:**
- Manage tmux sessions (create, list, attach, kill)
- Launch Coding Agent instances inside tmux sessions
- Start plain terminal sessions (tmux sessions without Claude, for file browsing etc.)
- Monitor Coding Agent output for "waiting for input" patterns (question marks, prompts, tool approval requests)
- Forward Coding Agent output/state to VPS relay
- Receive and inject input from VPS relay into the correct tmux session
- Report session health/status

**Key features:**
- Launch at login
  - Enabled by default. mc-agent manages its own launchd plist (`~/Library/LaunchAgents/`)
  - Toggle in menu bar UI: Settings → "Launch at Login" checkbox
  - First run auto-installs the plist. Toggling off removes it
- Single static binary, runs as a launchd daemon
- Each Coding Agent session gets its own tmux session named `ca-<project>-<short-id>` (e.g., `ca-mission-control-a1b2`)
- Each plain terminal session gets its own tmux session named `term-<project>-<short-id>` — not exposed to Discord 
- Monitors tmux pane content via `tmux capture-pane` polling
- Connects to VPS relay via WebSocket over WireGuard tunnel
- Terminal sessions (`term-*`) are plain tmux shells, not exposed to Discord. Only available via TUI and menu bar — never via relay/Discord (see Security Boundary).

**Agent's role with tmux:**
- **Create** sessions (`tmux new-session -d -s ca-project-id`)
- **Monitor** sessions (`tmux capture-pane` to read output)
- **Inject input** (`tmux send-keys` to type into sessions)
- **List/discover** sessions (`tmux list-sessions` filtered by `ca-` prefix for **mc-relay**, all sessions available for other frontends)
- **Kill** sessions on explicit user request only

**Agent behavior table:**

| Event                         | tmux sessions | mc-agent |
|-------------------------------|--------------|----------|
| Agent toggled OFF (menu bar)  | Keep running. Coding Agent continues working. | Stops monitoring, disconnects WS. You lose Discord visibility. |
| Agent toggled ON              | Reconnects. Discovers existing sessions. Resumes monitoring. | Re-syncs session list to Discord. |
| Laptop sleeps                 | Suspended (resume on wake) | Suspended (resume on wake) |
| Agent crashes                 | Keep running | Needs restart. Sessions still intact. |
| Kill via menu bar/TUI/Discord | Agent sends `tmux kill-session` | Agent cleans up + notifies Discord |
| Coding Agent finishes & exits | tmux session auto-closes | Detects exit, notifies Discord |

------------------------------------------------------------------------------------------------------------------------------------

| Frontend     | How it works                                                                                                   | Used by              |
|--------------|----------------------------------------------------------------------------------------------------------------|----------------------|
| **mc-agent gui** | Built into the daemon process via `systray`. Buttons send commands to the daemon's internal API directly.       | User on local machine  |
| **mc-agent tui**      | `mc-agent tui`. Interactive wrapper over the daemon's unix socket. Lists sessions, attach, start new sessions.  | User from phone via SSH or local terminal |
| **mc-relay**    | `session.create`, `session.input`, `session.kill` messages over WebSocket.                                      | Discord slash commands |

------------------------------------------------------------------------------------------------------------------------------------

### mc-agent tui (MacBook - frontend)
A thin wrapper over the daemon's unix socket to control the daemon.
Used to access Coding Agent sessions from the terminal (locally) or via SSH.
All functionality is implemented in the daemon, this is just a frontend.

Launching the TUI starts the daemon if it's not already running and displays a menu.
```
=== Mission Control ===
1 - Active sessions:
  1) ca-mission-control-a1b2    [Claude waiting]
  2) ca-api-server-c3d4         [Running]
  3) term-mission-control-a4f4  [Terminal]
 Actions (shown when selecting individual active session):
  a) Attach to session
  b) Kill session
  c) Back 
2 - New Coding Agent session
3 - New terminal session
4 - Stop daemon
5 - Settings
6 - Quit
```

The MacBook's SSH config launches it automatically for remote sessions (e.g., via `ForceCommand` or shell profile).
Terminal sessions (`term-*`) are plain tmux shells, not exposed to Discord. Only available via TUI and menu bar — never via relay/Discord (see Security Boundary).

------------------------------------------------------------------------------------------------------------------------------------

### mc-agent gui (MacBook - frontend)

Built into the daemon process via `systray` — not a separate app.

**Status colors:**
- **Green** = connected to VPS, all sessions healthy
- **Yellow** = one or more sessions need attention
- **Red** = disconnected from VPS
- **Gray** = agent off

**Click** = toggle agent on/off

**Dropdown menu:**
```
-> Active (2) sessions
──────────────────────────────────
  ca-mission-control-a1b2  ⚡ waiting
    [Attach]  [Kill]
  ca-api-server-c3d4       ● running
    [Attach]  [Kill]
──────────────────────────────────
-> [New Coding Agent Session]
──────────────────────────────────
  Settings
  Quit
```

- **Attach:** Opens a new Terminal.app window and runs `tmux attach -t <session>`. Agent continues monitoring in parallel.
- **Kill:** Sends `tmux kill-session`, notifies Discord to mark channel as closed.

The toggle starts/stops the WebSocket connection and session monitoring. Tmux sessions themselves persist independently.
Terminal sessions (`term-*`) are plain tmux shells, not exposed to Discord. Only available via TUI and menu bar — never via relay/Discord (see Security Boundary).

------------------------------------------------------------------------------------------------------------------------------------

### mc-relay (VPS - frontend & orchestrator) — TypeScript/Node

**Responsibilities:**
- Run the Discord bot (discord.js)
- Bridge Discord messages ↔ MacBook agent commands
- Manage Discord channel ↔ Coding Agent session mapping

### Interface, message types & security boundary

**Discord can only control Coding Agent sessions — never arbitrary shell access.**

The relay/agent protocol is deliberately restricted:
- `session.created` input from **mc-agent** that new session is created and new channel needs to be created.
- `session.create` can ONLY start a Claude Code process inside a tmux session. There is no message type or command to start a raw shell/terminal.
- `session.input` delivers text exclusively to a Claude Code session's stdin. The agent must validate the target session is a Claude Code session before injecting input.
- `session.output` receives output from Coding Agent session (streaming via WebSocket)
- `session.kill` can only terminate existing Claude Code sessions.
- There is no `exec`, `shell`, or `run` command in the protocol. The agent must reject any unrecognized message types.

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

`session.create` starts **Coding Agent only** — the agent hardcodes the launched process. There is no parameter to specify an arbitrary binary.

Terminal sessions (`term-*`) are plain tmux shells, not exposed to Discord. Only available via TUI and menu bar — never via relay/Discord (see Security Boundary).

------------------------------------------------------------------------------------------------------------------------------------

## Session Lifecycle Flows

### Starting from Laptop

1. User clicks "+ New Claude Session" in menu bar (or runs `mc-agent tui` in terminal)
2. Daemon creates tmux session `ca-mission-control-a1b2`, launches Coding Agent
3. Daemon sends `session.created` to relay
4. Relay/bot creates Discord channel `#ca-mission-control-a1b2`
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

### Coding Agent Needs Attention

1. Agent detects Claude waiting (output pattern matching on `tmux capture-pane`)
2. Sends `session.attention` to relay
3. Bot posts alert + @user ping in channel
4. User replies in Discord
5. Bot sends `session.input` → agent injects via `tmux send-keys`

### Closing a Session

1. Claude exits naturally OR user runs `/cc stop`
2. Agent cleans up tmux session (auto-close), sends `session.closed`
3. Bot renames channel to `closed-ca-...` and locks it (moved to Closed Sessions category)

------------------------------------------------------------------------------------------------------------------------------------

## Discord Bot Design

### Server Structure

```
My Dev Server
├── #mission-control        (slash commands, bot announcements)
├── 📂 Active Sessions
│   ├── #ca-mission-control-a1b2
│   └── #ca-api-server-c3d4
└── 📂 Closed Sessions
    └── #closed-ca-frontend-e5f6
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

Embeds give visual hierarchy — you can instantly spot attention items (yellow) vs normal output (blue) vs tool use (gray). 
Embeds have a 4096 char limit per embed (vs 2000 for regular messages), and on mobile the colored sidebars make scanning easy.

### Input Mode — Toggle

`/cc input on|off` per channel:
- **ON:** Messages go to Claude as input
- **OFF:** Messages are just notes/discussion (not forwarded)

### Session Close Behavior — Auto-close

When Claude exits, the tmux session is killed. The Discord channel gets renamed to `closed-ca-...` and locked (moved to Closed Sessions category). 
Channels are NOT deleted — they are kept but marked closed.

### Channel Output Modes

Each Discord channel has a configurable output mode, toggled via `/cc mode <mode>`:

| Mode | Behavior | Use Case |
|------|----------|----------|
| `quiet` | Only posts when Claude needs attention (questions, approvals, errors, task completion) | Fire-and-forget tasks |
| `full` | Streams all Claude output (batched every 2-3s, edits last message until pause, then posts new) | Active monitoring |
| `summary` | Posts periodic AI-summarized progress + attention alerts | Long-running tasks |

Default mode is configurable per-session at creation: `/cc start myproject --mode quiet`

Can be switched mid-session: `/cc mode full`

---

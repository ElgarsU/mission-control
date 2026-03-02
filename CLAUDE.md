# Mission Control

Personal infrastructure project: VPS + WireGuard VPN + relay server for managing devices.

## Architecture
- See `arch.md` for full architecture doc
- See `infra/vps-provisioning.md` for VPS setup plan

## Infrastructure
- **VPS:** Hetzner CX22, Ubuntu 24.04, IP `89.167.98.246`
- **SSH:** `ssh mc-vps` (root user, key auth only)
- **Domain:** `[...].eu` (DNS not yet configured)
- **WireGuard subnet:** `10.0.0.0/24` (VPS=.1, MacBook=.2, Phone=.3)

## Conventions
- User runs commands manually on VPS -- provide commands to copy/paste, don't SSH automatically
- Explain what commands do before asking user to run them
- Private keys never committed to repo; public keys are fine

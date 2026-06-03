# Mission Control

Personal infrastructure project: VPS + WireGuard VPN + relay server for managing devices.

> **Deployment / secrets note (2026-06-03).** The other personal apps deploy from
> the central **infra** repo (`git@github.com:ElgarsU/infra.git`). Mission Control
> is pre-MVP and NOT wired into infra yet (mc-agent runs on the MacBook via
> launchd; `mc-relay` on the VPS isn't built). When mc-relay ships it gets an
> `apps/mission-control/` entry there. **Ansible Vault has been removed** — the
> WireGuard private keys now live in `infra/ansible/group_vars/secrets.yml`
> (plaintext, gitignored; template at `secrets.yml.example`). No `ansible-vault`,
> no `--ask-vault-pass` — matches infra's simple-tools secrets convention so
> there's nothing to migrate to a new laptop.

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

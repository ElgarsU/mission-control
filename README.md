# Mission Control

A system for managing Coding Agent sessions locally (via menu bar app) or remotely via Discord, with option to SSH to development machine for terminal access.

See [ARCH.md](ARCH.md) for the high-level architecture overview.

> **Deployment note.** The other personal apps on the shared VPS deploy from the
> central **infra** repo (`git@github.com:ElgarsU/infra.git`). Mission Control is
> not wired into infra yet — it's pre-MVP (mc-agent runs on the MacBook via
> launchd; the VPS-side `mc-relay` isn't built). When `mc-relay` ships it'll get
> an `apps/mission-control/` entry there. The Ansible below (base + WireGuard) is
> kept here for now. **Ansible Vault has been removed** — secrets are plaintext
> gitignored files (see Secrets, below).

## Infrastructure

- **VPS:** Hetzner CX22, Ubuntu 24.04, Helsinki — `89.167.98.246`
- **WireGuard subnet:** `10.0.0.0/24` (VPS=.1, MacBook=.2, Phone=.3)
- **Domain:** `[...].eu`

## Ansible

Provisioning is managed with Ansible playbooks in `infra/ansible/`.

### Prerequisites

- [Ansible](https://docs.ansible.com/ansible/latest/installation_guide/) installed
- SSH access to VPS configured (`ssh mc-vps`)
- `group_vars/secrets.yml` populated with the WireGuard private keys
  (copy `group_vars/secrets.yml.example`). Plaintext, gitignored — no vault.

### Playbooks

Run from `infra/ansible/`:

| Playbook | Target | Description |
|---|---|---|
| `vps-provision.yml` | VPS | Full VPS setup (runs all vps-* playbooks) |
| `vps-base.yml` | VPS | Packages, SSH hardening, Docker |
| `vps-firewall.yml` | VPS | UFW rules |
| `vps-wireguard.yml` | VPS | WireGuard install, config + service |
| `macbook-provision.yml` | MacBook | Full MacBook setup (runs all macbook-* playbooks) |
| `macbook-wireguard.yml` | MacBook | WireGuard install + config |

### Usage

```sh
cd infra/ansible

# VPS
ansible-playbook playbooks/vps-provision.yml

# MacBook only
ansible-playbook playbooks/macbook-provision.yml
```

### Secrets

The WireGuard private keys live in `group_vars/secrets.yml` — **plaintext,
gitignored** (no Ansible Vault). Copy `group_vars/secrets.yml.example` to
`group_vars/secrets.yml` and fill in the keys. This matches the simple-tools
secrets convention used by the infra repo: real values stay on the laptop, never
committed. (Vault was removed 2026-06-03 to avoid carrying it to a new laptop.)

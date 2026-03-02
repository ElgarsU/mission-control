# Mission Control

A system for managing Claude Code sessions locally (via menu bar app) or remotely via Discord, with option to SSH to development machine for terminal access.

See [ARCH.md](ARCH.md) for the high-level architecture overview.

## Infrastructure

- **VPS:** Hetzner CX22, Ubuntu 24.04, Helsinki — `89.167.98.246`
- **WireGuard subnet:** `10.0.0.0/24` (VPS=.1, MacBook=.2, Phone=.3)
- **Domain:** `[...].eu`

## Ansible

Provisioning is managed with Ansible playbooks in `infra/ansible/`.

### Prerequisites

- [Ansible](https://docs.ansible.com/ansible/latest/installation_guide/) installed
- SSH access to VPS configured (`ssh mc-vps`)
- `group_vars/vault.yml` populated with WireGuard private keys and encrypted:
  ```
  ansible-vault encrypt infra/ansible/group_vars/vault.yml
  ```

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
ansible-playbook playbooks/vps-provision.yml --ask-vault-pass

# MacBook only
ansible-playbook playbooks/macbook-provision.yml --ask-vault-pass
```

### Vault

Secrets (WireGuard private keys) are stored in `group_vars/vault.yml`, encrypted with `ansible-vault`. This file is gitignored.

```sh
# Edit secrets
ansible-vault edit infra/ansible/group_vars/vault.yml
```

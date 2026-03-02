# VPS Provisioning Plan

> **Superseded by Ansible playbooks** — see `infra/ansible/`. This file kept as reference.

## Context
First step of Mission Control implementation (Phase 1 from arch.md). Provision the Hetzner VPS so it's ready for WireGuard and mc-relay.

---

## Steps

### 1. Create Server (Hetzner Console) -- DONE
- **Project:** mission-control
- **Image:** Ubuntu 24.04
- **Type:** CX22 (2 vCPU, 4GB RAM, 40GB disk, ~4.35/mo)
- **Location:** Helsinki (hel1)
- **SSH Key:** `personal.pub` (ed25519)
- **Name:** `mc-vps`
- **IP:** 89.167.98.246

### 2. DNS (Cloudflare)
- Add `A` record: `vps.elgars.eu` -> `89.167.98.246` (gray cloud / DNS only)

### 3. SSH Config (MacBook)
Add to `~/.ssh/config`:
```
Host mc-vps
    HostName 89.167.98.246
    User root
    IdentityFile ~/.ssh/personal
    IdentitiesOnly yes
```

### 4. Basic Server Hardening
1. Set up UFW firewall: allow SSH (22), WireGuard (51820/udp), HTTP/HTTPS (80/443)
2. Enable automatic security updates (unattended-upgrades)

### 5. Install Core Software
- Docker + Docker Compose
- WireGuard
- Node.js (for mc-relay later)
- tmux, htop, curl (utilities)

### 6. WireGuard Setup (VPS side)
- Generate VPS keypair
- Configure as hub (10.0.0.1/24)
- Add peer config for MacBook (10.0.0.2)
- Enable and start WireGuard service

### 7. WireGuard Setup (MacBook side)
- Install WireGuard (`brew install wireguard-tools`)
- Generate MacBook keypair
- Configure as peer pointing to VPS
- Test connectivity: `ping 10.0.0.1` from MacBook

## Files to Create
- `infra/wireguard/vps.conf` -- VPS WireGuard config
- `infra/wireguard/macbook.conf` -- MacBook WireGuard config
- `infra/wireguard/phone.conf` -- Phone WireGuard config (template)

## Verification Checklist
- [ ] `ssh mc-vps` connects without password prompt
- [ ] `ping 10.0.0.1` from MacBook over WireGuard works
- [ ] `ping 10.0.0.2` from VPS over WireGuard works
- [ ] UFW is active: only ports 22, 51820/udp, 80, 443 open
- [ ] Docker runs: `docker run hello-world`
- [ ] DNS resolves: `vps.elgars.dev` -> 89.167.98.246

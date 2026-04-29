# LivePulse — GitHub Actions Deploy Setup

Add these secrets in **GitHub → Settings → Secrets and variables → Actions → New repository secret** (repeat for each row).

> Only `VPS_HOST`, `VPS_USER`, and `VPS_SSH_KEY` are required. The rest have safe defaults.

---

## Required

| Name | Value | Notes |
|---|---|---|
| `VPS_HOST` | `your-vps-ip-or-hostname` | e.g. `143.198.85.42` |
| `VPS_USER` | `deploy` | SSH user on the VPS (avoid running as `root`) |
| `VPS_SSH_KEY` | *(private key content)* | Paste the **raw private key** from `~/.ssh/id_ed25519` (or whichever key you use for the VPS) |

---

## Optional

| Name | Default | Notes |
|---|---|---|
| `VPS_PORT` | `22` | Change if your SSH server listens on a non-standard port |
| `VPS_KNOWN_HOSTS` | *(empty)* | If set, paste the known_hosts entry to avoid `ssh-keyscan` on first connect |
| `DEPLOY_PATH` | `~/live-pulse` | Path on the VPS where the repo is cloned |
| `GHCR_USERNAME` | *(empty)* | Only needed if `GITHUB_TOKEN` doesn't have package push permission |
| `GHCR_TOKEN` | *(empty)* | Only needed if `GHCR_USERNAME` is set |

---

## How to find the SSH private key

On your VPS provisioning machine (where you generated the key pair):

```bash
# View the private key — copy everything from -----BEGIN OPENSSH PRIVATE KEY----- to -----END OPENSSH PRIVATE KEY-----
cat ~/.ssh/id_ed25519
```

> **Security note:** The private key must be added as a GitHub Secret exactly as-is (multi-line). Do not base64-encode it — GitHub Actions handles multi-line secrets natively.

---

## One-time VPS prerequisites

Before the workflow will succeed, the VPS needs:

1. **Docker & Docker Compose plugin installed**
2. **The repo cloned** at the deploy path:
   ```bash
   git clone https://github.com/Eahtasham/live-pulse.git ~/live-pulse
   ```
3. **`.env.production` created** at `~/live-pulse/.env.production` with all production env vars ( DATABASE_URL, REDIS_URL, JWT_SECRET, AUTH_SECRET, NEXTAUTH_URL, etc.)
4. **Ports 80 and 443 open** on the VPS firewall for Caddy (letsencrypt)
5. **DNS A/CNAME records** pointing to the VPS IP:
   - `livepulse.app` → VPS IP
   - `api.livepulse.app` → VPS IP
   - `rt.livepulse.app` → VPS IP

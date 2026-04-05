## Phase 20 — Containerized Deployment & Observability 🟠 INFRA

**Goal:** All services run as Docker containers in production. CI/CD pipeline builds and deploys on push to `main`. Health monitoring catches problems before users do.

**Prerequisite:** Phase 17 (all features complete)

### Target architecture

```
                    ┌─────────────────────────────────────────┐
                    │          DigitalOcean Droplet            │
                    │   ┌──────────┐ ┌──────────┐ ┌────────┐ │
 HTTPS ──→ Caddy ──→│   │ Next.js  │ │  Go API  │ │ Go RT  │ │
                    │   │  :3000   │ │  :8080   │ │ :8081  │ │
                    │   └──────────┘ └──────────┘ └────────┘ │
                    │                  │                │     │
                    │          ┌───────┴────────┐       │     │
                    │          │  Redis :6379   │◄──────┘     │
                    │          └────────────────┘             │
                    └──────────────────┬──────────────────────┘
                                       │
                                AWS RDS PostgreSQL 16
                                AWS S3 (exports)
```

### Docker Compose (production)

Create `infra/docker-compose.prod.yml` with services:
- `web` (Next.js) — image from GHCR, port 3000
- `api` (Go API) — image from GHCR, port 8080
- `realtime` (Go Realtime) — image from GHCR, port 8081
- `redis` — redis:7-alpine with AOF persistence
- `caddy` — Caddy reverse proxy with automatic HTTPS

All services use `restart: unless-stopped`.

### Caddy reverse proxy config

```
livepulse.app {
    reverse_proxy web:3000
}
api.livepulse.app {
    reverse_proxy api:8080
}
rt.livepulse.app {
    reverse_proxy realtime:8081
}
```

Caddy handles TLS automatically via Let's Encrypt.

### GitHub Actions CI/CD

`.github/workflows/deploy.yml`:

```yaml
on:
  push:
    branches: [main]

jobs:
  build-and-deploy:
    steps:
      - Checkout code
      - Build Docker images (api, realtime, web)
      - Push to GHCR (ghcr.io/eahtasham/live-pulse/api, etc.)
      - SSH into Droplet
      - Pull latest images
      - docker compose -f infra/docker-compose.prod.yml up -d
```

### Environment variable management

- `.env.production` on the Droplet (NOT in the repo)
- Contains: `DATABASE_URL` (RDS), `REDIS_URL` (internal), `JWT_SECRET`, AWS keys, etc.
- Docker Compose references `.env.production` via `env_file:`
- No credentials baked into Docker images (verified by `docker inspect`)

### Observability improvements

**Enhanced `/healthz`:**
```json
{
  "service": "api",
  "status": "ok",
  "uptime": "4h32m",
  "checks": {
    "postgres": "ok",
    "redis": "ok"
  }
}
```
- If Postgres is down: `{"status": "unhealthy", "checks": {"postgres": "error: connection refused", "redis": "ok"}}`
- If Redis is down: same pattern

**Request ID propagation:**
- Next.js generates `X-Request-ID` (or uses incoming one)
- Passes it to Go API in proxied requests
- Go API includes it in all log entries
- Same request ID visible across service logs for tracing

**WebSocket observability:**
- Log connect/disconnect with session code and room client count
- Log message broadcast count per room
- Log subscribe/unsubscribe events for Redis channels

### Cloud resource setup

| Resource | Provider | Tier | Config |
|----------|----------|------|--------|
| Compute | DigitalOcean Droplet | $6/mo (from $200 credit) | 1 vCPU, 1GB RAM, Ubuntu |
| Database | AWS RDS PostgreSQL 16 | Free tier | db.t3.micro, 20GB |
| Object storage | AWS S3 | Free tier | 5GB, presigned URLs |
| Container registry | GitHub Container Registry | Free | Public images |
| CI/CD | GitHub Actions | Free | 2000 min/month |
| DNS | Wherever domain is registered | — | A record → Droplet IP |

### Acceptance tests

- [ ] `docker compose -f infra/docker-compose.prod.yml up` → all services start and pass health checks
- [ ] Health checks from outside Docker network succeed for all 3 services
- [ ] WebSocket connections work through the reverse proxy (upgrade headers pass through)
- [ ] Push to `main` → GitHub Actions builds all 3 images successfully
- [ ] Images are pushed to GHCR and pullable from the Droplet
- [ ] HTTPS works on all domains (no mixed content)
- [ ] Environment variables are NOT baked into images (inspecting layers shows nothing sensitive)
- [ ] `GET /healthz` → returns `unhealthy` if Postgres is down
- [ ] `GET /healthz` → returns `unhealthy` if Redis is down
- [ ] Same `request_id` appears in Next.js and Go API logs for a proxied request
- [ ] Container crash → Docker auto-restarts it (`restart: unless-stopped`)
- [ ] Droplet → RDS connection works (security group configured)
- [ ] Redis port 6379 is NOT accessible from the public internet

### Files to create

- `infra/docker-compose.prod.yml` — Production Docker Compose
- `infra/Caddyfile` — Caddy reverse proxy config
- `.github/workflows/deploy.yml` — CI/CD pipeline
- `.github/workflows/ci.yml` — Build + test on PR (no deploy)
- Update `apps/api/internal/handler/health.go` — Add DB + Redis connectivity checks
- Update `apps/realtime/internal/handler/health.go` — Add Redis connectivity check

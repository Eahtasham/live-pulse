# LivePulse

**Real-Time Polling & Q&A Platform**

LivePulse is a real-time polling and Q&A platform for live events, classrooms, webinars, and team meetings. A host creates a session and shares a short 6-character code — audience members join instantly with no sign-up required — and interact through live polls and a moderated Q&A feed. Results update in real time across all connected clients via WebSockets.

---

## Table of Contents

- [Tech Stack](#tech-stack)
- [Architecture Overview](#architecture-overview)
- [Repository Structure](#repository-structure)
- [Prerequisites](#prerequisites)
- [Local Development Setup](#local-development-setup)
- [Task Runner Reference](#task-runner-reference)
- [Backend Deep Dive](#backend-deep-dive)
  - [API Service (Go)](#api-service-go---appsapi)
  - [Realtime Service (Go)](#realtime-service-go---appsrealtime)
  - [Database Schema](#database-schema)
  - [GORM Models](#gorm-models)
- [Frontend](#frontend)
- [Shared Types](#shared-types)
- [Docker & Infrastructure](#docker--infrastructure)
- [Environment Variables](#environment-variables)
- [WebSocket Event System](#websocket-event-system)
- [API Endpoints](#api-endpoints)
- [Troubleshooting](#troubleshooting)

---

## Tech Stack

| Layer            | Technology                                      |
| ---------------- | ----------------------------------------------- |
| Monorepo         | Turborepo + pnpm workspaces                     |
| Frontend         | Next.js 16 (App Router), TypeScript, Tailwind   |
| API Service      | Go, Chi router, GORM, slog                      |
| Realtime Service | Go, gorilla/websocket, Redis Pub/Sub            |
| Database         | PostgreSQL 16                                   |
| Cache / Pub/Sub  | Redis 7                                         |
| ORM              | GORM (gorm.io)                                  |
| Migrations       | golang-migrate                                  |
| Containerization | Docker, Docker Compose                          |

---

## Architecture Overview

LivePulse uses a **two-service Go backend** behind a shared PostgreSQL database, with Redis as the communication backbone. The two Go services **never call each other over HTTP** — they are fully decoupled via Redis Pub/Sub.

```
┌─────────────┐     HTTPS      ┌─────────────────┐
│             │ ──────────────► │   web (Next.js)  │ :3000
│   Browser   │                │   UI + Auth.js    │
│             │     WSS        ├─────────────────┤
│             │ ──────────────► │ realtime (Go)    │ :8081
└─────────────┘                │ WebSocket Hub     │
                               └────────┬─────────┘
                                        │ Redis SUBSCRIBE
                                        │
                               ┌────────┴─────────┐
                               │     Redis 7       │ :6379
                               │  Pub/Sub + Cache  │
                               └────────┬─────────┘
                                        │ Redis PUBLISH
                               ┌────────┴─────────┐
                               │   api (Go)        │ :8080
                               │ REST + Biz Logic  │
                               └────────┬─────────┘
                                        │
                               ┌────────┴─────────┐
                               │  PostgreSQL 16    │ :5432
                               │  Primary Store    │
                               └──────────────────┘
```

**Request Flow — Casting a Vote:**

1. Client sends `POST /api/sessions/:code/polls/:id/vote`
2. Next.js proxy forwards to the Go **api** service
3. API validates vote → writes to Postgres → publishes `vote_update` to Redis channel `session:{code}`
4. **realtime** service (subscribed to `session:{code}`) receives the event
5. Hub broadcasts to all WebSocket clients in the matching room
6. All browsers update live charts without polling

---

## Repository Structure

```
live-pulse/
│
├── package.json              # Root workspace — Turborepo dev dependency
├── pnpm-workspace.yaml       # Declares apps/* and packages/* as workspaces
├── turbo.json                # Turborepo pipeline config (dev, build, lint)
├── docker-compose.yml        # Local dev: Postgres 16 + Redis 7
├── run.ps1                   # PowerShell task runner (replaces Makefile)
├── .env.example              # All environment variables with defaults
├── .gitignore
│
├── apps/
│   ├── api/                  # Go REST API service ──────────── :8080
│   │   ├── main.go           #   Entrypoint: config → DB → Redis → router → serve
│   │   ├── go.mod            #   Go module (independent, doesn't import realtime)
│   │   ├── go.sum
│   │   ├── Dockerfile        #   Multi-stage: Go 1.22 builder → distroless runtime
│   │   └── internal/
│   │       ├── config/
│   │       │   └── config.go       # Loads env vars: PORT, DATABASE_URL, REDIS_URL, JWT_SECRET
│   │       ├── db/
│   │       │   ├── postgres.go     # GORM connection pool (gorm.io/driver/postgres)
│   │       │   └── redis.go        # go-redis client init + ping
│   │       ├── handler/
│   │       │   ├── health.go       # GET /healthz — returns JSON {service, status, uptime}
│   │       │   ├── session.go      # (stub) Session CRUD handlers
│   │       │   ├── poll.go         # (stub) Poll CRUD + vote handlers
│   │       │   └── qa.go           # (stub) Q&A submission + moderation handlers
│   │       ├── middleware/
│   │       │   ├── logger.go       # slog JSON structured logging (method, path, status, latency)
│   │       │   └── auth.go         # (stub) JWT validation middleware
│   │       ├── models/
│   │       │   └── models.go       # GORM model structs for all 7 tables
│   │       ├── router/
│   │       │   └── router.go       # Chi mux: RequestID → RealIP → Logger → Recoverer → /healthz
│   │       └── service/
│   │           └── service.go      # (stub) Business logic layer
│   │
│   ├── realtime/             # Go WebSocket hub ────────────── :8081
│   │   ├── main.go           #   Entrypoint: config → mux → /healthz → serve
│   │   ├── go.mod            #   Go module (independent, doesn't import api)
│   │   ├── Dockerfile        #   Multi-stage: Go 1.22 builder → distroless runtime
│   │   └── internal/
│   │       ├── config/
│   │       │   └── config.go       # Loads env vars: PORT, REDIS_URL
│   │       ├── handler/
│   │       │   ├── health.go       # GET /healthz — returns JSON {service, status, uptime}
│   │       │   └── ws.go           # (stub) WebSocket upgrade handler /ws/:code
│   │       ├── hub/
│   │       │   ├── hub.go          # (stub) Central hub: rooms map, register/unregister
│   │       │   ├── client.go       # (stub) Per-client read/write pump goroutines
│   │       │   └── room.go         # (stub) Room = set of clients for one session
│   │       ├── pubsub/
│   │       │   └── redis.go        # (stub) Redis SUBSCRIBE to session:{code} channels
│   │       └── message/
│   │           └── types.go        # Event type constants: vote_update, new_question, etc.
│   │
│   └── web/                  # Next.js frontend ────────────── :3000
│       ├── package.json      #   Next.js 16, React 19, Tailwind 4
│       ├── next.config.ts
│       ├── tsconfig.json     #   Path alias: @/* → ./*
│       ├── Dockerfile        #   Multi-stage: node:20 builder → runner
│       ├── app/
│       │   ├── layout.tsx          # Root layout (Geist fonts, global CSS)
│       │   ├── page.tsx            # Landing page
│       │   ├── globals.css         # Tailwind imports
│       │   ├── (auth)/
│       │   │   └── login/
│       │   │       └── page.tsx    # Host login placeholder
│       │   ├── dashboard/
│       │   │   └── page.tsx        # Host dashboard placeholder
│       │   └── session/
│       │       └── [code]/
│       │           └── page.tsx    # Audience join view (dynamic route)
│       ├── components/
│       │   ├── poll/               # Poll-related components (empty)
│       │   ├── qa/                 # Q&A components (empty)
│       │   ├── session/            # Session components (empty)
│       │   └── ui/                 # Shared UI primitives (empty)
│       ├── hooks/                  # Custom hooks: useWebSocket, etc. (empty)
│       └── lib/                    # Utilities: api.ts, auth.ts, ws.ts (empty)
│
├── packages/
│   └── types/                # Shared TypeScript types
│       ├── package.json      #   @livepulse/types — importable by web app
│       └── src/
│           ├── index.ts            # Barrel export
│           ├── session.ts          # Session interface
│           ├── poll.ts             # Poll, PollOption, PollOptionWithCount
│           ├── qa.ts               # QAEntry interface
│           └── ws-events.ts        # WSMessage<T>, event payload types
│
├── db/
│   └── migrations/           # golang-migrate SQL files (7 tables)
│       ├── 000001_create_users.up.sql / .down.sql
│       ├── 000002_create_sessions.up.sql / .down.sql
│       ├── 000003_create_polls.up.sql / .down.sql
│       ├── 000004_create_poll_options.up.sql / .down.sql
│       ├── 000005_create_votes.up.sql / .down.sql
│       ├── 000006_create_qa_entries.up.sql / .down.sql
│       └── 000007_create_qa_votes.up.sql / .down.sql
│
└── infra/
    ├── docker-compose.prod.yml     # Production stack (to be configured)
    └── deploy/                     # Cloud provider configs (to be configured)
```

---

## Prerequisites

Make sure you have these installed before running the project:

| Tool               | Version  | Install                                                             |
| ------------------ | -------- | ------------------------------------------------------------------- |
| **Node.js**        | >= 20    | https://nodejs.org                                                  |
| **pnpm**           | >= 9     | `npm install -g pnpm`                                               |
| **Go**             | >= 1.22  | https://go.dev/dl                                                   |
| **Docker Desktop** | latest   | https://www.docker.com/products/docker-desktop                      |
| **golang-migrate** | latest   | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |

> **Windows users:** All commands use PowerShell. The task runner is `run.ps1` — no `make` required.

---

## Local Development Setup

### 1. Clone the repo

```powershell
git clone https://github.com/Eahtasham/live-pulse.git
cd live-pulse
```

### 2. Copy environment variables

```powershell
cp .env.example .env
```

The defaults work out of the box for local development. No changes needed unless you want to customize ports.

### 3. Install Node.js dependencies

```powershell
pnpm install
```

### 4. Start Postgres & Redis

```powershell
.\run.ps1 docker-up
```

Verify both containers are healthy:

```powershell
.\run.ps1 docker-ps
```

You should see both `livepulse-postgres` and `livepulse-redis` with status `(healthy)`.

### 5. Run database migrations

```powershell
.\run.ps1 migrate-up
```

This applies all 7 migration files and creates the tables: `users`, `sessions`, `polls`, `poll_options`, `votes`, `qa_entries`, `qa_votes`.

### 6. Start individual services

**API service** (needs Postgres + Redis running):
```powershell
.\run.ps1 api
# → http://localhost:8080/healthz
```

**Realtime service** (in a separate terminal):
```powershell
.\run.ps1 realtime
# → http://localhost:8081/healthz
```

**Next.js frontend** (in a separate terminal):
```powershell
.\run.ps1 web
# → http://localhost:3000
```

### 7. Verify everything works

```powershell
# Health check — API
curl http://localhost:8080/healthz
# → {"service":"api","status":"ok","uptime":"5s"}

# Health check — Realtime
curl http://localhost:8081/healthz
# → {"service":"realtime","status":"ok","uptime":"3s"}

# Frontend
# Open http://localhost:3000 in browser
```

### Quick Setup (all-in-one)

If you want to do steps 3–5 in one command:

```powershell
.\run.ps1 setup
```

---

## Task Runner Reference

All commands are run from the repo root using `.\run.ps1 <command>`:

| Command            | Description                              |
| ------------------ | ---------------------------------------- |
| **Docker**         |                                          |
| `docker-up`        | Start Postgres + Redis containers        |
| `docker-down`      | Stop and remove containers               |
| `docker-ps`        | Show container status                    |
| **Database**       |                                          |
| `migrate-up`       | Run all pending migrations               |
| `migrate-down`     | Rollback the last migration              |
| `migrate-status`   | Show current migration version           |
| **Services**       |                                          |
| `api`              | Run the Go API service (:8080)           |
| `realtime`         | Run the Go Realtime service (:8081)      |
| `web`              | Run the Next.js dev server (:3000)       |
| `api-build`        | Compile API to `apps/api/bin/api.exe`    |
| `realtime-build`   | Compile Realtime to `apps/realtime/bin/` |
| **Turbo**          |                                          |
| `dev`              | Start all services via Turborepo         |
| `build`            | Build all apps                           |
| `lint`             | Lint all apps                            |
| **Setup**          |                                          |
| `setup`            | First-time setup (install + docker + migrate) |

---

## Backend Deep Dive

### API Service (Go) — `apps/api/`

The API service is the **primary backend**. It handles all REST endpoints, business logic, database writes, and publishes events to Redis for the realtime service to broadcast.

**Module:** `github.com/Eahtasham/live-pulse/apps/api`

**Key dependencies:**
- `go-chi/chi/v5` — Lightweight HTTP router with middleware support
- `gorm.io/gorm` + `gorm.io/driver/postgres` — ORM for PostgreSQL
- `redis/go-redis/v9` — Redis client
- `google/uuid` — UUID generation for primary keys

**Startup flow** (`main.go`):
1. Load config from env vars (with sensible defaults)
2. Initialize structured JSON logger (`slog`)
3. Connect to PostgreSQL via GORM
4. Connect to Redis
5. Build Chi router with middleware stack
6. Start HTTP server on `:8080` with graceful shutdown

**Middleware stack** (`router/router.go`):
```
Request → RequestID → RealIP → Logger (slog JSON) → Recoverer → Handler
```

Every HTTP request is logged as structured JSON:
```json
{
  "level": "INFO",
  "msg": "http request",
  "method": "GET",
  "path": "/healthz",
  "status": 200,
  "latency_ms": 0,
  "bytes": 52,
  "request_id": "abc123"
}
```

**Package layout:**

| Package      | Responsibility                                                    |
| ------------ | ----------------------------------------------------------------- |
| `config`     | Load and validate environment variables                           |
| `db`         | Database and Redis connection initialization                      |
| `handler`    | Thin HTTP handlers — parse request, call service, write response  |
| `service`    | Business logic layer (to be implemented)                          |
| `models`     | GORM struct definitions for all database tables                   |
| `middleware` | Request logging (slog), JWT auth (stub)                           |
| `router`     | Chi mux setup and route registration                              |

### Realtime Service (Go) — `apps/realtime/`

The realtime service is a **WebSocket hub**. It subscribes to Redis Pub/Sub channels and broadcasts events to connected browser clients. It has **no direct connection to PostgreSQL** and **never calls the API service over HTTP**.

**Module:** `github.com/Eahtasham/live-pulse/apps/realtime`

**Startup flow** (`main.go`):
1. Load config (PORT, REDIS_URL)
2. Initialize structured JSON logger
3. Register `/healthz` route
4. Start HTTP server on `:8081` with graceful shutdown

**Package layout:**

| Package   | Responsibility                                                |
| --------- | ------------------------------------------------------------- |
| `config`  | Load env vars                                                 |
| `handler` | HTTP handlers — healthz + WebSocket upgrade (stub)            |
| `hub`     | Central hub managing rooms and clients (stub)                 |
| `pubsub`  | Redis SUBSCRIBE to `session:{code}` channels (stub)           |
| `message` | Event type constants (`vote_update`, `new_question`, etc.)    |

**Important:** Both Go services have their own `go.mod` — they are **independent Go modules** and do not import each other. Communication happens exclusively through Redis.

### Database Schema

7 tables managed via golang-migrate SQL migrations in `db/migrations/`:

```
┌──────────┐     ┌───────────┐     ┌──────────────┐     ┌─────────┐
│  users   │◄────│ sessions  │◄────│    polls     │◄────│ votes   │
│          │     │           │     │              │     │         │
│ id (PK)  │     │ id (PK)   │     │ id (PK)      │     │ id (PK) │
│ email    │     │ host_id → │     │ session_id → │     │ poll_id →│
│ name     │     │ code (6)  │     │ question     │     │ option →│
│ provider │     │ title     │     │ answer_mode  │     │ uid    │
│ avatar   │     │ status    │     │ time_limit   │     └─────────┘
└──────────┘     │ closed_at │     │ status       │
                 └───────────┘     └──────┬───────┘
                       │                  │
                       │           ┌──────┴───────┐
                       │           │ poll_options  │
                       │           │ id, label,    │
                       │           │ position      │
                       │           └──────────────┘
                       │
                 ┌─────┴──────┐     ┌──────────┐
                 │ qa_entries  │◄────│ qa_votes │
                 │ id (PK)    │     │ id (PK)  │
                 │ session_id→│     │ entry_id→│
                 │ entry_type │     │ voter_uid│
                 │ body       │     │ vote_val │
                 │ score      │     └──────────┘
                 │ status     │
                 │ is_hidden  │
                 └────────────┘
```

**Key design decisions:**
- **UUIDs** as primary keys (generated via `gen_random_uuid()` in Postgres)
- **TEXT with CHECK constraints** instead of PostgreSQL ENUM types — easier to extend in future migrations
- **ON DELETE CASCADE** on all child foreign keys — cleanup cascades when a parent is deleted
- **ON DELETE SET NULL** on `sessions.host_id` — deleting a user doesn't delete their sessions
- **`is_hidden`** as a separate boolean on `qa_entries` — orthogonal to `status`, so a question can be `answered` AND hidden simultaneously
- **`qa_votes`** table with UNIQUE constraint on `(qa_entry_id, voter_uid)` — prevents double-voting
- **`updated_at`** on mutable tables (`sessions`, `polls`, `qa_entries`) — tracks state changes
- The `votes` table has a UNIQUE constraint on `(poll_id, audience_uid, option_id)` — enforces one vote per poll per user

### GORM Models

All models are defined in `apps/api/internal/models/models.go`. Each struct maps to a database table with GORM tags for column types, constraints, and defaults. UUID primary keys are auto-generated via a `BeforeCreate` hook.

Models defined: `User`, `Session`, `Poll`, `PollOption`, `Vote`, `QAEntry`, `QAVote`

> **Note:** Database tables are created via golang-migrate SQL files, NOT via GORM's AutoMigrate. This gives us version-controlled, reversible migrations with proper up/down scripts.

---

## Frontend

The Next.js app lives in `apps/web/` and uses the **App Router** pattern (no `src/` directory).

| Route                  | Purpose                        |
| ---------------------- | ------------------------------ |
| `/`                    | Landing page                   |
| `/login`               | Host authentication (OAuth)    |
| `/dashboard`           | Host session management        |
| `/session/[code]`      | Audience join view (dynamic)   |

**Path alias:** `@/*` maps to the `apps/web/` root (configured in `tsconfig.json`).

**Component directories** (empty, to be built):
- `components/poll/` — Poll creation, voting UI, result charts
- `components/qa/` — Question/comment submission, feed, moderation
- `components/session/` — Session creation, join, code display
- `components/ui/` — Shared buttons, inputs, cards, modals
- `hooks/` — `useWebSocket`, `usePollVotes`, `useQAFeed`
- `lib/` — `api.ts` (REST client), `auth.ts` (Auth.js config), `ws.ts` (WebSocket wrapper)

---

## Shared Types

The `packages/types/` package (`@livepulse/types`) contains TypeScript interfaces shared between the frontend and any future TS-based services.

| File            | Exports                                                  |
| --------------- | -------------------------------------------------------- |
| `session.ts`    | `Session` interface                                      |
| `poll.ts`       | `Poll`, `PollOption`, `PollOptionWithCount`              |
| `qa.ts`         | `QAEntry` interface                                      |
| `ws-events.ts`  | `WSMessage<T>`, `VoteUpdatePayload`, `NewQuestionPayload`, `NewCommentPayload`, `QAUpdatePayload`, `SessionClosedPayload` |

Import in the web app:
```typescript
import type { Session, Poll } from "@livepulse/types";
```

---

## Docker & Infrastructure

### Local Development

`docker-compose.yml` at the repo root runs:

| Service    | Image              | Port  | Healthcheck               | Volume       |
| ---------- | ------------------ | ----- | ------------------------- | ------------ |
| PostgreSQL | `postgres:16-alpine` | 5432  | `pg_isready`             | `pgdata`     |
| Redis      | `redis:7-alpine`   | 6379  | `redis-cli ping`          | `redisdata`  |

Both containers are on the `livepulse-net` bridge network.

Redis runs with `--appendonly yes` for AOF persistence.

### Dockerfiles (for deployment)

Each app has a multi-stage Dockerfile:

- **Go services** (`apps/api/Dockerfile`, `apps/realtime/Dockerfile`):
  - Build stage: `golang:1.22-alpine` → compile static binary
  - Runtime stage: `gcr.io/distroless/static-debian12` (tiny, no shell, secure)

- **Next.js** (`apps/web/Dockerfile`):
  - Dependencies stage: `node:20-alpine` → `pnpm install --frozen-lockfile`
  - Build stage: `pnpm run build`
  - Runtime stage: `node:20-alpine` → `node server.js` (standalone output)

Build images:
```powershell
docker build -t livepulse/api:latest ./apps/api
docker build -t livepulse/realtime:latest ./apps/realtime
docker build -t livepulse/web:latest ./apps/web
```

---

## Environment Variables

Copy `.env.example` to `.env` and fill in the values:

| Variable                | Service         | Description                          | Default                     |
| ----------------------- | --------------- | ------------------------------------ | --------------------------- |
| `POSTGRES_USER`         | Docker/DB       | Postgres username                    | `livepulse`                 |
| `POSTGRES_PASSWORD`     | Docker/DB       | Postgres password                    | `livepulse_dev`             |
| `POSTGRES_DB`           | Docker/DB       | Postgres database name               | `livepulse`                 |
| `DATABASE_URL`          | api             | Full PostgreSQL DSN                  | `postgres://livepulse:...`  |
| `REDIS_URL`             | api, realtime   | Redis connection string              | `redis://localhost:6379/0`  |
| `API_PORT`              | api             | API service port                     | `8080`                      |
| `JWT_SECRET`            | api             | Secret key for JWT signing           | `change-me-...`             |
| `REALTIME_PORT`         | realtime        | WebSocket service port               | `8081`                      |
| `NEXTAUTH_URL`          | web             | Next.js base URL                     | `http://localhost:3000`     |
| `NEXTAUTH_SECRET`       | web             | Auth.js session encryption key       | `change-me-...`             |
| `GOOGLE_CLIENT_ID`      | web             | Google OAuth Client ID               | (empty)                     |
| `GOOGLE_CLIENT_SECRET`  | web             | Google OAuth Client Secret           | (empty)                     |
| `S3_BUCKET`             | api             | AWS S3 bucket for exports            | (empty)                     |
| `S3_REGION`             | api             | AWS S3 region                        | (empty)                     |
| `AWS_ACCESS_KEY_ID`     | api             | AWS credentials                      | (empty)                     |
| `AWS_SECRET_ACCESS_KEY` | api             | AWS credentials                      | (empty)                     |

> **Security:** Never commit `.env` to git. Only `.env.example` is tracked.

---

## WebSocket Event System

Clients connect to `ws://localhost:8081/ws/:sessionCode`. Events are JSON-encoded:

| Event Type       | Direction        | Payload                                    | Trigger               |
| ---------------- | ---------------- | ------------------------------------------ | --------------------- |
| `vote_update`    | Server → Client  | `{pollId, options: [{id, count}]}`         | Vote written to DB    |
| `new_question`   | Server → Client  | `{id, body, score, status}`                | Question submitted    |
| `new_comment`    | Server → Client  | `{id, body, authorUid}`                    | Comment submitted     |
| `qa_update`      | Server → Client  | `{id, status, score}`                      | Host moderates entry  |
| `session_closed` | Server → Client  | `{code, closedAt}`                         | Host closes session   |
| `ping`           | Client → Server  | `{}`                                       | Keepalive (every 30s) |

Redis Pub/Sub channel pattern: `session:{code}` (e.g., `session:A1B2C3`)

---

## API Endpoints

All endpoints served by the Go API on port `8080`. Base path: `/v1`

### Sessions

| Method  | Path                            | Auth     | Description                      |
| ------- | ------------------------------- | -------- | -------------------------------- |
| `POST`  | `/v1/sessions`                  | Required | Create a new session             |
| `GET`   | `/v1/sessions/:code`            | Public   | Get session details by code      |
| `PATCH` | `/v1/sessions/:code/close`      | Required | Close session, archive Q&A       |
| `GET`   | `/v1/sessions/:code/export`     | Required | Export results (S3 presigned URL)|

### Polls

| Method  | Path                                    | Auth     | Description                |
| ------- | --------------------------------------- | -------- | -------------------------- |
| `POST`  | `/v1/sessions/:code/polls`              | Required | Create poll for session    |
| `GET`   | `/v1/sessions/:code/polls`              | Public   | List polls in session      |
| `GET`   | `/v1/sessions/:code/polls/:id`          | Public   | Get poll + live counts     |
| `PATCH` | `/v1/sessions/:code/polls/:id`          | Required | Update status              |
| `POST`  | `/v1/sessions/:code/polls/:id/vote`     | Public   | Submit vote (anon)         |

### Q&A

| Method  | Path                                    | Auth     | Description                |
| ------- | --------------------------------------- | -------- | -------------------------- |
| `POST`  | `/v1/sessions/:code/qa`                 | Public   | Submit question or comment |
| `GET`   | `/v1/sessions/:code/qa`                 | Public   | List active Q&A (paginated)|
| `PATCH` | `/v1/sessions/:code/qa/:id`             | Required | Moderate (pin/answer/hide) |
| `POST`  | `/v1/sessions/:code/qa/:id/vote`        | Public   | Upvote or downvote         |

### Health

| Method | Path       | Auth   | Description                     |
| ------ | ---------- | ------ | ------------------------------- |
| `GET`  | `/healthz` | Public | Liveness probe — JSON status    |

> **Note:** Only `/healthz` is currently implemented. All other endpoints are planned — handler stubs exist in the codebase.

---

## Troubleshooting

### Docker containers won't start
```powershell
# Check if ports are already in use
netstat -ano | findstr "5432"
netstat -ano | findstr "6379"

# Reset everything
.\run.ps1 docker-down
docker volume rm live-pulse_pgdata live-pulse_redisdata
.\run.ps1 docker-up
```

### API service exits immediately
The API needs both Postgres and Redis running. Start Docker first:
```powershell
.\run.ps1 docker-up
# Wait for healthy status
.\run.ps1 docker-ps
# Then start API
.\run.ps1 api
```

### Migration errors
```powershell
# Check current version
.\run.ps1 migrate-status

# If stuck in a dirty state, force a version
migrate -path db/migrations -database "postgres://livepulse:livepulse_dev@localhost:5432/livepulse?sslmode=disable" force <version_number>
```

### golang-migrate not found
```powershell
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```
Make sure `$GOPATH/bin` (usually `%USERPROFILE%\go\bin`) is in your system PATH.

### PowerShell execution policy
If `.\run.ps1` fails with "cannot be loaded because running scripts is disabled":
```powershell
Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned
```

---

## License

Internal project — Cloud Computing Mini Project.

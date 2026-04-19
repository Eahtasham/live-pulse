# LivePulse — API Service

The Go REST API powering LivePulse. Handles authentication, sessions, polls, votes, Q&A, and publishes real-time events to Redis Pub/Sub.

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Handlers   │────▶│   Services   │────▶│  PostgreSQL  │
│  (chi router)│     │ (biz logic)  │     │   (GORM)     │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐     ┌──────────────┐
                     │  Publisher   │────▶│    Redis     │
                     │ (Pub/Sub)   │     │  (go-redis)  │
                     └──────────────┘     └──────────────┘
```

## Directory Structure

```
apps/api/
├── main.go                      # Entry point, wires dependencies
├── docs/                        # Swagger auto-generated docs
├── internal/
│   ├── config/config.go         # Environment-based configuration
│   ├── db/
│   │   ├── postgres.go          # GORM PostgreSQL connection
│   │   └── redis.go             # go-redis client factory
│   ├── handler/                  # HTTP handlers (one file per domain)
│   │   ├── auth.go
│   │   ├── health.go
│   │   ├── poll.go
│   │   ├── qa.go
│   │   ├── qavote.go
│   │   ├── session.go
│   │   └── vote.go
│   ├── middleware/
│   │   ├── jwt.go               # JWT auth middleware
│   │   └── logger.go            # Structured request logging
│   ├── models/                   # GORM models
│   │   ├── poll.go              # Poll, PollOption, Vote
│   │   ├── qa.go                # QAEntry, QAVote
│   │   ├── session.go
│   │   └── user.go
│   ├── router/router.go         # Route definitions + middleware stack
│   └── service/                  # Business logic layer
│       ├── auth.go
│       ├── poll.go
│       ├── publisher.go         # ★ Redis Pub/Sub event publisher
│       ├── qa.go
│       ├── qavote.go
│       ├── session.go
│       └── vote.go
```

## Environment Variables

| Variable       | Default                                                              | Description                 |
| -------------- | -------------------------------------------------------------------- | --------------------------- |
| `API_PORT`     | `8080`                                                               | HTTP listen port            |
| `DATABASE_URL` | `postgres://livepulse:livepulse_dev@localhost:5432/livepulse?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL`    | `redis://localhost:6379/0`                                           | Redis connection string     |
| `JWT_SECRET`   | `dev-secret-change-me`                                               | JWT signing secret          |
| `JWT_EXPIRY`   | `24h`                                                                | JWT token expiry duration   |

## Running

```bash
# From repo root
cd apps/api
go run main.go

# Or with Docker
docker compose up api
```

Swagger UI available at `http://localhost:8080/swagger/index.html`.

## API Endpoints

### Auth
| Method | Path                  | Auth | Description              |
| ------ | --------------------- | ---- | ------------------------ |
| POST   | `/v1/auth/register`   | No   | Register with email      |
| POST   | `/v1/auth/login`      | No   | Login with email         |
| POST   | `/v1/auth/callback`   | No   | OAuth callback           |

### Sessions
| Method | Path                       | Auth | Description              |
| ------ | -------------------------- | ---- | ------------------------ |
| POST   | `/v1/sessions`             | JWT  | Create session           |
| GET    | `/v1/sessions`             | JWT  | List host's sessions     |
| GET    | `/v1/sessions/{code}`      | No   | Get session by code      |
| POST   | `/v1/sessions/{code}/join` | No   | Join as audience         |

### Polls
| Method | Path                                        | Auth     | Description         |
| ------ | ------------------------------------------- | -------- | ------------------- |
| POST   | `/v1/sessions/{code}/polls`                 | JWT      | Create poll         |
| GET    | `/v1/sessions/{code}/polls`                 | Optional | List polls          |
| GET    | `/v1/sessions/{code}/polls/{pollID}`        | Optional | Get poll            |
| PATCH  | `/v1/sessions/{code}/polls/{pollID}`        | JWT      | Update poll status  |
| POST   | `/v1/sessions/{code}/polls/{pollID}/vote`   | No*      | Cast vote           |

### Q&A
| Method | Path                                       | Auth | Description         |
| ------ | ------------------------------------------ | ---- | ------------------- |
| GET    | `/v1/sessions/{code}/qa`                   | No   | List Q&A entries    |
| POST   | `/v1/sessions/{code}/qa`                   | No*  | Submit question     |
| PATCH  | `/v1/sessions/{code}/qa/{id}`              | JWT  | Moderate entry      |
| POST   | `/v1/sessions/{code}/qa/{id}/vote`         | No*  | Upvote/downvote     |

*Requires `X-Audience-UID` header (obtained from `/join`).

---

## Redis Pub/Sub — Publisher

The API service acts as the **publisher** in the Redis Pub/Sub architecture. After every successful database write that affects real-time state, the service publishes a JSON event to the Redis channel `session:{CODE}`.

### How It Works

1. A `Publisher` struct wraps the Redis client (`service/publisher.go`)
2. It's injected into `VoteService`, `QAService`, and `QAVoteService` at startup
3. After each successful DB transaction, the service publishes an event
4. Publishing is **fire-and-forget** — failures are logged but don't fail the HTTP response

### Channel Format

```
session:{SESSION_CODE}
```

Example: `session:A1B2C3`

One channel per active session. The realtime service subscribes to these channels per-room.

### Event Types & Payloads

#### `vote_update` — Cast vote

Published after a vote is inserted. Contains **ALL options with current total counts** (not deltas).

```json
{
  "type": "vote_update",
  "payload": {
    "pollId": "189e5856-4773-419d-80f5-5515cab2731c",
    "options": [
      { "id": "uuid-1", "label": "Option A", "vote_count": 42 },
      { "id": "uuid-2", "label": "Option B", "vote_count": 37 }
    ]
  }
}
```

**Why full state?** The frontend receives the complete snapshot and re-renders the chart from scratch. This prevents state accumulation bugs where a missed delta message causes the UI to be permanently out of sync.

#### `new_question` — Question submitted

```json
{
  "type": "new_question",
  "payload": {
    "id": "uuid",
    "entry_type": "question",
    "body": "What's the roadmap for Q3?",
    "score": 0,
    "author_uid": "audience-uid",
    "created_at": "2026-04-15T12:00:00Z"
  }
}
```

#### `new_comment` — Comment submitted

```json
{
  "type": "new_comment",
  "payload": {
    "id": "uuid",
    "entry_type": "comment",
    "body": "Great presentation!",
    "author_uid": "audience-uid",
    "created_at": "2026-04-15T12:00:00Z"
  }
}
```

#### `qa_update` — Moderation or Q&A vote

Published when host pins/hides/archives an entry OR when an audience member upvotes/downvotes.

```json
{
  "type": "qa_update",
  "payload": {
    "id": "uuid",
    "status": "pinned",
    "is_hidden": false,
    "score": 15
  }
}
```

#### `session_closed` — Session closed

```json
{
  "type": "session_closed",
  "payload": {
    "code": "A1B2C3",
    "closed_at": "2026-04-15T14:00:00Z"
  }
}
```

### Publish Trigger Points

| Service          | Method          | Event Published     | Timing                        |
| ---------------- | --------------- | ------------------- | ----------------------------- |
| `VoteService`    | `CastVote`      | `vote_update`       | After vote insert (goroutine) |
| `QAService`      | `CreateEntry`   | `new_question` / `new_comment` | After entry insert |
| `QAService`      | `ModerateEntry` | `qa_update`         | After entry update            |
| `QAVoteService`  | `CastVote`      | `qa_update`         | After each commit             |

### Important Implementation Details

- **`vote_update` runs in a goroutine** with `context.Background()` — not the HTTP request context, since the request context is canceled when the response is sent.
- **All other publishes are synchronous** but non-blocking — Redis PUBLISH is fast (~1ms).
- **Nil-safe**: If `Publisher` is nil (e.g., in tests), publish calls are skipped.

### Verifying with redis-cli

```bash
# Monitor all Redis commands
redis-cli MONITOR

# Subscribe to a specific session channel
redis-cli SUBSCRIBE session:A1B2C3
```

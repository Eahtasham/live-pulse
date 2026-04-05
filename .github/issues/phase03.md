## Phase 3 — Session API 🟢 BACKEND

**Goal:** Full CRUD for sessions. Hosts create sessions, audiences join with a 6-character code and receive an ephemeral identity.

**Prerequisite:** Phase 2

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/sessions` | Host JWT | Create a new session with a unique 6-char code |
| `GET` | `/v1/sessions` | Host JWT | List all sessions for the authenticated host |
| `GET` | `/v1/sessions/:code` | Public | Get session details by code |
| `POST` | `/v1/sessions/:code/join` | Public | Join as audience — issues ephemeral UID stored in Redis |

### Session code generation

- 6-character alphanumeric string (A-Z, 0-9), uppercase only
- Generated server-side using `crypto/rand` (not `math/rand`)
- Retry on collision (check DB uniqueness before returning)
- Example codes: `A1B2C3`, `X9Y8Z7`

### Audience join flow

1. `POST /v1/sessions/:code/join` is called by the frontend when an audience member opens `/session/:code`
2. If the request includes a client fingerprint (e.g., `X-Client-ID` header or cookie), check Redis for an existing UID for that client+session
3. If found, return the existing UID (idempotent)
4. If not found, generate a new UUID v4, store in Redis as `audience:{code}:{uid}` with a 24-hour TTL, and return it
5. The UID is used in all subsequent vote and Q&A submissions

### Request/Response examples

**Create session:**
```json
// POST /v1/sessions
// Request:
{"title": "CS101 Lecture 5"}

// Response (201):
{"id": "uuid", "code": "A1B2C3", "title": "CS101 Lecture 5", "status": "active", "created_at": "..."}
```

**Join session:**
```json
// POST /v1/sessions/A1B2C3/join
// Response (200):
{"audience_uid": "uuid-v4", "session_title": "CS101 Lecture 5"}
```

### Acceptance tests

- [ ] `POST /v1/sessions` with `{title: "CS101 Lecture"}` → returns `{id, code, title, status: "active"}`
- [ ] The `code` is exactly 6 alphanumeric characters (uppercase + digits)
- [ ] Creating 100 sessions produces 100 unique codes (no collisions)
- [ ] `GET /v1/sessions/A1B2C3` → returns session details (no auth needed)
- [ ] `GET /v1/sessions/XXXXXX` (invalid code) → 404
- [ ] `GET /v1/sessions` with auth → returns list of sessions for that host, sorted by `created_at DESC`
- [ ] `GET /v1/sessions` without auth → 401
- [ ] `POST /v1/sessions/:code/join` → returns `{audience_uid: "uuid"}` and creates a Redis key `audience:{code}:{uid}` with TTL
- [ ] Calling join again with the same client identifier → returns the SAME uid (idempotent)
- [ ] A different client joining → gets a DIFFERENT uid
- [ ] The Redis TTL is set (e.g., 24 hours) — verify with `redis-cli TTL audience:{code}:{uid}`
- [ ] Creating a session without auth → 401

### Files to create/modify

- `apps/api/internal/handler/session.go` — Session CRUD handlers
- `apps/api/internal/service/session.go` — Session business logic (code generation, Redis ops)
- `apps/api/internal/router/router.go` — Register session routes

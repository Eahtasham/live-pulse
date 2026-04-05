## Phase 10 — Session Lifecycle API 🟢 BACKEND

**Goal:** Host can close a session. Closing archives Q&A, blocks new actions, broadcasts `session_closed` to all connected clients, and cleans up resources.

**Prerequisite:** Phase 9

### Endpoint

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `PATCH` | `/v1/sessions/:code/close` | Host JWT | Close/archive a session |

### Close session flow

1. Validate the caller is the session host (JWT `user_id` == `sessions.host_id`)
2. Validate the session is currently `active` — return 400 if already archived
3. **Transaction:**
   - Set `sessions.status = 'archived'` and `sessions.closed_at = NOW()`
   - Bulk-update all `qa_entries` for this session: `status = 'archived'`
4. Publish `session_closed` event to Redis channel `session:{code}`
5. Return updated session object

### What the `session_closed` event triggers (in the Realtime service)

- Hub broadcasts `session_closed` to all WebSocket clients in the room
- After broadcast, the hub forcefully closes all WebSocket connections for that room (with close code `1000` — normal closure)
- Room is removed from the hub

### Post-close behavior (enforced on all endpoints)

Every endpoint that writes data must check `sessions.status`:
- `POST /v1/sessions/:code/polls` → 400 "Session is archived"
- `PATCH /v1/sessions/:code/polls/:id` → 400 "Session is archived"
- `POST /v1/sessions/:code/polls/:id/vote` → 400 "Session is archived"
- `POST /v1/sessions/:code/qa` → 400 "Session is archived"
- `POST /v1/sessions/:code/qa/:id/vote` → 400 "Session is archived"

Read-only endpoints continue working:
- `GET /v1/sessions/:code` → returns session with `status: "archived"`
- `GET /v1/sessions/:code/polls` → returns polls with final vote counts
- `GET /v1/sessions/:code/qa` → returns archived Q&A entries

### Ephemeral UID cleanup

- Audience Redis keys (`audience:{code}:*`) can be immediately deleted on session close, OR left to expire via their existing TTL
- Immediate cleanup is cleaner but TTL-based expiry is simpler to implement and equally safe

### Acceptance tests

- [ ] `PATCH /v1/sessions/:code/close` → `status = 'archived'`, `closed_at` is populated
- [ ] All `qa_entries` for the session are bulk-updated to `status = 'archived'`
- [ ] A `session_closed` event is published to Redis channel `session:{code}`
- [ ] WebSocket clients in the room receive the `session_closed` event
- [ ] After closing: `POST /v1/sessions/:code/polls` → 400 ("Session is archived")
- [ ] After closing: `POST /v1/sessions/:code/polls/:id/vote` → 400
- [ ] After closing: `POST /v1/sessions/:code/qa` → 400
- [ ] After closing: `GET /v1/sessions/:code` still works — returns archived session
- [ ] After closing: `GET /v1/sessions/:code/polls` still works — returns final vote counts
- [ ] WebSocket connections for the closed room are terminated server-side
- [ ] New WebSocket connections to a closed session are rejected
- [ ] Only the session host can close (other users → 403)
- [ ] Ephemeral audience Redis keys for this session are cleaned up (or left to expire)

### Files to create/modify

- `apps/api/internal/handler/session.go` — Add close session handler
- `apps/api/internal/service/session.go` — Close logic with transaction + PUBLISH
- `apps/api/internal/middleware/session_guard.go` — Middleware to check session status on write endpoints
- `apps/api/internal/router/router.go` — Register close route

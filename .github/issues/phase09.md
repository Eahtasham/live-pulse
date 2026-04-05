## Phase 9 — Redis Pub/Sub Bridge 🟢 BACKEND

**Goal:** Wire the API service (publisher) to the realtime service (subscriber). Votes, Q&A submissions, and moderation actions trigger real-time broadcasts to all room clients.

**Prerequisite:** Phase 5 + Phase 7 + Phase 8

### Architecture

```
┌───────────────────┐       PUBLISH         ┌──────────┐      SUBSCRIBE       ┌─────────────────────┐
│   Go API Service  │ ──────────────────────→│  Redis   │──────────────────────→│ Go Realtime Service │
│  (writes to DB)   │  channel: session:CODE │  Pub/Sub │  channel: session:*  │    (WebSocket Hub)  │
└───────────────────┘                        └──────────┘                       └─────────────────────┘
                                                                                         │
                                                                              broadcast to Room clients
```

### Publisher side (API service)

After each successful DB write, publish an event to Redis:

```go
// After vote insert:
rdb.Publish(ctx, "session:"+code, `{"type":"vote_update","payload":{...}}`)

// After Q&A submit:
rdb.Publish(ctx, "session:"+code, `{"type":"new_question","payload":{...}}`)

// After moderation action:
rdb.Publish(ctx, "session:"+code, `{"type":"qa_update","payload":{...}}`)
```

Create a `publisher` package/service in the API that wraps Redis PUBLISH with typed event constructors.

### Subscriber side (Realtime service)

- When the first client joins a room for session code `ABC123`, subscribe to Redis channel `session:ABC123`
- On each message received from Redis, broadcast the raw JSON to all WebSocket clients in that room
- When the last client leaves a room, unsubscribe from `session:ABC123`

### Event types and payloads

| Event | Trigger | Payload |
|-------|---------|---------|
| `vote_update` | Vote cast | `{pollId, options: [{id, label, vote_count}, ...]}` — ALL options with current counts |
| `new_question` | Question submitted | `{id, entry_type, body, score, author_uid, created_at}` |
| `new_comment` | Comment submitted | `{id, entry_type, body, author_uid, created_at}` |
| `qa_update` | Moderation action or Q&A vote | `{id, status, is_hidden, score}` |
| `session_closed` | Session closed | `{code, closed_at}` |

### Important: `vote_update` sends full state

The `vote_update` payload must include ALL options with their CURRENT total counts — not just a delta or the single option that was voted. This allows the frontend to fully re-render the chart from one message without local state accumulation bugs.

### Acceptance tests

- [ ] Cast a vote via API → `redis-cli MONITOR` shows `PUBLISH session:A1B2C3 {...}`
- [ ] WebSocket client in room `A1B2C3` receives the `vote_update` event within 1 second
- [ ] WebSocket client in a DIFFERENT room does NOT receive the event
- [ ] Submit a question → clients receive `new_question` event
- [ ] Submit a comment → clients receive `new_comment` event
- [ ] Host moderates (pin/hide/answer) → clients receive `qa_update` event
- [ ] Q&A upvote → clients receive `qa_update` with new score
- [ ] `vote_update` payload contains ALL options with current total counts (not just delta)
- [ ] If no clients are connected to a room, PUBLISH still succeeds (fire-and-forget)
- [ ] Realtime service subscribes to the channel when first client joins, unsubscribes when last client leaves
- [ ] 10 rapid votes → all 10 `vote_update` events are received by clients (no drops)

### Files to create/modify

**API service:**
- `apps/api/internal/service/publisher.go` — Redis PUBLISH wrapper with typed event constructors
- `apps/api/internal/handler/vote.go` — Add PUBLISH after vote insert
- `apps/api/internal/handler/qa.go` — Add PUBLISH after Q&A submit / moderation

**Realtime service:**
- `apps/realtime/internal/pubsub/redis.go` — Redis SUBSCRIBE/UNSUBSCRIBE management
- `apps/realtime/internal/hub/hub.go` — Wire pubsub messages into room broadcasts
- `apps/realtime/go.mod` — Add `github.com/redis/go-redis/v9` dependency

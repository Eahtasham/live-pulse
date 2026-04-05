## Phase 8 — WebSocket Hub & Rooms 🟢 BACKEND

**Goal:** The realtime service accepts WebSocket connections, organizes clients into rooms by session code, and manages client lifecycle with ping/pong keepalive.

**Prerequisite:** Phase 3

**Scope:** Realtime service only — no Redis Pub/Sub wiring yet. This phase builds the WebSocket infrastructure that Phase 9 will connect to Redis.

### Architecture

```
Client A ──ws──┐
Client B ──ws──┤──→ Hub ──→ Room("A1B2C3") ──→ [Client A, Client B, Client C]
Client C ──ws──┘          → Room("X9Y8Z7") ──→ [Client D]
Client D ──ws─────────────┘
```

### Components to build

**Hub** (`internal/hub/hub.go`):
- Singleton goroutine managing all rooms
- `map[string]*Room` keyed by session code
- Channels: `register`, `unregister`, `broadcast`
- Creates a room when the first client for a code connects
- Destroys a room when the last client disconnects

**Room** (`internal/hub/room.go`):
- Session code + set of connected clients
- `broadcast(message []byte)` sends to all clients in the room
- Tracks client count for logging

**Client** (`internal/hub/client.go`):
- Per-connection struct with a `send` channel (buffered, e.g., 256 messages)
- Two goroutines per client:
  - **readPump**: reads messages from WebSocket, handles ping/pong
  - **writePump**: drains `send` channel, writes to WebSocket
- Ping interval: 30 seconds
- Pong deadline: 60 seconds (disconnect if no pong received)

**WebSocket Handler** (`internal/handler/ws.go`):
- `GET /ws/:code` — upgrade to WebSocket
- Validate that session code exists via API health check or Redis lookup
- Reject connections to invalid or archived sessions with close code `4404`
- Register client with Hub on connect, unregister on disconnect

### Connection lifecycle

1. Client sends `GET /ws/A1B2C3` with `Upgrade: websocket` header
2. Server validates session exists
3. Server upgrades connection and creates a `Client` struct
4. Client is registered with the Hub → added to Room `A1B2C3`
5. readPump and writePump goroutines start
6. On disconnect (browser close, network drop, pong timeout): unregister from Hub, close WebSocket

### Acceptance tests

- [ ] WebSocket connection to `/ws/A1B2C3` (valid session) → accepted, stays open
- [ ] WebSocket connection to `/ws/XXXXXX` (invalid session) → rejected with close code
- [ ] Connect 3 clients to `/ws/A1B2C3` → room has 3 clients (log or internal counter)
- [ ] Disconnect 1 → room has 2
- [ ] Disconnect all → room is cleaned up (no memory leak)
- [ ] Client that stops sending pings → disconnected after timeout
- [ ] Connecting to a closed/archived session → rejected
- [ ] Connect/disconnect events are logged with session code and client count
- [ ] Sending an invalid message (not JSON) → client receives an error frame, not a server crash
- [ ] Server can handle 100 simultaneous connections without error

### Dependencies to add

- `github.com/gorilla/websocket` in `apps/realtime/go.mod`

### Files to create/modify

- `apps/realtime/internal/hub/hub.go` — Hub struct and run loop
- `apps/realtime/internal/hub/room.go` — Room struct
- `apps/realtime/internal/hub/client.go` — Client struct with readPump/writePump
- `apps/realtime/internal/handler/ws.go` — WebSocket upgrade handler
- `apps/realtime/main.go` — Initialize Hub, register `/ws/:code` route

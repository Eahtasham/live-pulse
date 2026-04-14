# LivePulse — Realtime Service

The Go WebSocket server powering LivePulse's real-time features. Manages WebSocket connections organized into rooms by session code, and bridges Redis Pub/Sub messages to connected browser clients.

## Architecture

```
                                    ┌──────────────────────────────────┐
                                    │         Realtime Service         │
                                    │                                  │
Browser A ──ws──┐                   │  ┌─────────┐    ┌────────────┐  │
Browser B ──ws──┤──→ /ws/{code} ──→ │  │   Hub   │◄───│ Subscriber │  │
Browser C ──ws──┘                   │  │         │    │  (Redis)   │  │
                                    │  │  Room   │    └──────┬─────┘  │
Browser D ──ws────→ /ws/{code} ──→  │  │  Room   │           │        │
                                    │  └─────────┘           │        │
                                    └────────────────────────┼────────┘
                                                             │
                                    ┌────────────────────────┼────────┐
                                    │         Redis          │        │
                                    │    channel: session:*   │        │
                                    └────────────────────────┼────────┘
                                                             │
                                    ┌────────────────────────┼────────┐
                                    │       API Service      │        │
                                    │      (Publisher)       │        │
                                    └─────────────────────────────────┘
```

## Directory Structure

```
apps/realtime/
├── main.go                          # Entry point, wires Hub + Subscriber + Redis
├── Dockerfile
├── internal/
│   ├── config/config.go             # Environment-based configuration
│   ├── handler/
│   │   ├── health.go                # GET /healthz
│   │   └── ws.go                    # GET /ws/{code} — WebSocket upgrade handler
│   ├── hub/
│   │   ├── hub.go                   # ★ Hub — central room manager + event loop
│   │   ├── room.go                  # ★ Room — group of clients for one session
│   │   └── client.go               # ★ Client — single WebSocket connection
│   ├── message/
│   │   └── types.go                 # Event type constants
│   └── pubsub/
│       └── redis.go                 # ★ Redis Subscriber — per-room channel management
```

## Environment Variables

| Variable        | Default                      | Description                              |
| --------------- | ---------------------------- | ---------------------------------------- |
| `REALTIME_PORT` | `8081`                       | HTTP/WebSocket listen port               |
| `REDIS_URL`     | `redis://localhost:6379/0`   | Redis connection string                  |
| `API_BASE_URL`  | `http://localhost:8080`      | API service URL for session validation   |

## Running

```bash
cd apps/realtime
go run main.go

# Or with Docker
docker compose up realtime
```

## Endpoints

| Method | Path          | Description                                      |
| ------ | ------------- | ------------------------------------------------ |
| GET    | `/healthz`    | Health check (JSON: service, status, uptime)     |
| GET    | `/ws/{code}`  | WebSocket upgrade for a session room             |

---

## WebSocket Connection Lifecycle

### 1. Connection

```
Browser                          Realtime Service                    API Service
  │                                    │                                  │
  │─── GET /ws/A1B2C3 ──────────────▶ │                                  │
  │    Upgrade: websocket              │                                  │
  │                                    │─── GET /v1/sessions/A1B2C3 ────▶│
  │                                    │◄── 200 {status: "active"} ──────│
  │                                    │                                  │
  │◄── 101 Switching Protocols ───────│                                  │
  │                                    │                                  │
  │    [WebSocket open]                │  Hub.Register(client)            │
  │                                    │  → Room "A1B2C3" created        │
  │                                    │  → Redis SUBSCRIBE session:A1B2C3│
```

### 2. Receiving Events

```
API Service                     Redis                          Realtime Service           Browser
  │                               │                                  │                      │
  │── PUBLISH session:A1B2C3 ──▶ │                                  │                      │
  │   {type: "vote_update",...}   │── channel message ─────────────▶│                      │
  │                               │                                  │  Hub.Broadcast       │
  │                               │                                  │  → Room.broadcast()  │
  │                               │                                  │  → Client.send chan  │
  │                               │                                  │── ws text frame ───▶│
  │                               │                                  │                      │
```

### 3. Disconnection

```
Browser                          Realtime Service
  │                                    │
  │─── [connection closed] ──────────▶│
  │                                    │  Hub.Unregister(client)
  │                                    │  → Room "A1B2C3" has 0 clients
  │                                    │  → Room destroyed
  │                                    │  → Redis UNSUBSCRIBE session:A1B2C3
```

### Rejection Cases

| Scenario                        | Response                                |
| ------------------------------- | --------------------------------------- |
| Invalid/unknown session code    | WebSocket close code `4404`             |
| Session status is `closed`      | WebSocket close code `4404`             |
| Session status is `archived`    | WebSocket close code `4404`             |
| API service unreachable         | WebSocket close code `4404`             |

---

## Component Deep Dive

### Hub (`internal/hub/hub.go`)

The Hub is a **single goroutine** running an infinite `select` loop over three channels:

```go
type Hub struct {
    rooms      map[string]*Room     // session code → Room
    register   chan *Client         // new client connected
    unregister chan *Client         // client disconnected
    Broadcast  chan BroadcastMessage // message to send to a room
    subscriber RoomSubscriber       // Redis subscribe/unsubscribe hooks
}
```

**Why a single goroutine?** All room mutations (create, destroy, add client, remove client) happen in one goroutine, eliminating the need for mutexes. The `register`, `unregister`, and `Broadcast` channels serialize all operations.

**Room lifecycle hooks:** When a room is created, `subscriber.Subscribe(code)` is called. When destroyed, `subscriber.Unsubscribe(code)`. This ensures Redis subscriptions exactly match active rooms.

### Room (`internal/hub/room.go`)

```go
type Room struct {
    Code    string
    clients map[*Client]bool
}
```

- `broadcast(message)` pushes to each client's `send` channel
- If a client's `send` buffer is full, the client is dropped (prevents slow clients from blocking the room)
- Logs client count on join/leave for monitoring

### Client (`internal/hub/client.go`)

Each WebSocket connection spawns two goroutines:

```
                    ┌──────────────┐
 ws.ReadMessage ◄───│   readPump   │──▶ Hub.unregister (on error/close)
                    └──────────────┘
                    ┌──────────────┐
 ws.WriteMessage ◄──│  writePump   │◀── client.send channel
                    │              │──▶ ws.PingMessage (every 30s)
                    └──────────────┘
```

| Parameter         | Value   | Purpose                                    |
| ----------------- | ------- | ------------------------------------------ |
| `sendBufferSize`  | 256     | Buffered `send` channel capacity           |
| `maxMessageSize`  | 4 KB    | `SetReadLimit` — prevents abuse            |
| `pingPeriod`      | 30s     | How often server sends ping frames         |
| `pongWait`        | 60s     | Disconnect if no pong received             |
| `writeWait`       | 10s     | Deadline for each write operation          |

**readPump** currently logs and discards incoming messages (future phases will add client→server commands).

**writePump** batches queued messages — if multiple messages are in the `send` channel, they're written in a single WebSocket write for efficiency.

### Subscriber (`internal/pubsub/redis.go`)

Manages one Redis Pub/Sub subscription per active room:

```go
type Subscriber struct {
    rdb     *redis.Client
    subs    map[string]*redis.PubSub  // code → subscription
    handler func(code string, message []byte)
}
```

- **`Subscribe(code)`** — Creates a `redis.PubSub` for channel `session:{code}`, starts a goroutine that reads messages and calls the handler. Idempotent — calling twice for the same code is a no-op.
- **`Unsubscribe(code)`** — Closes the Pub/Sub and removes it from the map.
- **`Close()`** — Closes all subscriptions on shutdown.

The handler function is wired in `main.go` to push messages into `Hub.Broadcast`:

```go
sub := pubsub.NewSubscriber(rdb, func(code string, message []byte) {
    h.Broadcast <- hub.BroadcastMessage{
        Code:    code,
        Message: message,
    }
})
```

### Session Validation (`internal/handler/ws.go`)

Before upgrading a WebSocket connection, the handler validates the session by calling the API service:

```
GET {API_BASE_URL}/v1/sessions/{code}
```

- **200 + `status: "active"`** → proceed with upgrade
- **200 + any other status** → reject (close code 4404)
- **404 or error** → reject (close code 4404)

This keeps the realtime service stateless — it has no database connection and relies on the API as the source of truth.

---

## Redis Pub/Sub — Subscriber Side

### Channel Convention

```
session:{SESSION_CODE}
```

Example: `session:A1B2C3`

The API service publishes to this channel. The realtime service subscribes per-room.

### Subscribe/Unsubscribe Lifecycle

| Event                              | Redis Action                          |
| ---------------------------------- | ------------------------------------- |
| First client connects to room      | `SUBSCRIBE session:{code}`            |
| Additional clients join same room  | No action (already subscribed)        |
| Client disconnects, room not empty | No action (still subscribed)          |
| Last client disconnects            | `UNSUBSCRIBE session:{code}`          |

This means:
- Zero clients connected → zero Redis subscriptions → zero overhead
- 1000 clients in one room → one Redis subscription for that room
- Each active room = exactly one Redis channel subscription

### Message Flow

```
Redis channel message
    │
    ▼
Subscriber.handler(code, rawJSON)
    │
    ▼
Hub.Broadcast <- BroadcastMessage{Code, Message}
    │
    ▼
Hub event loop receives on Broadcast channel
    │
    ▼
Room.broadcast(message)
    │
    ▼
For each client in room:
    client.send <- message  (buffered channel, cap 256)
    │
    ▼
Client.writePump reads from send channel
    │
    ▼
ws.WriteMessage(TextMessage, message)
    │
    ▼
Browser receives WebSocket text frame
```

The raw JSON from Redis is forwarded **as-is** to WebSocket clients — no parsing or transformation on the realtime side.

### Event Types Received

| Event              | Trigger                              |
| ------------------ | ------------------------------------ |
| `vote_update`      | Vote cast on a poll                  |
| `new_question`     | Audience submitted a question        |
| `new_comment`      | Audience submitted a comment         |
| `qa_update`        | Host moderated or audience voted on Q&A |
| `session_closed`   | Host closed the session              |

See the [API README](../api/README.md) for payload schemas.

---

## Monitoring

The service logs structured JSON to stdout:

```json
{"time":"...","level":"INFO","msg":"room created","code":"A1B2C3"}
{"time":"...","level":"INFO","msg":"client joined room","code":"A1B2C3","clients":1}
{"time":"...","level":"INFO","msg":"subscribed to redis channel","channel":"session:A1B2C3"}
{"time":"...","level":"INFO","msg":"client joined room","code":"A1B2C3","clients":2}
{"time":"...","level":"INFO","msg":"client left room","code":"A1B2C3","clients":1}
{"time":"...","level":"INFO","msg":"client left room","code":"A1B2C3","clients":0}
{"time":"...","level":"INFO","msg":"room destroyed","code":"A1B2C3"}
{"time":"...","level":"INFO","msg":"unsubscribed from redis channel","channel":"session:A1B2C3"}
```

---

## Load Testing

A browser-based load test tool is available at `/load-test.html` (served from `apps/web/public/`).

It connects a WebSocket, then fires N votes (each with a unique audience UID) at configurable delay, displaying a live-updating bar chart as `vote_update` events stream back through the WebSocket.

See [load-test.html](../../apps/web/public/load-test.html) for details.

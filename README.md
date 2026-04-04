# LivePulse 🟢

**A Scalable Real-Time Polling and Q&A Platform Using Event-Driven Cloud Architecture**

LivePulse lets presenters and meeting hosts engage their audience through live polls and crowd-sourced Q&A sessions. Results update in real-time for all participants via WebSocket — no page refresh needed.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser (React SPA)                       │
│  HomePage | SessionPage | PollCard | QuestionList | Forms        │
│                    ↑↓ REST API   ↑↓ WebSocket                   │
└──────────────────────────────┬──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                    Backend (Node.js + Express)                    │
│  /api/sessions  /api/polls  /api/questions                       │
│  Socket.io server ← EventBus (pub/sub) ← Redis                  │
└───────────────┬─────────────────────────────┬───────────────────┘
                │                             │
┌───────────────▼──────────┐   ┌─────────────▼───────────────────┐
│     MongoDB (Mongoose)    │   │     Redis (ioredis pub/sub)      │
│  sessions / polls /       │   │  Decouples HTTP handlers from   │
│  questions collections    │   │  Socket.io broadcast layer      │
└──────────────────────────┘   └─────────────────────────────────┘
```

### Key design decisions

| Concern | Solution |
|---|---|
| Real-time delivery | Socket.io rooms scoped per session (`session:<id>`) |
| Horizontal scalability | All events go through Redis pub/sub before hitting Socket.io; any number of backend pods can receive and relay them |
| Graceful degradation | `EventBus` falls back to an in-process `EventEmitter` when Redis is unavailable (useful for local dev / CI) |
| Data model | MongoDB stores sessions, polls (with embedded option sub-documents), and questions |

---

## Features

- 🎤 **Host a session** — create a session with a shareable 6-char join code
- 🔗 **Join a session** — attendees paste the code to enter the live room
- 📊 **Live polls** — hosts launch multiple-choice polls; votes appear as animated bar charts in real-time
- 🙋 **Live Q&A** — attendees submit questions; everyone can upvote; hosts mark questions answered or pin them
- ⚡ **Event-driven** — every vote / question / answer is published to the Redis event bus and broadcast to all connected clients within milliseconds

---

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React 18, Vite, React Router, Socket.io-client |
| Backend | Node.js 20, Express 4, Socket.io 4 |
| Database | MongoDB 7 (Mongoose ODM) |
| Event bus | Redis 7 (ioredis pub/sub) |
| Containers | Docker, Docker Compose |

---

## Quick Start (Docker Compose)

```bash
# 1. Clone the repo
git clone https://github.com/Eahtasham/LivePulse.git
cd LivePulse

# 2. Spin up all services (MongoDB, Redis, backend, frontend)
docker compose up --build

# 3. Open the app
open http://localhost:5173
```

### Local development (without Docker)

Prerequisites: Node.js >= 20, a running MongoDB instance, a running Redis instance.

```bash
# Backend
cd backend
cp .env.example .env          # edit MONGODB_URI / REDIS_URL if needed
npm install
npm run dev                   # listens on :4000

# Frontend (new terminal)
cd frontend
npm install
npm run dev                   # listens on :5173
```

---

## Running Tests

```bash
cd backend
npm test
```

All 15 unit / integration tests pass without any external services (MongoDB and Redis are mocked).

---

## API Reference

### Sessions

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/sessions` | Create a session |
| `GET` | `/api/sessions/:id` | Get session by id |
| `GET` | `/api/sessions/code/:code` | Get session by join code |
| `PATCH` | `/api/sessions/:id/close` | Close a session |

### Polls

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/polls` | Create a poll |
| `GET` | `/api/polls/session/:sessionId` | List polls for a session |
| `GET` | `/api/polls/:id` | Get a single poll |
| `POST` | `/api/polls/:id/vote` | Cast a vote (`{ optionIndex }`) |
| `PATCH` | `/api/polls/:id/close` | Close a poll |

### Questions

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/questions` | Submit a question |
| `GET` | `/api/questions/session/:sessionId` | List questions (sorted by votes) |
| `POST` | `/api/questions/:id/upvote` | Upvote a question |
| `PATCH` | `/api/questions/:id/answer` | Mark as answered |
| `PATCH` | `/api/questions/:id/pin` | Toggle pin |

### Socket.io Events

Clients join a room by emitting `join-session` with `{ sessionId }`.

| Event | Direction | Payload |
|---|---|---|
| `join-session` | client → server | `{ sessionId }` |
| `poll:created` | server → room | Poll document |
| `poll:updated` | server → room | Poll document |
| `poll:vote` | server → room | Poll document (with updated counts) |
| `question:created` | server → room | Question document |
| `question:upvoted` | server → room | Question document |
| `question:answered` | server → room | Question document |
| `question:pinned` | server → room | Question document |

---

## Project Structure

```
LivePulse/
├── backend/
│   ├── src/
│   │   ├── config/        # MongoDB connection
│   │   ├── models/        # Mongoose schemas (Session, Poll, Question)
│   │   ├── routes/        # Express route handlers
│   │   ├── services/      # EventBus (Redis pub/sub)
│   │   ├── socket/        # Socket.io handlers
│   │   ├── __tests__/     # Jest tests
│   │   └── server.js
│   ├── Dockerfile
│   └── package.json
├── frontend/
│   ├── src/
│   │   ├── components/    # Navbar, PollCard, QuestionList, Forms
│   │   ├── pages/         # HomePage, SessionPage
│   │   ├── context/       # SocketContext
│   │   ├── api.js         # Fetch wrapper for REST API
│   │   ├── channels.js    # Socket event channel names
│   │   └── App.jsx
│   ├── Dockerfile
│   └── package.json
├── docker-compose.yml
└── README.md
```

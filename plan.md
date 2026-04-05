# LivePulse — Implementation Roadmap

**Version:** 2.0  
**Date:** April 5, 2026  
**Total Phases:** 20  
**Current Status:** Phase 1 complete (scaffolding, healthz, DB schema, Docker)

---

## How to Read This Document

Each phase describes **what** to build and **how to verify it works** — not how to write the code. Every phase ends with a set of **acceptance tests** written as plain-language scenarios. A phase is only complete when every test passes.

**Each phase is strictly single-domain** — either Backend (Go), Frontend (Next.js), or Infra. No phase mixes frontend and backend work.

Phases are ordered by dependency. Some phases can be worked on in parallel (marked with `∥`). Each phase lists its prerequisites.

**Legend:**
- 🟢 **BACKEND** — Go API / Realtime service work
- 🔵 **FRONTEND** — Next.js / React / UI work
- 🟠 **INFRA** — Docker, cloud, CI/CD, deployment

---

## Phase 1 — Project Foundation 🟠 INFRA ✅ COMPLETE

**Goal:** Bootable monorepo with infrastructure running locally.

**What exists:**
- Turborepo monorepo with Next.js + two Go services
- Docker Compose running Postgres 16 + Redis 7
- Database migrations for all 7 tables
- GORM models matching the schema
- `/healthz` endpoints on both Go services returning JSON
- Structured JSON logging via slog
- PowerShell task runner (`run.ps1`)

**Tests — all passing:**
- [ ] `.\run.ps1 docker-up` → both containers reach `healthy` status
- [ ] `.\run.ps1 migrate-up` → all 7 migrations apply without error
- [ ] `curl localhost:8080/healthz` → `{"service":"api","status":"ok",...}`
- [ ] `curl localhost:8081/healthz` → `{"service":"realtime","status":"ok",...}`
- [ ] `localhost:3000` → Next.js landing page renders

---

## ── BACKEND BLOCK 1: Core API ──────────────────────────────

---

## Phase 2 — Auth Backend 🟢 BACKEND

**Goal:** The Go API can validate JWTs, extract user identity, and protect routes. Users table is populated on first login.

**Prerequisite:** Phase 1

**Scope (backend only — NO login UI):**
- JWT validation middleware on the Go API (`middleware/auth.go`)
- A shared secret between Next.js (Auth.js) and Go for JWT signing/verification
- `POST /v1/auth/callback` — receives user info from the OAuth flow, creates/finds a `users` row, returns a JWT
- Protected route group: any route under `/v1/sessions` (POST, PATCH, DELETE) requires a valid JWT
- Public route group: GET routes and vote/Q&A submission routes remain open

**How to test without frontend:** Use `curl` or Postman with a manually crafted JWT.

**Tests:**
- [ ] `POST /v1/sessions` without an Authorization header → 401 Unauthorized
- [ ] `POST /v1/sessions` with a valid JWT → 200 (or 201)
- [ ] `POST /v1/sessions` with an expired JWT → 401 (not 500)
- [ ] `POST /v1/sessions` with a malformed JWT → 401
- [ ] `GET /v1/sessions/:code` without a JWT → 200 (public route, no auth needed)
- [ ] `POST /v1/auth/callback` with `{email, name, avatar, provider}` → creates a `users` row and returns a JWT
- [ ] Calling callback again with the same email → does NOT create a duplicate user, returns JWT for existing user
- [ ] The JWT payload contains `user_id` and `email` — extractable in handlers
- [ ] The JWT expires after 24 hours (configurable via env var)
- [ ] Invalid JWT signatures (wrong secret) are rejected with 401

---

## Phase 3 — Session API 🟢 BACKEND

**Goal:** Full CRUD for sessions. Hosts create sessions, audiences join with a 6-character code and receive an ephemeral identity.

**Prerequisite:** Phase 2

**Endpoints:**
- `POST /v1/sessions` (auth required) — create session, generate 6-char code
- `GET /v1/sessions/:code` (public) — get session by code
- `GET /v1/sessions` (auth required) — list host's sessions
- `POST /v1/sessions/:code/join` (public) — issue ephemeral UID, store in Redis with TTL

**Tests:**
- [ ] `POST /v1/sessions` with `{title: "CS101 Lecture"}` → returns `{id, code, title, status: "active"}`
- [ ] The `code` is exactly 6 alphanumeric characters
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

---

## Phase 4 — Poll API 🟢 BACKEND

**Goal:** Full CRUD for polls within a session. Hosts create polls with options, manage lifecycle (draft → active → closed).

**Prerequisite:** Phase 3

**Endpoints:**
- `POST /v1/sessions/:code/polls` (auth) — create poll with options
- `GET /v1/sessions/:code/polls` (public) — list polls (only active + closed for audience)
- `GET /v1/sessions/:code/polls/:id` (public) — get poll with option vote counts
- `PATCH /v1/sessions/:code/polls/:id` (auth) — update status (activate/close)

**Tests:**
- [ ] Create poll with `{question, answer_mode: "single", options: [{label, position},...]}` → returns poll with options
- [ ] Poll is created with `status: "draft"` by default
- [ ] Cannot create a poll with fewer than 2 options → 400
- [ ] Cannot create a poll with more than 6 options → 400
- [ ] `GET /v1/sessions/:code/polls` as audience → returns only `active` and `closed` polls (NOT `draft`)
- [ ] `PATCH` to activate → `status` changes to `active`
- [ ] `PATCH` to close → `status` changes to `closed`
- [ ] Cannot transition from `closed` → `active` → 400
- [ ] Cannot transition from `closed` → `draft` → 400
- [ ] Options are returned in `position` order
- [ ] Only the session host can create/activate/close polls → other users get 403
- [ ] Creating a poll for a non-existent session → 404
- [ ] Creating a poll for an archived session → 400

---

## Phase 5 — Vote API 🟢 BACKEND

**Goal:** Audience members can cast votes on active polls. One-vote-per-poll enforcement. Both single and multi-select modes.

**Prerequisite:** Phase 4

**Endpoint:**
- `POST /v1/sessions/:code/polls/:id/vote` (public, requires audience UID)

**Request body:** `{option_ids: ["uuid"], audience_uid: "uuid"}`

**Tests:**
- [ ] Vote on an active poll → 200, vote row created in DB
- [ ] `votes` row contains correct `poll_id`, `option_id`, `audience_uid`
- [ ] Voting on the same poll again with the same UID → error (duplicate vote)
- [ ] In `single` mode, sending 2 option IDs → 400 validation error
- [ ] In `multi` mode, sending 2 option IDs → 200, 2 vote rows created
- [ ] Voting on a `draft` poll → 400 ("Poll is not active")
- [ ] Voting on a `closed` poll → 400 ("Poll is closed")
- [ ] Voting with an `option_id` that doesn't belong to this poll → 400
- [ ] Voting with a fabricated `audience_uid` not in Redis → 401
- [ ] After voting, `GET /polls/:id` shows updated vote counts per option
- [ ] Two different UIDs both voting → both succeed, counts reflect both votes
- [ ] Vote counts are accurate: 3 votes for A, 2 for B → `{A: 3, B: 2}`

---

## Phase 6 — Q&A API 🟢 BACKEND

**Goal:** Full CRUD for Q&A entries. Audience submits questions/comments. Host moderates (answer, pin, hide, archive).

**Prerequisite:** Phase 3

**Endpoints:**
- `POST /v1/sessions/:code/qa` (public, requires audience UID) — submit entry
- `GET /v1/sessions/:code/qa` (public) — list active entries (cursor-paginated)
- `PATCH /v1/sessions/:code/qa/:id` (auth) — moderate (pin/answer/hide/archive)

**Tests:**
- [ ] Submit question: `{entry_type: "question", body: "What is Big-O?"}` → 201, row created
- [ ] Submit comment: `{entry_type: "comment", body: "Great lecture!"}` → 201
- [ ] Empty body → 400 validation error
- [ ] Body over 2000 characters → 400
- [ ] `author_uid` in the DB matches the audience UID
- [ ] `GET /qa` returns entries sorted by score DESC, then created_at ASC
- [ ] Hidden entries (`is_hidden = TRUE`) are NOT returned by `GET /qa`
- [ ] Cursor-based pagination: `GET /qa?cursor=xxx&limit=20` returns next 20 entries
- [ ] Host marks as "answered" → `status = 'answered'` in DB
- [ ] Host pins → `status = 'pinned'`
- [ ] Host hides → `is_hidden = TRUE`, status remains unchanged
- [ ] Host unhides → `is_hidden = FALSE`
- [ ] A question that is `answered` AND hidden: `status = 'answered'`, `is_hidden = TRUE`
- [ ] Non-host sending PATCH → 403
- [ ] Submitting Q&A to an archived session → 400

---

## Phase 7 — Q&A Vote API 🟢 BACKEND

**Goal:** Upvote/downvote system for questions. Toggle behavior. Score recalculation.

**Prerequisite:** Phase 6

**Endpoint:**
- `POST /v1/sessions/:code/qa/:id/vote` (public, requires audience UID)

**Request body:** `{audience_uid: "uuid", value: 1}` (1 = upvote, -1 = downvote)

**Tests:**
- [ ] Upvote a question → `qa_votes` row with `vote_value = 1`
- [ ] Downvote → row with `vote_value = -1`
- [ ] Upvote same question a second time → removes the vote (toggle off)
- [ ] Upvote then downvote → row updates from `1` to `-1` (upsert)
- [ ] `qa_entries.score` = `SUM(vote_value)` from all `qa_votes` for that entry
- [ ] 3 upvotes + 1 downvote → score = 2
- [ ] Attempting to vote on a comment → 400 ("Cannot vote on comments")
- [ ] `UNIQUE(qa_entry_id, voter_uid)` prevents DB-level duplicates
- [ ] Two different audience members can both vote on the same question
- [ ] After voting, `GET /qa` returns entries with updated scores, re-sorted

---

## Phase 8 — WebSocket Hub & Rooms 🟢 BACKEND

**Goal:** The realtime service accepts WebSocket connections, organizes clients into rooms by session code, and manages client lifecycle.

**Prerequisite:** Phase 3

**Scope:** realtime service only — no Redis Pub/Sub wiring yet. Just the WebSocket infrastructure.

**What to build:**
- `GET /ws/:code` — WebSocket upgrade handler
- Hub struct: map of rooms → set of clients
- Room struct: session code + connected clients
- Client struct: per-client read/write pump goroutines
- Ping/pong keepalive (client pings every 30s, server disconnects after 60s of silence)
- Graceful disconnect cleanup

**Tests:**
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

---

## Phase 9 — Redis Pub/Sub Bridge 🟢 BACKEND

**Goal:** Wire the API service (publisher) to the realtime service (subscriber). Votes, Q&A submissions, and moderation actions trigger real-time broadcasts to all room clients.

**Prerequisite:** Phase 5 + Phase 7 + Phase 8

**How it works:**
1. API service: after a successful DB write (vote, Q&A submit, moderation action), PUBLISH an event to Redis channel `session:{code}`
2. Realtime service: SUBSCRIBE to `session:{code}` channels when the first client joins a room
3. On receiving a Redis message, the hub broadcasts it to all WebSocket clients in that room
4. UNSUBSCRIBE when the last client leaves (room empty)

**Event types published by API:**
- `vote_update` — after a vote is cast
- `new_question` — after a question is submitted
- `new_comment` — after a comment is submitted
- `qa_update` — after a moderation action or Q&A vote
- `session_closed` — after a session is closed

**Tests:**
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

---

## Phase 10 — Session Lifecycle API 🟢 BACKEND

**Goal:** Host can close a session. Closing archives Q&A, blocks new actions, broadcasts `session_closed`, and cleans up.

**Prerequisite:** Phase 9

**Endpoint:**
- `PATCH /v1/sessions/:code/close` (auth required)

**Tests:**
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

---

## Phase 11 — Export API 🟢 BACKEND

**Goal:** Host can export session results (polls + Q&A) as CSV or JSON. File is uploaded to S3 and a presigned download URL is returned.

**Prerequisite:** Phase 10

**Endpoint:**
- `GET /v1/sessions/:code/export?format=csv` (auth required)
- `GET /v1/sessions/:code/export?format=json` (auth required)

**Tests:**
- [ ] Export as CSV → response contains `{download_url: "https://s3.../..."}` 
- [ ] Export as JSON → same response structure
- [ ] The `download_url` is a valid presigned S3 URL
- [ ] Downloading within 15 minutes → file downloads successfully
- [ ] Downloading after 15 minutes → access denied (expired)
- [ ] CSV contains headers: session info, poll questions with options and vote counts, Q&A entries with scores
- [ ] JSON is valid and contains nested structure: session → polls → options → counts + qa_entries
- [ ] Exporting an active session works (not just archived)
- [ ] Exporting an empty session (no polls, no Q&A) → valid file with just session metadata
- [ ] Only the session host can export → 403 for others
- [ ] Invalid format parameter (`?format=xml`) → 400

---

## ── FRONTEND BLOCK: Build the UI ───────────────────────────

---

## Phase 12 — Auth UI & Dashboard 🔵 FRONTEND

**Goal:** Host can sign in with Google, see a dashboard listing their sessions, and sign out.

**Prerequisite:** Phase 2 + Phase 3 (backend auth + session API must exist)

**Pages:**
- `/login` — "Sign in with Google" button
- `/dashboard` — session list, "New Session" button, sign out
- Protected routes: redirect to `/login` if unauthenticated

**Tests:**
- [ ] Clicking "Sign in with Google" redirects to Google consent screen
- [ ] After consent, user lands on `/dashboard` (not `/login`)
- [ ] Refreshing `/dashboard` keeps user logged in (cookie persists)
- [ ] The dashboard shows a list of the host's sessions (fetched from `GET /v1/sessions`)
- [ ] Sessions are sorted by most recent first
- [ ] Each session shows: title, code, status badge (Active / Archived), created date
- [ ] Visiting `/dashboard` without auth → redirected to `/login`
- [ ] Clicking "Sign out" → clears session, redirects to `/login`
- [ ] After sign out, `/dashboard` redirects to `/login`

---

## Phase 13 — Session UI 🔵 FRONTEND

**Goal:** Host can create a session from the dashboard. Audience can join via code input or direct URL.

**Prerequisite:** Phase 12 + Phase 3 (session API)

**Pages / Components:**
- Dashboard: "New Session" modal/form → calls `POST /v1/sessions`
- `/session/[code]` — audience view (calls `POST /v1/sessions/:code/join` on load)
- Landing page: "Join Session" input field (enter code → navigate to `/session/:code`)

**Tests:**
- [ ] Clicking "New Session" → modal appears with title input
- [ ] Entering a title and submitting → session is created, code is displayed prominently
- [ ] The session code can be copied to clipboard with one click
- [ ] A shareable link (`/session/A1B2C3`) is shown
- [ ] Navigating to `/session/A1B2C3` as an audience member → shows session title, "You've joined" confirmation
- [ ] Navigating to `/session/XXXXXX` → shows "Session not found" error page
- [ ] The landing page has a "Join Session" input — entering a code navigates to `/session/:code`
- [ ] Code input is case-insensitive: entering `a1b2c3` navigates to `/session/A1B2C3`
- [ ] New session appears in the dashboard list immediately after creation

---

## Phase 14 — Poll UI 🔵 FRONTEND

**Goal:** Host can create polls within a session. Audience can see active polls, vote, and see results.

**Prerequisite:** Phase 13 + Phase 4 + Phase 5 (session UI + poll API + vote API)

**Components:**
- Host: "Create Poll" form (question, options, answer mode, time limit)
- Host: Poll list with Activate/Close buttons per poll
- Audience: Active poll display with vote buttons (radio for single, checkbox for multi)
- Both: Results bar chart showing vote distribution

**Tests:**
- [ ] Host clicks "Create Poll" → form appears with question + option inputs
- [ ] Can add up to 6 options, cannot submit with fewer than 2
- [ ] After creating, poll appears in the poll list as `Draft`
- [ ] Host clicks "Activate" → poll status changes to `Active`
- [ ] Audience sees the active poll with votable options
- [ ] Single-answer poll: radio buttons — selecting one deselects others
- [ ] Multi-select poll: checkboxes — can select multiple
- [ ] After voting, audience sees "You voted" confirmation and a bar chart of results
- [ ] Attempting to vote again shows "Already voted" message
- [ ] Host clicks "Close" → poll status changes to `Closed`
- [ ] Closed poll shows final results (static chart) for both host and audience
- [ ] Draft polls are NOT visible to the audience

---

## Phase 15 — Q&A UI 🔵 FRONTEND

**Goal:** Audience can submit questions/comments. Host has a moderation panel with action buttons. Upvote/downvote on questions.

**Prerequisite:** Phase 13 + Phase 6 + Phase 7 (session UI + Q&A API + Q&A vote API)

**Components:**
- Audience: text input with Question/Comment toggle + submit button
- Audience: Q&A feed showing entries sorted by score
- Audience: upvote/downvote arrows on questions (not comments)
- Host: moderation panel — same feed with action buttons (Answer, Pin, Hide, Archive)

**Tests:**
- [ ] Audience types a question and submits → it appears in the feed
- [ ] Audience toggles to "Comment" and submits → comment appears (no vote arrows)
- [ ] Questions show upvote/downvote arrows, comments do not
- [ ] Clicking upvote → score increments by 1 visually
- [ ] Clicking upvote again → score decrements (toggle off)
- [ ] Clicking downvote after upvote → score changes by -2 (flip)
- [ ] Host clicks "Mark Answered" → green "Answered" badge appears
- [ ] Host clicks "Pin" → entry moves to the top
- [ ] Host clicks "Hide" → entry disappears from audience view, stays in host panel with "[Hidden]" label
- [ ] Host clicks "Unhide" → entry reappears for audience
- [ ] Feed loads 20 entries at a time with "Load more" button (cursor pagination)

---

## Phase 16 — WebSocket Client & Live Updates 🔵 FRONTEND

**Goal:** The frontend connects to the realtime WebSocket service and updates polls, Q&A, and session state in real time without page refresh.

**Prerequisite:** Phase 14 + Phase 15 + Phase 9 (poll UI + Q&A UI + Redis Pub/Sub bridge)

**What to build:**
- `useWebSocket` hook: connect to `ws://localhost:8081/ws/:code`, auto-reconnect, 30s ping
- `usePollVotes` hook: listen for `vote_update` events, update bar chart state
- `useQAFeed` hook: listen for `new_question`, `new_comment`, `qa_update` events, update feed state
- Handle `session_closed` event: show "Session ended" overlay, disable all inputs

**Tests:**
- [ ] Opening a session page → WebSocket connects (visible in browser DevTools Network tab)
- [ ] Client A votes → Client B (same session, different browser) sees updated chart within 1 second
- [ ] Client C (different session) does NOT see the update
- [ ] New question submitted → appears in all connected clients' feeds instantly
- [ ] Host moderates (pin/hide/answer) → change is reflected on all audience screens
- [ ] Q&A upvote → score updates on all screens, feed re-sorts
- [ ] If WebSocket disconnects (e.g., network drop), it auto-reconnects within 5 seconds
- [ ] After reconnect, client fetches current state via REST as a fallback (no stale data)
- [ ] Host closes session → all audience screens show "This session has ended"
- [ ] After session close, inputs are disabled and no further submissions are possible
- [ ] 10 rapid votes → chart updates smoothly for all 10 (no dropped events)

---

## Phase 17 — Frontend Polish & UX 🔵 FRONTEND

**Goal:** Responsive design, smooth animations, loading states, error handling, and accessibility.

**Prerequisite:** Phase 16

**Areas:**
- Mobile-first responsive layout (audience is typically on phones)
- Chart animations on vote count changes
- Loading skeletons while fetching data
- Error toasts for failed submissions
- Large, tap-friendly controls (44x44px minimum touch targets)
- Session code displayed in a large, mono-spaced, copyable format
- Branding: LivePulse logo, consistent color scheme

**Tests:**
- [ ] App is usable on 375px-wide screen (iPhone SE) — no horizontal scrolling
- [ ] Vote buttons are at least 44x44px (touch target guideline)
- [ ] After voting, selected option shows a visual "selected" state within 200ms
- [ ] Chart bars animate smoothly when counts change (CSS transitions)
- [ ] Loading states (spinners or skeletons) are shown while waiting for API responses
- [ ] Failed API calls show a user-friendly error message (not raw error text)
- [ ] "Join Session" input accepts codes in any case (`a1b2c3` → `A1B2C3`)
- [ ] Session code is displayed in a large, copyable, mono-spaced format
- [ ] HTML is semantically correct — proper headings, labels, ARIA attributes
- [ ] Color contrast ratios meet WCAG AA (4.5:1 for text)
- [ ] Tab navigation works through all interactive elements (keyboard accessible)

---

## ── BACKEND BLOCK 2: Hardening ─────────────────────────────

---

## Phase 18 — Audience Identity Hardening 🟢 BACKEND

**Goal:** Harden the ephemeral audience identity system against spoofing. Ensure server-side enforcement is airtight.

**Prerequisite:** Phase 5 (can be done in parallel with frontend work)

**What to tighten:**
- Audience UID is issued as a short-lived JWT (not bare UUID) — signed with the same JWT_SECRET
- The API validates the audience JWT on every vote and Q&A submission
- Fabricating a UID without knowing the secret is impossible
- Redis TTL serves as a secondary guard — expired UIDs are rejected even if JWT is valid
- The Postgres UNIQUE constraint is the final enforcement layer (defense in depth)

**Tests:**
- [ ] A fabricated UID (valid UUID format but not issued by the server) → 401
- [ ] An expired audience JWT → 401
- [ ] A valid audience JWT for session A used on session B → 401 (session-scoped)
- [ ] After a session is closed, audience Redis keys are cleaned up (or left to TTL-expire)
- [ ] The audience JWT contains NO personally identifiable information (just uid + session code)
- [ ] Even if someone extracts the UID from the JWT, the Postgres UNIQUE constraint prevents double-voting
- [ ] Clearing cookies and re-joining → new UID issued (fresh identity, can't vote again on already-voted polls — wait: they CAN since it's a new UID. This is acceptable for an anonymous system)

---

## Phase 19 — Security Hardening & Rate Limiting 🟢 BACKEND

**Goal:** Protect against common attacks. Rate-limit abuse-prone endpoints. Pass a basic security review.

**Prerequisite:** Phase 10

**Rate Limiting (Redis token-bucket):**
- Vote endpoint: 10 requests/minute per audience UID
- Q&A submission: 5 requests/minute per audience UID
- Session creation: 3 requests/minute per host
- Returns `429 Too Many Requests` with `Retry-After` header

**Input Validation:**
- Session code: exactly 6 alphanumeric characters
- Poll question: max 500 characters
- Q&A body: max 2000 characters, HTML stripped
- Poll options: max 200 characters each, 2–6 options

**Security Headers & CORS:**
- CORS: allow only the frontend origin
- HTTP-only, Secure, SameSite=Lax cookies
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`

**Tests:**
- [ ] 20 votes in 10 seconds → first 10 succeed, next 10 return 429
- [ ] Rate limit resets after the window expires → next request succeeds
- [ ] `<script>alert('xss')</script>` as Q&A body → stored as plain text, NOT executed when rendered
- [ ] Q&A body > 2000 characters → 400
- [ ] Poll question > 500 characters → 400
- [ ] API rejects requests from `http://evil-site.com` (wrong CORS origin)
- [ ] API accepts requests from `http://localhost:3000` (correct origin)
- [ ] `Set-Cookie` header includes `HttpOnly`, `Secure`, `SameSite` flags
- [ ] SQL injection attempt in session code (`' OR 1=1--`) → 404, no stack trace leak
- [ ] All responses include `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY`
- [ ] No credentials visible in error responses or logs

---

## ── INFRA BLOCK: Ship It ───────────────────────────────────

---

## Phase 20 — Containerized Deployment & Observability 🟠 INFRA

**Goal:** All services run as Docker containers in production. CI/CD pipeline builds and deploys on push. Health monitoring catches problems before users do.

**Prerequisite:** Phase 17 (all features complete)

**Deployment target:**
- DigitalOcean Droplet (from $200 student credit) running Docker Compose
- AWS RDS PostgreSQL 16 (free tier)
- AWS S3 for exports (free tier)
- Redis as a Docker container on the Droplet (sidecar)
- GitHub Actions → build images → push to GHCR → SSH deploy

**Observability:**
- Request IDs propagated across services (`X-Request-ID` header)
- WebSocket connect/disconnect logged with session code and client count
- `/healthz` checks DB + Redis connectivity (not just "process is alive")
- JSON logs collected by Docker and queryable

**Tests:**
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

---

## Phase Summary

| #  | Phase                              | Domain       | Depends On    |
| -- | ---------------------------------- | ------------ | ------------- |
| 1  | Project Foundation ✅               | 🟠 INFRA     | —             |
| 2  | Auth Backend                       | 🟢 BACKEND   | 1             |
| 3  | Session API                        | 🟢 BACKEND   | 2             |
| 4  | Poll API                           | 🟢 BACKEND   | 3             |
| 5  | Vote API                           | 🟢 BACKEND   | 4             |
| 6  | Q&A API                            | 🟢 BACKEND   | 3             |
| 7  | Q&A Vote API                       | 🟢 BACKEND   | 6             |
| 8  | WebSocket Hub & Rooms              | 🟢 BACKEND   | 3             |
| 9  | Redis Pub/Sub Bridge               | 🟢 BACKEND   | 5, 7, 8       |
| 10 | Session Lifecycle API              | 🟢 BACKEND   | 9             |
| 11 | Export API                         | 🟢 BACKEND   | 10            |
| 12 | Auth UI & Dashboard                | 🔵 FRONTEND  | 2, 3          |
| 13 | Session UI                         | 🔵 FRONTEND  | 12, 3         |
| 14 | Poll UI                            | 🔵 FRONTEND  | 13, 4, 5      |
| 15 | Q&A UI                             | 🔵 FRONTEND  | 13, 6, 7      |
| 16 | WebSocket Client & Live Updates    | 🔵 FRONTEND  | 14, 15, 9     |
| 17 | Frontend Polish & UX               | 🔵 FRONTEND  | 16            |
| 18 | Audience Identity Hardening        | 🟢 BACKEND   | 5 ∥           |
| 19 | Security Hardening & Rate Limiting | 🟢 BACKEND   | 10            |
| 20 | Deployment & Observability         | 🟠 INFRA     | 17            |

---

## Dependency Graph

```
Phase 1 (Foundation)
  │
  ├── Phase 2 (Auth Backend)
  │     │
  │     ├── Phase 3 (Session API)
  │     │     │
  │     │     ├── Phase 4 (Poll API)
  │     │     │     │
  │     │     │     └── Phase 5 (Vote API) ──────┐
  │     │     │                                   │
  │     │     ├── Phase 6 (Q&A API)               │
  │     │     │     │                             │
  │     │     │     └── Phase 7 (Q&A Vote API)───┐│
  │     │     │                                  ││
  │     │     └── Phase 8 (WebSocket Hub) ───────┤│
  │     │                                        ││
  │     │                    Phase 9 (Pub/Sub) ◄─┘│
  │     │                         │               │
  │     │                    Phase 10 (Lifecycle)  │
  │     │                         │               │
  │     │                    Phase 11 (Export)     │
  │     │                                         │
  │     │                    Phase 18 (Identity) ◄┘  ∥ parallel
  │     │                    Phase 19 (Security)
  │     │
  │     ├── Phase 12 (Auth UI) ◄── needs Phase 2, 3
  │     │     │
  │     │     └── Phase 13 (Session UI) ◄── needs Phase 3
  │     │           │
  │     │           ├── Phase 14 (Poll UI) ◄── needs Phase 4, 5
  │     │           │
  │     │           └── Phase 15 (Q&A UI) ◄── needs Phase 6, 7
  │     │                 │
  │     │                 └── Phase 16 (WS Client) ◄── needs Phase 9
  │     │                       │
  │     │                       └── Phase 17 (Polish)
  │     │                             │
  │     └─────────────────────── Phase 20 (Deploy)
```

---

## Suggested Sprint Groupings

| Sprint | Phases         | Theme                             | Domain                  |
| ------ | -------------- | --------------------------------- | ----------------------- |
| 1      | 2, 3           | Auth + Sessions backend           | 🟢 Backend only         |
| 2      | 4, 5, 6, 7     | Polls + Votes + Q&A backend       | 🟢 Backend only         |
| 3      | 8, 9           | WebSocket + Pub/Sub               | 🟢 Backend only         |
| 4      | 10, 11         | Lifecycle + Export                 | 🟢 Backend only         |
| 5      | 12, 13         | Auth UI + Session UI              | 🔵 Frontend only        |
| 6      | 14, 15         | Poll UI + Q&A UI                  | 🔵 Frontend only        |
| 7      | 16, 17         | Live updates + Polish             | 🔵 Frontend only        |
| 8      | 18, 19, 20     | Security + Deploy                 | 🟢 Backend + 🟠 Infra   |

**Parallelizable:**
- Sprints 1–4 (backend) can have Phase 18 done anytime after Phase 5
- Phases 6, 7 are independent of Phases 4, 5 — can be worked in parallel
- Phases 14 and 15 are independent — can be worked in parallel
- Phases 18 and 19 are independent of each other

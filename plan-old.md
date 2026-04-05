# LivePulse — Implementation Roadmap

**Version:** 1.0  
**Date:** April 5, 2026  
**Total Phases:** 18  
**Current Status:** Phase 1 complete (scaffolding, healthz, DB schema, Docker)

---

## How to Read This Document

Each phase describes **what** to build and **how to verify it works** — not how to write the code. Every phase ends with a set of **acceptance tests** written as plain-language scenarios. A phase is only complete when every test passes.

---

## Phase 1 — Project Foundation ✅ COMPLETE

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

## Phase 2 — Host Authentication

**Goal:** A host can sign up / sign in via Google OAuth, receive a JWT, and access protected routes.

**Prerequisite:** Phase 1

**User Flow:**
1. Host visits `/login` and clicks "Sign in with Google"
2. Redirected to Google consent screen → grants permission
3. Redirected back to `/dashboard` with an active session
4. A `users` row is created (or existing one is found by email)
5. JWT is stored in an HTTP-only secure cookie
6. The Go API validates this JWT on all protected endpoints via middleware

**Key Decisions:**
- Auth.js (NextAuth v5) handles the OAuth flow on the Next.js side
- The Next.js backend issues a JWT after successful OAuth
- The Go API receives this JWT in the `Authorization` header (forwarded by Next.js API proxy) and validates the signature using a shared secret
- No audience sign-up — only hosts authenticate

**Tests:**
- [ ] Clicking "Sign in with Google" redirects to Google's consent screen
- [ ] After Google consent, user lands on `/dashboard` — not `/login`
- [ ] Refreshing `/dashboard` keeps the user logged in (cookie persists)
- [ ] A new `users` row exists in Postgres with the Google email, name, and avatar
- [ ] Signing in again with the same Google account does NOT create a duplicate user
- [ ] Visiting `/dashboard` without authentication redirects to `/login`
- [ ] The Go API rejects requests to protected routes (e.g., `POST /v1/sessions`) with 401 when no JWT is provided
- [ ] The Go API accepts requests with a valid JWT and extracts the user ID correctly
- [ ] An expired JWT returns 401, not 500
- [ ] Clicking "Sign out" clears the session and redirects to `/login`

---

## Phase 3 — Session Creation & Join

**Goal:** A host can create a session with a unique 6-character code. An audience member can join using that code — no account needed.

**Prerequisite:** Phase 2

**Host Flow — Create:**
1. Host clicks "New Session" on the dashboard
2. Enters a session title (e.g., "CS101 Lecture 5")
3. System generates a unique 6-character alphanumeric code (e.g., `A1B2C3`)
4. Host sees the session page with the code prominently displayed and a shareable link
5. Session status is `active`

**Audience Flow — Join:**
1. Audience member opens the app or navigates to `/session/A1B2C3`
2. No login required — they see the session title and a "Joined" confirmation
3. Server issues an ephemeral UUID (stored in Redis with a TTL matching the session)
4. This ephemeral ID is stored client-side (cookie or localStorage) and used for all subsequent actions

**Tests:**
- [ ] Creating a session returns a response with a 6-character alphanumeric `code`
- [ ] The code is unique — creating 100 sessions produces 100 distinct codes
- [ ] The session appears in the host's dashboard session list
- [ ] `GET /v1/sessions/A1B2C3` returns session details (title, status, created_at) — no auth needed
- [ ] Visiting `/session/A1B2C3` in the browser shows the session title
- [ ] Visiting `/session/XXXXXX` (invalid code) shows a "Session not found" error
- [ ] The audience member receives an ephemeral UID (visible in Redis: key like `audience:{code}:{uid}` with TTL)
- [ ] The same browser tab re-joining the same session reuses the same ephemeral UID (not a new one)
- [ ] A different browser/incognito window joining the same session gets a DIFFERENT ephemeral UID
- [ ] The host's dashboard shows a list of all their sessions sorted by most recent
- [ ] Creating a session without authentication returns 401

---

## Phase 4 — Poll Creation & Management

**Goal:** A host can create polls within a session, configure answer mode, and control poll lifecycle (draft → active → closed).

**Prerequisite:** Phase 3

**Host Flow:**
1. Inside a session, host clicks "Create Poll"
2. Enters a question: "What is the time complexity of binary search?"
3. Adds 2–6 options: "O(1)", "O(log n)", "O(n)", "O(n log n)"
4. Chooses answer mode: Single answer or Multi-select
5. Optionally sets a time limit (e.g., 30 seconds)
6. Poll is created in `draft` status — not yet visible to audience
7. Host clicks "Activate" → poll goes `active` and appears on all audience screens
8. Host clicks "Close" → poll goes `closed`, no more votes accepted

**Tests:**
- [ ] Host can create a poll with 2 options — succeeds
- [ ] Host cannot create a poll with 1 option — returns validation error
- [ ] Host cannot create a poll with 7 options — returns validation error
- [ ] Poll is created with status `draft` by default
- [ ] Audience cannot see polls in `draft` status (GET /polls returns only active/closed)
- [ ] Activating a poll changes its status to `active`
- [ ] Closing a poll changes its status to `closed`
- [ ] A poll cannot go from `closed` back to `active`
- [ ] A poll cannot go from `closed` back to `draft`
- [ ] Poll options are returned in the correct `position` order
- [ ] A poll with `time_limit_sec: 30` — the frontend receives the limit value
- [ ] Only the session host can create/activate/close polls (other authenticated users get 403)
- [ ] Creating a poll in a non-existent session returns 404

---

## Phase 5 — Vote Casting

**Goal:** Audience members can vote on active polls. One vote per poll per user is enforced. Both single-answer and multi-select modes work correctly.

**Prerequisite:** Phase 4

**Audience Flow — Single Answer:**
1. An active poll appears on the audience screen with radio-button options
2. Audience member selects one option and clicks "Vote"
3. Vote is recorded — option highlight changes to show "You voted for X"
4. Attempting to vote again on the same poll shows "Already voted"

**Audience Flow — Multi-Select:**
1. An active poll appears with checkbox options
2. Audience member selects one or more options and clicks "Vote"
3. All selected options are recorded
4. Attempting to vote again on the same poll shows "Already voted"

**Tests:**
- [ ] Audience can submit a vote on an active poll — returns 200
- [ ] The `votes` table records the correct `poll_id`, `option_id`, and `audience_uid`
- [ ] Voting on the same poll a second time returns an error (duplicate vote)
- [ ] In single-answer mode, submitting 2 option IDs in one request returns a validation error
- [ ] In multi-select mode, submitting multiple option IDs succeeds
- [ ] Voting on a `draft` poll returns 400 ("Poll is not active")
- [ ] Voting on a `closed` poll returns 400 ("Poll is closed")
- [ ] Voting with an option ID that doesn't belong to the poll returns 400
- [ ] After voting, `GET /v1/sessions/:code/polls/:id` shows updated vote counts
- [ ] Two different audience members can both vote on the same poll (their UIDs differ)
- [ ] Vote counts are accurate: 3 votes for option A and 2 for option B → counts show `{A: 3, B: 2}`

---

## Phase 6 — WebSocket Connection & Room Management

**Goal:** Audience members connect to the realtime service via WebSocket. Connections are organized into rooms by session code. The hub manages client lifecycle.

**Prerequisite:** Phase 3 (needs session codes for room keying)

**Flow:**
1. When an audience member opens `/session/A1B2C3`, the frontend opens a WebSocket to `ws://localhost:8081/ws/A1B2C3`
2. The realtime service registers this client in room `A1B2C3`
3. If another user joins the same session, they join the same room
4. When a client disconnects (closes browser tab), they are removed from the room
5. Clients send a `ping` message every 30 seconds to keep the connection alive
6. The server drops clients that haven't pinged in 60 seconds

**Tests:**
- [ ] A WebSocket connection to `/ws/A1B2C3` is accepted and stays open
- [ ] A WebSocket connection to `/ws/XXXXXX` (invalid session) is rejected with an appropriate close code
- [ ] Opening 3 connections to `/ws/A1B2C3` → the room has 3 clients
- [ ] Closing 1 of those connections → the room has 2 clients
- [ ] Closing all connections → the room is cleaned up (memory freed)
- [ ] A client that stops sending pings is disconnected after the timeout
- [ ] The realtime service logs when clients connect and disconnect
- [ ] Connecting to a `closed`/`archived` session is rejected

---

## Phase 7 — Real-Time Vote Streaming

**Goal:** When a vote is cast, all connected clients in the session see the updated vote counts instantly — pushed via WebSocket, not polling.

**Prerequisite:** Phase 5 + Phase 6

**Flow:**
1. Audience member A casts a vote via `POST /v1/sessions/:code/polls/:id/vote`
2. The API service writes the vote to Postgres
3. After the DB commit, the API publishes a `vote_update` event to Redis channel `session:{code}`
4. The realtime service (subscribed to `session:{code}`) receives the event
5. The hub broadcasts a `vote_update` WebSocket message to ALL clients in room `{code}`
6. Every connected browser receives the push and updates the bar chart / vote count — no page refresh

**Event Payload:**
```json
{
  "type": "vote_update",
  "payload": {
    "pollId": "uuid",
    "options": [
      {"id": "uuid-opt-1", "count": 15},
      {"id": "uuid-opt-2", "count": 8},
      {"id": "uuid-opt-3", "count": 22}
    ]
  }
}
```

**Tests:**
- [ ] Client A votes → Client B (same session, different browser) sees updated counts within 1 second
- [ ] Client C (different session) does NOT receive the vote_update
- [ ] The vote_update payload contains ALL options with their CURRENT total counts (not just the delta)
- [ ] If 10 votes arrive rapidly, the frontend shows all 10 updates (no dropped events)
- [ ] Redis MONITOR shows the PUBLISH command on channel `session:A1B2C3` after each vote
- [ ] If no clients are connected to a room, the publish still succeeds (fire-and-forget pattern)
- [ ] After reconnecting a dropped WebSocket, the client can fetch current counts via REST as a fallback

---

## Phase 8 — Q&A Submission

**Goal:** Audience members can submit questions and comments to a live Q&A feed. Entries appear on the host's moderation panel in real time.

**Prerequisite:** Phase 6 + Phase 7 (needs WebSocket broadcasting)

**Audience Flow:**
1. In the session view, audience member sees a text input with a toggle: "Question" or "Comment"
2. Types their entry: "Can you explain Big-O notation again?"
3. Selects "Question" and submits
4. Entry appears in the Q&A feed immediately (for the submitter)
5. All other connected clients receive the entry via WebSocket push
6. The host's moderation panel shows the new entry at the top of the feed

**Host View:**
- Sees all questions and comments in real time
- Questions show a score (starts at 0) and action buttons
- Comments show as plain text with no actions

**Tests:**
- [ ] Submitting a question creates a `qa_entries` row with `entry_type = 'question'`
- [ ] Submitting a comment creates a row with `entry_type = 'comment'`
- [ ] Both types appear in the `GET /v1/sessions/:code/qa` response
- [ ] After submission, a `new_question` or `new_comment` WebSocket event is broadcast to the room
- [ ] All connected clients see the new entry appear without refreshing
- [ ] Empty body submissions are rejected (validation error)
- [ ] The `author_uid` on the entry matches the audience member's ephemeral UID
- [ ] Entries are returned sorted by score (descending), then by creation time (ascending)
- [ ] The Q&A feed uses cursor-based pagination — loading 20 entries at a time
- [ ] Loading more entries (scrolling) fetches the next page correctly

---

## Phase 9 — Q&A Voting (Upvote / Downvote)

**Goal:** Audience members can upvote or downvote questions (not comments). The score updates in real time for all viewers.

**Prerequisite:** Phase 8

**Flow:**
1. Audience member sees a question with score `5` and up/down arrows
2. Clicks upvote → score becomes `6`
3. If they click upvote again → their vote is removed (toggle behavior), score goes back to `5`
4. If they click downvote instead → their vote flips from +1 to -1, score goes from `6` to `4`
5. Score change broadcasts to all clients via `qa_update` WebSocket event

**Comments:** No upvote/downvote buttons shown. The UI enforces this — the API also rejects vote attempts on comments at the server level.

**Tests:**
- [ ] Upvoting a question creates a `qa_votes` row with `vote_value = 1`
- [ ] Downvoting creates a row with `vote_value = -1`
- [ ] Upvoting the same question twice → the second request removes the vote (toggle off)
- [ ] Switching from upvote to downvote → the `qa_votes` row updates from `1` to `-1`
- [ ] The `qa_entries.score` column is recalculated correctly: `SUM(vote_value)` across all qa_votes for that entry
- [ ] Attempting to vote on a comment returns 400
- [ ] After voting, a `qa_update` event broadcasts the new score to all room clients
- [ ] The Q&A feed re-sorts by score after each update (highest score at top)
- [ ] Two different audience members can both vote on the same question
- [ ] The UNIQUE constraint on `(qa_entry_id, voter_uid)` prevents database-level duplicates

---

## Phase 10 — Host Moderation Controls

**Goal:** The host can moderate the Q&A feed — mark as answered, pin, hide, or archive entries. All state changes broadcast in real time.

**Prerequisite:** Phase 8

**Host Actions:**

| Action         | Effect                                                        | Visual                       |
| -------------- | ------------------------------------------------------------- | ---------------------------- |
| Mark Answered  | Sets `status = 'answered'`                                    | Green "Answered" badge       |
| Pin            | Sets `status = 'pinned'`                                      | Pinned to top of feed        |
| Hide           | Sets `is_hidden = TRUE`                                       | Removed from audience view   |
| Unhide         | Sets `is_hidden = FALSE`                                      | Reappears in audience view   |
| Archive        | Sets `status = 'archived'`                                    | Removed from active feed     |

**Key Behavior:**
- `is_hidden` is independent of `status` — a question can be `answered` AND hidden
- Only the session host can moderate (enforced server-side)
- Every moderation action broadcasts a `qa_update` event

**Tests:**
- [ ] Host marks a question as "Answered" → status changes to `answered` in the database
- [ ] The audience sees a green "Answered" badge appear on that question in real time
- [ ] Host pins a question → it moves to the top of the feed for all viewers
- [ ] Host hides a question → it disappears from the audience view but remains in the host panel
- [ ] Host unhides a question → it reappears in the audience view
- [ ] Host hides a comment → it disappears from the audience view
- [ ] `is_hidden = TRUE` entries are NOT returned by `GET /v1/sessions/:code/qa` (audience endpoint)
- [ ] `is_hidden = TRUE` entries ARE visible in the host's moderation panel
- [ ] A non-host user sending a PATCH to moderate returns 403
- [ ] All moderation actions emit a `qa_update` WebSocket event with the updated entry state
- [ ] Archiving an entry removes it from the active feed but it remains in the database (`status = 'archived'`)
- [ ] The host can see archived entries in a separate "Archived" tab

---

## Phase 11 — Session Lifecycle & Auto-Archive

**Goal:** A host can close a session. Closing archives all Q&A, disables voting, and notifies all connected clients. Closed sessions remain queryable and exportable.

**Prerequisite:** Phase 10

**Host Flow — Close Session:**
1. Host clicks "End Session" on the session page
2. Confirmation dialog: "Are you sure? This will close the session for all participants."
3. Host confirms → API sets `sessions.status = 'archived'`, `sessions.closed_at = NOW()`
4. All Q&A entries for the session are bulk-updated to `status = 'archived'`
5. A `session_closed` event broadcasts to all connected WebSocket clients
6. Audience sees "This session has ended" — polls and Q&A are read-only
7. All WebSocket connections for that room are gracefully closed

**Post-Close:**
- The session page still loads (read-only) — useful for the host to review
- No new polls, votes, or Q&A submissions are accepted
- The host can still view results and export data from the dashboard

**Tests:**
- [ ] `PATCH /v1/sessions/:code/close` sets `status = 'archived'` and populates `closed_at`
- [ ] All `qa_entries` for the session now have `status = 'archived'`
- [ ] All connected WebSocket clients receive a `session_closed` event
- [ ] After closing, `POST /v1/sessions/:code/polls` returns 400 ("Session is archived")
- [ ] After closing, `POST /v1/sessions/:code/polls/:id/vote` returns 400 ("Session is archived")
- [ ] After closing, `POST /v1/sessions/:code/qa` returns 400 ("Session is archived")
- [ ] `GET /v1/sessions/:code` still works — returns session with `status: 'archived'`
- [ ] `GET /v1/sessions/:code/polls` still works — returns polls with their final vote counts
- [ ] The WebSocket connections for the closed session are terminated server-side
- [ ] New WebSocket connections to a closed session are rejected
- [ ] The host's dashboard shows the session with an "Archived" badge
- [ ] Only the session host can close the session (other users get 403)

---

## Phase 12 — Live Analytics Dashboard

**Goal:** The host sees a real-time dashboard showing vote distribution as bar charts, total votes, and participation percentage for each poll.

**Prerequisite:** Phase 7 (real-time vote streaming)

**Host View:**
1. Each active poll shows a horizontal bar chart with option labels and vote counts
2. Bars animate as votes come in (smooth transitions)
3. Below the chart: "42 votes cast · 78% participation" (participation = votes / connected audience)
4. Multiple polls can be active simultaneously — each has its own chart
5. Closed polls show final results (static chart)

**Audience View:**
- Before voting: sees options as buttons/radio/checkboxes
- After voting: sees the same bar chart as the host (their selection is highlighted)
- Charts update live as other votes come in

**Tests:**
- [ ] Host dashboard shows a bar chart for each active poll
- [ ] Vote counts in the chart match the database exactly
- [ ] When a new vote arrives via WebSocket, the chart updates within 1 second (no refresh)
- [ ] The bar widths are proportional to vote counts (10 vs 5 → bar A is twice as wide)
- [ ] "Total votes" counter increments with each vote
- [ ] A poll with zero votes shows empty bars (not an error state)
- [ ] Closed polls show final results as a static chart
- [ ] The audience sees the chart ONLY after they've voted (or the poll is closed)
- [ ] Multi-select polls show correct counts (each option's count is independent)
- [ ] The chart renders correctly with 2 options and with 6 options (min/max)

---

## Phase 13 — Export Results

**Goal:** The host can export poll results and Q&A archives as CSV or JSON files. Files are uploaded to cloud object storage and a time-limited download link is returned.

**Prerequisite:** Phase 11 (session close & archive)

**Host Flow:**
1. On the dashboard or session page, host clicks "Export Results"
2. Chooses format: CSV or JSON
3. Server generates the file containing:
   - All polls with their options and vote counts
   - All Q&A entries with scores and statuses
   - Session metadata (title, code, created/closed timestamps)
4. File is uploaded to S3 (or compatible object storage)
5. A presigned URL (valid for 15 minutes) is returned to the host
6. Host's browser automatically starts downloading the file

**Tests:**
- [ ] `GET /v1/sessions/:code/export?format=csv` returns a JSON response with a `download_url`
- [ ] `GET /v1/sessions/:code/export?format=json` also works
- [ ] The `download_url` is a valid presigned S3 URL
- [ ] Downloading the URL within 15 minutes succeeds — returns the file
- [ ] Downloading the URL after 15 minutes fails (expired)
- [ ] The CSV file contains correct headers and data for polls + Q&A
- [ ] The JSON file is valid JSON with nested poll → options → vote counts
- [ ] Exporting an active session works (not just archived ones)
- [ ] Only the session host can export (other users get 403)
- [ ] Exporting a session with no polls/Q&A returns a valid file with just metadata

---

## Phase 14 — Audience Identity & Vote Enforcement ∥

**Goal:** Harden the ephemeral audience identity system. Ensure one-vote-per-poll is enforced both client-side and server-side, even against spoofing attempts.

**Prerequisite:** Phase 5

**How It Works:**
1. When an audience member first joins a session, the API generates a UUID
2. This UUID is stored in Redis with a TTL matching the session duration
3. The UUID is returned to the client and stored (cookie or short-lived JWT)
4. Every vote and Q&A submission includes this UUID
5. The API validates the UUID exists in Redis before accepting the action
6. The UNIQUE constraint in Postgres is the final enforcement layer

**Tests:**
- [ ] Joining a session creates a Redis key `audience:{code}:{uid}` with a TTL
- [ ] The same browser re-joining the same session gets back the SAME UID (idempotent)
- [ ] A fabricated UID (not in Redis) is rejected by the API with 401
- [ ] An expired UID (TTL elapsed) is rejected
- [ ] Even if the client modifies their UID cookie, the Postgres UNIQUE constraint prevents double-voting
- [ ] Clearing cookies and re-joining produces a NEW UID — this is a fresh identity
- [ ] The ephemeral UID contains NO personally identifiable information
- [ ] After a session is closed, the associated Redis keys are cleaned up

---

## Phase 15 — Frontend Polish & UX

**Goal:** A cohesive, responsive, and accessible user interface. Smooth animations and clear feedback for all interactions.

**Prerequisite:** Phase 12 (all features built)

**Areas:**

**Landing Page:**
- Clear value proposition: "Create a live poll in seconds"
- "Create Session" button for hosts, "Join Session" input for audience (enter code)
- Clean, modern design with LivePulse branding

**Session View (Audience):**
- Mobile-first layout — most audience members are on phones
- Large, tap-friendly vote buttons
- Smooth chart animations when votes arrive
- Clear "You voted!" feedback
- Q&A feed with pull-to-refresh on mobile

**Dashboard (Host):**
- Session list with status badges (Active / Archived)
- Inside a session: tabs for Polls, Q&A, Settings
- Poll creation modal with a clean form
- Moderation panel with swipe-to-action on mobile

**Tests:**
- [ ] The app is usable on a 375px-wide screen (iPhone SE) — no horizontal scrolling
- [ ] Vote buttons are at least 44x44px (touch target guideline)
- [ ] After voting, the selected option shows a visual "selected" state within 200ms
- [ ] Chart bars animate smoothly when vote counts change (CSS transition or similar)
- [ ] Loading states are shown while waiting for API responses (spinners/skeletons)
- [ ] Error states are user-friendly: "Something went wrong. Try again." (not raw error codes)
- [ ] The session code is displayed in a large, copyable format on the host's session page
- [ ] "Join Session" input accepts codes in any case (a1b2c3 → A1B2C3)
- [ ] HTML is semantically correct — headings, labels, ARIA attributes
- [ ] Color contrast ratios meet WCAG AA (4.5:1 for text)

---

## Phase 16 — Containerized Deployment

**Goal:** All three services (web, api, realtime) run as Docker containers in production. Deployed on a cloud provider. Redis runs as a container sidecar. Postgres is a managed service.

**Prerequisite:** Phase 15

**Target Architecture:**
- DigitalOcean Droplet running Docker Compose (3 app containers + Redis)
- AWS RDS PostgreSQL (managed, free tier)
- AWS S3 for export file storage
- GitHub Actions CI/CD pipeline: build images → push to GHCR → deploy

**Steps:**
1. Configure `infra/docker-compose.prod.yml` with all 4 services
2. Environment variables injected via `.env` on the server (or Doppler later)
3. Set up GitHub Actions workflow: on push to `main` → build & push images
4. Set up DNS: `livepulse.app`, `api.livepulse.app`, `rt.livepulse.app`
5. Configure HTTPS (Let's Encrypt via Caddy or nginx-proxy)

**Tests:**
- [ ] `docker compose -f infra/docker-compose.prod.yml up` starts all services
- [ ] All 3 service health checks pass from outside the Docker network
- [ ] The Next.js app proxies API requests correctly to the Go service
- [ ] WebSocket connections work through the reverse proxy (upgrade headers pass through)
- [ ] Pushing to `main` triggers a GitHub Actions build — all images build successfully
- [ ] Images are pushed to GHCR and are pullable from the deployment server
- [ ] HTTPS works on all three domains (no mixed content warnings)
- [ ] Environment variables are NOT baked into Docker images (verified by inspecting image layers)
- [ ] Containers restart automatically if they crash (`restart: unless-stopped`)
- [ ] Database connection from the Droplet to RDS succeeds (security group configured)

---

## Phase 17 — Observability & Health Monitoring

**Goal:** Comprehensive logging, health monitoring, and error visibility across all services. Every request is traceable. Problems are detectable before users report them.

**Prerequisite:** Phase 16

**What's Already Done:**
- Structured JSON logging via slog (method, path, status, latency, request_id)
- `/healthz` endpoints on both Go services

**What to Add:**
- Request IDs propagated from Next.js → API → Realtime (pass `X-Request-ID` header)
- Log WebSocket events: connect, disconnect, broadcast, errors
- Health check includes DB connectivity and Redis connectivity (not just "service up")
- Optional: Datadog integration (free 2yr via GitHub Student Pack)

**Tests:**
- [ ] Every API request log includes `request_id`, `method`, `path`, `status`, `latency_ms`
- [ ] The same `request_id` appears in both Next.js and Go API logs for a proxied request
- [ ] WebSocket connect/disconnect events are logged with session code and client count
- [ ] `GET /healthz` checks Postgres connectivity — returns unhealthy if DB is down
- [ ] `GET /healthz` checks Redis connectivity — returns unhealthy if Redis is down
- [ ] Logs are JSON-formatted and parseable by any log aggregator
- [ ] Error logs include stack traces for panics (recovered by middleware)
- [ ] No sensitive data (passwords, JWTs, emails) appears in logs
- [ ] Container orchestrator (Docker) uses `/healthz` as liveness probe and restarts unhealthy containers
- [ ] A dashboard or log search can answer: "How many votes were cast in the last hour?"

---

## Phase 18 — Security Hardening & Rate Limiting

**Goal:** Protect the platform against common attacks. Rate-limit abuse-prone endpoints. Pass a basic security review.

**Prerequisite:** Phase 16

**Areas:**

**Rate Limiting:**
- Vote endpoint: 10 requests/minute per audience UID
- Q&A submission: 5 requests/minute per audience UID
- Session creation: 3 requests/minute per host
- Implemented via Redis token-bucket (INCR + EXPIRE)

**Input Validation:**
- All text inputs sanitized (HTML stripped from Q&A body)
- Session code: exactly 6 alphanumeric characters
- Poll question: max 500 characters
- Q&A body: max 2000 characters
- Poll options: max 200 characters each

**Auth & Sessions:**
- JWT expiry: 24 hours for host tokens
- Ephemeral audience UIDs: TTL matches session
- CORS configured to allow only the frontend origin
- HTTP-only, Secure, SameSite=Lax cookies

**Infrastructure:**
- Postgres credentials are not in image layers
- Redis is not exposed to the public internet (only internal Docker network)
- Helmet-style security headers on all responses

**Tests:**
- [ ] Sending 20 votes in 10 seconds → first 10 succeed, next 10 return 429 Too Many Requests
- [ ] Rate limit resets after the window expires
- [ ] Submitting `<script>alert('xss')</script>` as a Q&A body → stored as plain text (not executed)
- [ ] Submitting a Q&A body longer than 2000 characters → rejected with validation error
- [ ] CORS: API rejects requests from `http://evil-site.com` (wrong origin)
- [ ] CORS: API accepts requests from `http://localhost:3000` (correct origin)
- [ ] The `Set-Cookie` header includes `HttpOnly`, `Secure`, and `SameSite` flags
- [ ] JWT tokens expire after 24 hours — requests with expired tokens return 401
- [ ] Redis port 6379 is NOT accessible from outside the server (only via Docker network)
- [ ] `docker inspect livepulse/api:latest` → no credentials visible in environment or layers
- [ ] All responses include `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY`
- [ ] SQL injection attempt in session code (e.g., `' OR 1=1--`) → returns 404, no error leak

---

## Phase Summary

| #  | Phase                              | Depends On  | Focus Area        |
| -- | ---------------------------------- | ----------- | ----------------- |
| 1  | Project Foundation                 | —           | Infrastructure    |
| 2  | Host Authentication                | 1           | Auth              |
| 3  | Session Creation & Join            | 2           | Core Feature      |
| 4  | Poll Creation & Management         | 3           | Core Feature      |
| 5  | Vote Casting                       | 4           | Core Feature      |
| 6  | WebSocket Connection & Rooms       | 3           | Real-Time Infra   |
| 7  | Real-Time Vote Streaming           | 5, 6        | Real-Time Feature |
| 8  | Q&A Submission                     | 6, 7        | Core Feature      |
| 9  | Q&A Voting (Upvote/Downvote)       | 8           | Core Feature      |
| 10 | Host Moderation Controls           | 8           | Core Feature      |
| 11 | Session Lifecycle & Auto-Archive   | 10          | Core Feature      |
| 12 | Live Analytics Dashboard           | 7           | Frontend          |
| 13 | Export Results                      | 11          | Cloud Feature     |
| 14 | Audience Identity & Vote Enforce   | 5 ∥         | Security          |
| 15 | Frontend Polish & UX               | 12          | Frontend          |
| 16 | Containerized Deployment           | 15          | Cloud / DevOps    |
| 17 | Observability & Health Monitoring  | 16          | Operations        |
| 18 | Security Hardening & Rate Limiting | 16          | Security          |

**Parallelizable Pairs:**
- Phase 6 ∥ Phase 5 (WebSocket setup doesn't need votes)
- Phase 9 ∥ Phase 10 (Q&A voting and moderation are independent)
- Phase 14 ∥ any phase after 5 (identity hardening is orthogonal)
- Phase 17 ∥ Phase 18 (observability and security are independent)

---

## Suggested Sprint Groupings

| Sprint | Phases    | Theme                         | Estimated Duration |
| ------ | --------- | ----------------------------- | ------------------ |
| 1      | 2, 3      | Auth + Sessions               | —                  |
| 2      | 4, 5, 6   | Polls + Voting + WebSocket    | —                  |
| 3      | 7, 8      | Real-Time Streaming + Q&A     | —                  |
| 4      | 9, 10, 11 | Q&A Voting + Moderation + Lifecycle | —            |
| 5      | 12, 13, 14| Analytics + Export + Identity  | —                  |
| 6      | 15        | Frontend Polish               | —                  |
| 7      | 16, 17, 18| Deploy + Observe + Harden     | —                  |

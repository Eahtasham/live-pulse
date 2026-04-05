## Phase 16 — WebSocket Client & Live Updates 🔵 FRONTEND

**Goal:** The frontend connects to the realtime WebSocket service and updates polls, Q&A, and session state in real time without page refresh.

**Prerequisite:** Phase 14 + Phase 15 + Phase 9

### React hooks to build

**`useWebSocket(code: string)`:**
- Connects to `ws://localhost:8081/ws/{code}` when component mounts
- Auto-reconnect with exponential backoff (1s → 2s → 4s → ... → 30s max)
- Sends `ping` frame every 30 seconds
- Returns connection state (`connecting`, `connected`, `disconnected`)
- On reconnect, triggers REST fallback to re-fetch current state
- Cleans up on unmount (close connection)

**`usePollVotes(code: string)`:**
- Listens for `vote_update` events from WebSocket
- Updates local React state with new vote counts from the event payload
- The `ResultsChart` component re-renders with updated bars
- On reconnect: fetches `GET /v1/sessions/:code/polls` to sync state

**`useQAFeed(code: string)`:**
- Listens for `new_question`, `new_comment`, `qa_update` events
- `new_question` / `new_comment`: prepends new entry to feed state
- `qa_update`: updates existing entry (score, status, is_hidden) in place
- Feed re-sorts by score after each update
- On reconnect: fetches `GET /v1/sessions/:code/qa` to sync state

**`useSessionStatus(code: string)`:**
- Listens for `session_closed` event
- On close: sets `sessionEnded = true`, triggers an overlay on the session page
- Disables all inputs and submit buttons when session is ended

### WebSocket message format (from server)

```json
{
  "type": "vote_update",
  "payload": {
    "pollId": "uuid",
    "options": [
      {"id": "uuid", "label": "O(1)", "vote_count": 5},
      {"id": "uuid", "label": "O(log n)", "vote_count": 12}
    ]
  }
}
```

All messages are JSON with `type` (string) and `payload` (object).

### Connection state UI

- Small indicator badge in session header:
  - 🟢 Connected
  - 🟡 Reconnecting...
  - 🔴 Disconnected
- "Session ended" overlay when `session_closed` received

### REST fallback on reconnect

After a WebSocket reconnection, the client should NOT rely on receiving missed events. Instead:
1. Fetch `GET /v1/sessions/:code/polls` → fully replace poll state
2. Fetch `GET /v1/sessions/:code/qa` → fully replace Q&A state
3. This ensures no stale data after a network interruption

### Acceptance tests

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

### Files to create/modify

- `apps/web/hooks/useWebSocket.ts` — Core WebSocket hook
- `apps/web/hooks/usePollVotes.ts` — Poll vote subscription
- `apps/web/hooks/useQAFeed.ts` — Q&A feed subscription
- `apps/web/hooks/useSessionStatus.ts` — Session close detection
- `apps/web/components/session/ConnectionIndicator.tsx` — Status badge
- `apps/web/components/session/SessionEndedOverlay.tsx`
- Update `PollCard`, `ResultsChart`, `QAFeed` to accept live data from hooks

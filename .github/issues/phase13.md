## Phase 13 — Session UI 🔵 FRONTEND

**Goal:** Host can create a session from the dashboard. Audience can join via code input or direct URL. Session page shows the live session view.

**Prerequisite:** Phase 12 + Phase 3

### User flows

**Host — Create session:**
1. Click "New Session" on dashboard
2. Modal appears with title input
3. Submit → calls `POST /v1/sessions` → session created
4. Modal shows the generated code prominently + shareable link + copy button
5. Host clicks "Go to Session" → navigates to `/session/[code]`
6. New session immediately appears in dashboard list

**Audience — Join via URL:**
1. Open `/session/A1B2C3` (shared link)
2. Frontend calls `POST /v1/sessions/A1B2C3/join` automatically on page load
3. Receives ephemeral UID, stored in localStorage or cookie
4. Session title displayed, "You've joined" confirmation
5. Session page shows polls + Q&A (built in Phases 14–15)

**Audience — Join via code input (landing page):**
1. Visit `/` (landing page)
2. Type a session code into the "Join Session" input
3. Input auto-uppercases and validates (6 alphanumeric chars)
4. Press Enter or click "Join" → navigate to `/session/A1B2C3`

### Components to build

- `CreateSessionModal` — Title input, submit button, code display + copy
- `JoinSessionInput` — Code input with validation, uppercase transform
- `SessionHeader` — Session title, code badge, status indicator, participant count
- `SessionPage` — Layout for host vs audience views (tabs for Polls / Q&A)

### Host vs Audience view detection

The session page at `/session/[code]` needs to know if the viewer is the host or an audience member:
- If the user has a valid JWT AND their `user_id` matches `session.host_id` → render host view (with controls)
- Otherwise → render audience view
- The `GET /v1/sessions/:code` response should include `host_id` to enable this check client-side

### Acceptance tests

- [ ] Clicking "New Session" → modal appears with title input
- [ ] Entering a title and submitting → session is created, code is displayed prominently
- [ ] The session code can be copied to clipboard with one click
- [ ] A shareable link (`/session/A1B2C3`) is shown
- [ ] Navigating to `/session/A1B2C3` as an audience member → shows session title, "You've joined" confirmation
- [ ] Navigating to `/session/XXXXXX` → shows "Session not found" error page
- [ ] The landing page has a "Join Session" input — entering a code navigates to `/session/:code`
- [ ] Code input is case-insensitive: entering `a1b2c3` navigates to `/session/A1B2C3`
- [ ] New session appears in the dashboard list immediately after creation

### Files to create/modify

- `apps/web/app/page.tsx` — Landing page with join input
- `apps/web/app/session/[code]/page.tsx` — Session page
- `apps/web/components/session/CreateSessionModal.tsx`
- `apps/web/components/session/JoinSessionInput.tsx`
- `apps/web/components/session/SessionHeader.tsx`
- `apps/web/hooks/useAudienceUid.ts` — Join session + persist UID

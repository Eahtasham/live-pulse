## Phase 12 — Auth UI & Dashboard 🔵 FRONTEND

**Goal:** Host can sign in with Google, see a dashboard listing their sessions, and sign out.

**Prerequisite:** Phase 2 + Phase 3 (backend auth + session API must exist)

### Pages

| Path | Access | Description |
|------|--------|-------------|
| `/login` | Public | "Sign in with Google" button, redirect to dashboard on success |
| `/dashboard` | Protected | Session list, "New Session" button, sign out |

### Auth flow (Next.js side)

1. Configure **Auth.js v5** (NextAuth) with Google OAuth provider
2. On successful Google sign-in, call `POST /v1/auth/callback` on the Go API to get a JWT
3. Store the JWT in an HTTP-only cookie (managed by Auth.js session)
4. Include the JWT in the `Authorization: Bearer <token>` header on subsequent API requests
5. Use Auth.js middleware to protect `/dashboard` — redirect to `/login` if unauthenticated

### Environment variables needed

```env
NEXTAUTH_URL=http://localhost:3000
NEXTAUTH_SECRET=<random-string>
GOOGLE_CLIENT_ID=<from-google-cloud-console>
GOOGLE_CLIENT_SECRET=<from-google-cloud-console>
```

### Dashboard features

- **Session list**: Fetched from `GET /v1/sessions` with the host's JWT
- **Sort**: Most recent first (`created_at DESC`)
- **Each card shows**: title, 6-char code, status badge (Active green / Archived gray), created date
- **"New Session" button**: Opens create modal (Phase 13)
- **Sign out button**: Clears Auth.js session + cookie, redirects to `/login`
- **Empty state**: "No sessions yet. Create your first one!" with a CTA button

### UI components to build

- `LoginButton` — Google OAuth trigger
- `SessionCard` — Title, code, status badge, date
- `SessionList` — Maps sessions to cards
- `DashboardLayout` — Header with user avatar + sign out

### Acceptance tests

- [ ] Clicking "Sign in with Google" redirects to Google consent screen
- [ ] After consent, user lands on `/dashboard` (not `/login`)
- [ ] Refreshing `/dashboard` keeps user logged in (cookie persists)
- [ ] The dashboard shows a list of the host's sessions (fetched from `GET /v1/sessions`)
- [ ] Sessions are sorted by most recent first
- [ ] Each session shows: title, code, status badge (Active / Archived), created date
- [ ] Visiting `/dashboard` without auth → redirected to `/login`
- [ ] Clicking "Sign out" → clears session, redirects to `/login`
- [ ] After sign out, `/dashboard` redirects to `/login`

### Files to create/modify

- `apps/web/app/(auth)/login/page.tsx` — Login page with Google button
- `apps/web/app/dashboard/page.tsx` — Dashboard page
- `apps/web/components/session/SessionCard.tsx`
- `apps/web/components/session/SessionList.tsx`
- `apps/web/lib/auth.ts` — Auth.js config (Google provider)
- `apps/web/middleware.ts` — Route protection
- `apps/web/lib/api.ts` — API client with JWT header injection

### Dependencies to add

- `next-auth` (v5 / Auth.js)
- `@auth/core`

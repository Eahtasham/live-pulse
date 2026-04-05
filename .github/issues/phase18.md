## Phase 18 — Audience Identity Hardening 🟢 BACKEND

**Goal:** Harden the ephemeral audience identity system against spoofing. Ensure server-side enforcement is airtight with defense in depth.

**Prerequisite:** Phase 5 (can be done in parallel with frontend phases)

### Current state (what Phase 3 built)

- Audience joins via `POST /v1/sessions/:code/join` → receives a bare UUID
- UUID stored in Redis with TTL
- UUID sent with every vote and Q&A submission
- Redis lookup verifies the UID exists

### What this phase adds

**Audience JWT (instead of bare UUID):**
- `POST /v1/sessions/:code/join` now returns a **signed JWT** (not a bare UUID)
- The JWT payload contains:
  ```json
  {
    "uid": "uuid-v4",
    "session_code": "A1B2C3",
    "type": "audience",
    "exp": 1717200000  // 24h from issuance
  }
  ```
- Signed with the same `JWT_SECRET` used for host JWTs
- The `type: "audience"` claim distinguishes it from host JWTs

**Validation on every request:**
1. Parse the audience JWT from the request (body field or header)
2. Verify signature — reject if tampered
3. Verify `type == "audience"` — reject if it's a host JWT being misused
4. Verify `session_code` matches the URL path `:code` — reject cross-session reuse
5. Verify not expired — reject if TTL elapsed
6. Verify UID exists in Redis — reject if session was cleaned up
7. The Postgres UNIQUE constraint is the final enforcement layer

**Three layers of defense:**
| Layer | What it catches |
|-------|----------------|
| JWT signature | Fabricated/tampered UIDs |
| JWT `session_code` claim | Cross-session reuse attacks |
| Redis TTL lookup | Expired/cleaned-up sessions |
| Postgres UNIQUE constraint | Race conditions, double-voting |

### Privacy

- The audience JWT contains NO personally identifiable information
- No IP addresses, browser fingerprints, or tracking data stored
- Clearing cookies produces a fresh identity (by design — anonymous system)

### Acceptance tests

- [ ] A fabricated UID (valid UUID format but not issued by the server) → 401
- [ ] An expired audience JWT → 401
- [ ] A valid audience JWT for session A used on session B → 401 (session-scoped)
- [ ] After a session is closed, audience Redis keys are cleaned up (or left to TTL-expire)
- [ ] The audience JWT contains NO personally identifiable information (just uid + session code)
- [ ] Even if someone extracts the UID from the JWT, the Postgres UNIQUE constraint prevents double-voting
- [ ] Clearing cookies and re-joining → new UID issued (fresh identity — acceptable for anonymous system)

### Files to modify

- `apps/api/internal/handler/session.go` — Modify join handler to return audience JWT
- `apps/api/internal/middleware/audience.go` — New middleware for audience JWT validation
- `apps/api/internal/handler/vote.go` — Use audience middleware
- `apps/api/internal/handler/qa.go` — Use audience middleware
- `apps/api/internal/handler/qa_vote.go` — Use audience middleware

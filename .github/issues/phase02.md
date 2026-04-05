## Phase 2 — Auth Backend 🟢 BACKEND

**Goal:** The Go API can validate JWTs, extract user identity, and protect routes. Users table is populated on first login.

**Prerequisite:** Phase 1 ✅

### Scope (backend only — NO login UI)

- **JWT validation middleware** (`internal/middleware/auth.go`): Parse the `Authorization: Bearer <token>` header, validate signature using `JWT_SECRET`, extract `user_id` and `email` claims, and inject them into the request context.
- **Shared secret**: Auth.js (Next.js) and the Go API share `JWT_SECRET` for signing/verification. The Next.js side signs a JWT after successful OAuth; the Go API only verifies.
- **`POST /v1/auth/callback`**: Receives `{email, name, avatar_url, provider}` from the OAuth flow. Creates a `users` row if one doesn't exist for that email, or finds the existing one. Returns a signed JWT containing `user_id` and `email`.
- **Protected route group**: `POST /v1/sessions`, `PATCH /v1/sessions/*`, `DELETE /v1/sessions/*` require a valid JWT.
- **Public route group**: `GET /v1/sessions/:code`, vote endpoints, Q&A submission endpoints remain open (no JWT needed).
- **JWT expiry**: 24 hours by default, configurable via `JWT_EXPIRY` env var.

### Implementation notes

- Use `github.com/golang-jwt/jwt/v5` for JWT parsing and validation
- Use GORM to find/create user: `db.Where(User{Email: email}).FirstOrCreate(&user)`
- The middleware should set `ctx.Value("user_id")` and `ctx.Value("user_email")` for downstream handlers
- Return a structured JSON error on all auth failures: `{"error": "unauthorized", "message": "..."}`
- Do NOT return different error messages for "invalid signature" vs "expired" vs "malformed" to avoid information leakage to attackers

### How to test without frontend

```bash
# Generate a test JWT (using jwt.io or a Go test helper)
# Then:
curl -X POST http://localhost:8080/v1/sessions \
  -H "Authorization: Bearer <valid-jwt>" \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Session"}'

# Should return 401:
curl -X POST http://localhost:8080/v1/sessions
```

### Acceptance tests

- [ ] `POST /v1/sessions` without an Authorization header → 401 Unauthorized
- [ ] `POST /v1/sessions` with a valid JWT → 200 (or 201)
- [ ] `POST /v1/sessions` with an expired JWT → 401 (not 500)
- [ ] `POST /v1/sessions` with a malformed JWT → 401
- [ ] `GET /v1/sessions/:code` without a JWT → 200 (public route, no auth needed)
- [ ] `POST /v1/auth/callback` with `{email, name, avatar_url, provider}` → creates a `users` row and returns a JWT
- [ ] Calling callback again with the same email → does NOT create a duplicate user, returns JWT for existing user
- [ ] The JWT payload contains `user_id` and `email` — extractable in handlers via `r.Context().Value("user_id")`
- [ ] The JWT expires after 24 hours (configurable via `JWT_EXPIRY` env var)
- [ ] Invalid JWT signatures (wrong secret) are rejected with 401

### Files to modify/create

- `apps/api/internal/middleware/auth.go` — JWT validation middleware
- `apps/api/internal/handler/auth.go` — Auth callback handler
- `apps/api/internal/router/router.go` — Add protected route groups
- `apps/api/go.mod` — Add `golang-jwt/jwt/v5` dependency
- `apps/api/internal/config/config.go` — Add `JWT_SECRET`, `JWT_EXPIRY` config fields

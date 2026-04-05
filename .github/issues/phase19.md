## Phase 19 — Security Hardening & Rate Limiting 🟢 BACKEND

**Goal:** Protect against common attacks. Rate-limit abuse-prone endpoints. Pass a basic security review checklist.

**Prerequisite:** Phase 10

### Rate limiting (Redis token-bucket)

Implement rate limiting using Redis `INCR` + `EXPIRE` (token bucket pattern):

| Endpoint | Limit | Key pattern |
|----------|-------|-------------|
| `POST .../vote` | 10 req/min | `rl:vote:{audience_uid}` |
| `POST .../qa` | 5 req/min | `rl:qa:{audience_uid}` |
| `POST /v1/sessions` | 3 req/min | `rl:session:{user_id}` |
| `POST .../qa/:id/vote` | 20 req/min | `rl:qa_vote:{audience_uid}` |

**Response on limit exceeded:**
```json
HTTP 429 Too Many Requests
Retry-After: 45

{"error": "rate_limited", "message": "Too many requests. Try again in 45 seconds."}
```

### Rate limiter middleware

```go
// Pseudocode for Redis rate limiter
func RateLimit(rdb *redis.Client, prefix string, limit int, window time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := prefix + ":" + extractIdentifier(r)
            count := rdb.Incr(ctx, key)
            if count == 1 { rdb.Expire(ctx, key, window) }
            if count > limit {
                ttl := rdb.TTL(ctx, key)
                w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))
                http.Error(w, "Too many requests", 429)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Input validation (tighten existing endpoints)

| Field | Constraint |
|-------|-----------|
| Session code | Exactly 6 chars, `^[A-Z0-9]{6}$` regex |
| Poll question | Max 500 characters |
| Poll option label | Max 200 characters |
| Q&A body | Max 2000 characters, HTML tags stripped via `bluemonday` or `strings.ReplaceAll` |
| Options count | 2–6 per poll |
| `answer_mode` | Enum: `"single"` or `"multi"` |
| `entry_type` | Enum: `"question"` or `"comment"` |
| `vote_value` | Must be `1` or `-1` |

### Security headers middleware

Add to all API responses:
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 0
Referrer-Policy: strict-origin-when-cross-origin
```

### CORS configuration

```go
cors := cors.New(cors.Options{
    AllowedOrigins:   []string{os.Getenv("FRONTEND_URL")},  // e.g., "http://localhost:3000"
    AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Client-ID"},
    AllowCredentials: true,
    MaxAge:           86400,
})
```

### Cookie security

When setting cookies (if applicable):
```go
http.SetCookie(w, &http.Cookie{
    Name:     "session",
    HttpOnly: true,
    Secure:   true,      // HTTPS only in production
    SameSite: http.SameSiteLaxMode,
    Path:     "/",
    MaxAge:   86400,
})
```

### Error response sanitization

- Never expose stack traces, file paths, or internal error messages to clients
- All error responses use structured JSON: `{"error": "...", "message": "..."}`
- Log the full error server-side (slog) with request_id for correlation
- GORM errors are caught and mapped to generic messages

### Acceptance tests

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

### Files to create/modify

- `apps/api/internal/middleware/ratelimit.go` — Redis rate limiter middleware
- `apps/api/internal/middleware/security.go` — Security headers middleware
- `apps/api/internal/middleware/cors.go` — CORS configuration
- `apps/api/internal/router/router.go` — Apply rate limits to specific routes
- All handlers — Review and tighten input validation

### Dependencies to add

- `github.com/rs/cors` (or implement manually with chi middleware)

## Phase 6 — Q&A API 🟢 BACKEND

**Goal:** Full CRUD for Q&A entries. Audience submits questions/comments. Host moderates (answer, pin, hide, archive).

**Prerequisite:** Phase 3

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/sessions/:code/qa` | Audience UID | Submit a question or comment |
| `GET` | `/v1/sessions/:code/qa` | Public | List active entries (cursor-paginated) |
| `PATCH` | `/v1/sessions/:code/qa/:id` | Host JWT | Moderate: answer, pin, hide, unhide, archive |

### Request/Response examples

**Submit entry:**
```json
// POST /v1/sessions/A1B2C3/qa
{
  "entry_type": "question",
  "body": "Can you explain Big-O notation again?",
  "audience_uid": "uuid"
}

// Response (201):
{
  "id": "uuid",
  "entry_type": "question",
  "body": "Can you explain Big-O notation again?",
  "score": 0,
  "status": "active",
  "is_hidden": false,
  "author_uid": "uuid",
  "created_at": "..."
}
```

**List entries (cursor-paginated):**
```
GET /v1/sessions/A1B2C3/qa?limit=20&cursor=base64-encoded-cursor
```
- Default limit: 20, max limit: 50
- Sorted by `score DESC`, then `created_at ASC`
- Hidden entries (`is_hidden = TRUE`) are excluded from audience responses
- Cursor encodes the last entry's `(score, created_at, id)` tuple

**Moderate entry:**
```json
// PATCH /v1/sessions/A1B2C3/qa/{entry-id}
{"action": "answer"}   // Sets status = 'answered'
{"action": "pin"}       // Sets status = 'pinned'
{"action": "hide"}      // Sets is_hidden = true
{"action": "unhide"}    // Sets is_hidden = false
{"action": "archive"}   // Sets status = 'archived'
```

### Moderation rules

| Action | Effect | Notes |
|--------|--------|-------|
| `answer` | `status = 'answered'` | Shows green badge |
| `pin` | `status = 'pinned'` | Entry sorted to top |
| `hide` | `is_hidden = TRUE` | Independent of `status` — a question can be answered AND hidden |
| `unhide` | `is_hidden = FALSE` | Restores visibility |
| `archive` | `status = 'archived'` | Removed from active feed |

### Validation

- `entry_type`: must be `"question"` or `"comment"`
- `body`: required, 1–2000 characters, HTML tags stripped on input
- `audience_uid`: validated against Redis
- Only the session host can moderate (PATCH) — verified via JWT `user_id` vs `sessions.host_id`
- Cannot submit Q&A to an archived session

### Acceptance tests

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

### Files to create/modify

- `apps/api/internal/handler/qa.go` — Q&A handlers
- `apps/api/internal/service/qa.go` — Q&A business logic, pagination, moderation
- `apps/api/internal/router/router.go` — Register Q&A routes

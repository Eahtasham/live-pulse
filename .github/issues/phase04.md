## Phase 4 — Poll API 🟢 BACKEND

**Goal:** Full CRUD for polls within a session. Hosts create polls with options, manage lifecycle (draft → active → closed).

**Prerequisite:** Phase 3

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/sessions/:code/polls` | Host JWT | Create a new poll with 2-6 options |
| `GET` | `/v1/sessions/:code/polls` | Public | List polls (audience sees only active + closed) |
| `GET` | `/v1/sessions/:code/polls/:id` | Public | Get poll details with vote counts per option |
| `PATCH` | `/v1/sessions/:code/polls/:id` | Host JWT | Update poll status (activate / close) |

### Poll lifecycle state machine

```
draft  ──→  active  ──→  closed
  │                        ✗ (no going back)
  └──→ (delete while draft — optional)
```

- Polls are created in `draft` status by default
- Only forward transitions are allowed: `draft → active → closed`
- Reverse transitions (`closed → active`, `closed → draft`) return 400

### Request/Response examples

**Create poll:**
```json
// POST /v1/sessions/A1B2C3/polls
{
  "question": "What is the time complexity of binary search?",
  "answer_mode": "single",
  "time_limit_sec": 30,
  "options": [
    {"label": "O(1)", "position": 0},
    {"label": "O(log n)", "position": 1},
    {"label": "O(n)", "position": 2},
    {"label": "O(n log n)", "position": 3}
  ]
}

// Response (201):
{
  "id": "uuid",
  "question": "What is the time complexity of binary search?",
  "answer_mode": "single",
  "status": "draft",
  "time_limit_sec": 30,
  "options": [
    {"id": "uuid", "label": "O(1)", "position": 0, "vote_count": 0},
    {"id": "uuid", "label": "O(log n)", "position": 1, "vote_count": 0},
    ...
  ]
}
```

### Validation rules

- `question`: required, max 500 characters
- `options`: 2–6 items required
- `option.label`: required, max 200 characters
- `answer_mode`: must be `"single"` or `"multi"`
- `time_limit_sec`: optional, positive integer if provided
- Only the session host (verified by JWT `user_id` matching `sessions.host_id`) can create/modify polls
- Cannot create polls in an archived session

### Acceptance tests

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

### Files to create/modify

- `apps/api/internal/handler/poll.go` — Poll CRUD handlers
- `apps/api/internal/service/poll.go` — Poll business logic + validation
- `apps/api/internal/router/router.go` — Register poll routes

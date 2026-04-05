## Phase 5 — Vote API 🟢 BACKEND

**Goal:** Audience members can cast votes on active polls. One-vote-per-poll enforcement. Both single and multi-select modes work correctly.

**Prerequisite:** Phase 4

### Endpoint

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/sessions/:code/polls/:id/vote` | Audience UID | Cast vote(s) on an active poll |

### Request/Response

```json
// POST /v1/sessions/A1B2C3/polls/{poll-id}/vote
{
  "option_ids": ["uuid-of-option"],
  "audience_uid": "uuid-of-audience-member"
}

// Response (200):
{
  "message": "Vote recorded",
  "poll_id": "uuid",
  "options": [
    {"id": "uuid", "label": "O(1)", "vote_count": 3},
    {"id": "uuid", "label": "O(log n)", "vote_count": 15},
    ...
  ]
}
```

### Business logic

1. Validate the `audience_uid` exists in Redis (`audience:{code}:{uid}`) — reject with 401 if not found
2. Validate the poll exists and belongs to the session — 404 if not
3. Validate the poll is `active` — 400 if `draft` or `closed`
4. Validate all `option_ids` belong to this poll — 400 if any don't
5. Enforce answer mode:
   - `single`: only 1 option_id allowed → 400 if more than 1
   - `multi`: 1 or more option_ids allowed
6. Check for duplicate vote: `UNIQUE(poll_id, audience_uid, option_id)` — return 409 Conflict if already voted
7. Insert vote row(s) into `votes` table
8. Return updated vote counts for all options in the poll

### Vote counting

- Vote counts are computed as `COUNT(*)` grouped by `option_id` for the given `poll_id`
- In multi-select mode, each selected option creates a separate `votes` row
- The response always returns ALL options with counts (not just the one voted for)

### Acceptance tests

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

### Files to create/modify

- `apps/api/internal/handler/vote.go` — Vote handler
- `apps/api/internal/service/vote.go` — Vote business logic + validation
- `apps/api/internal/router/router.go` — Register vote route

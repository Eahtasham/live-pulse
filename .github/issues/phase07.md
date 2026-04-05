## Phase 7 — Q&A Vote API 🟢 BACKEND

**Goal:** Upvote/downvote system for questions. Toggle behavior (vote again to remove). Score recalculation on the `qa_entries` table.

**Prerequisite:** Phase 6

### Endpoint

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/sessions/:code/qa/:id/vote` | Audience UID | Upvote or downvote a question |

### Request/Response

```json
// POST /v1/sessions/A1B2C3/qa/{entry-id}/vote
{
  "audience_uid": "uuid",
  "value": 1    // 1 = upvote, -1 = downvote
}

// Response (200):
{
  "entry_id": "uuid",
  "new_score": 5,
  "your_vote": 1   // null if toggled off
}
```

### Business logic — Toggle/Upsert behavior

1. Validate `audience_uid` in Redis
2. Validate the entry is a `question` (not `comment`) — return 400 if comment
3. Validate `value` is `1` or `-1` — return 400 otherwise
4. Check for existing `qa_votes` row for `(qa_entry_id, voter_uid)`:
   - **No existing vote** → INSERT new row with `vote_value`
   - **Same value exists** (e.g., upvote when already upvoted) → DELETE the row (toggle off)
   - **Opposite value exists** (e.g., downvote when upvoted) → UPDATE `vote_value` to new value (flip)
5. Recalculate `qa_entries.score` = `SELECT COALESCE(SUM(vote_value), 0) FROM qa_votes WHERE qa_entry_id = ?`
6. Update the `qa_entries.score` column
7. Return the new score and the user's current vote state

### Score examples

| Action | qa_votes rows | Score |
|--------|--------------|-------|
| User A upvotes | `{A: +1}` | 1 |
| User B upvotes | `{A: +1, B: +1}` | 2 |
| User C downvotes | `{A: +1, B: +1, C: -1}` | 1 |
| User A upvotes again (toggle) | `{B: +1, C: -1}` | 0 |
| User A downvotes | `{A: -1, B: +1, C: -1}` | -1 |

### Database constraint

`UNIQUE(qa_entry_id, voter_uid)` in the `qa_votes` table prevents duplicate rows at the DB level as a safety net.

### Acceptance tests

- [ ] Upvote a question → `qa_votes` row with `vote_value = 1`
- [ ] Downvote → row with `vote_value = -1`
- [ ] Upvote same question a second time → removes the vote (toggle off)
- [ ] Upvote then downvote → row updates from `1` to `-1` (upsert)
- [ ] `qa_entries.score` = `SUM(vote_value)` from all `qa_votes` for that entry
- [ ] 3 upvotes + 1 downvote → score = 2
- [ ] Attempting to vote on a comment → 400 ("Cannot vote on comments")
- [ ] `UNIQUE(qa_entry_id, voter_uid)` prevents DB-level duplicates
- [ ] Two different audience members can both vote on the same question
- [ ] After voting, `GET /qa` returns entries with updated scores, re-sorted

### Files to create/modify

- `apps/api/internal/handler/qa_vote.go` — Q&A vote handler
- `apps/api/internal/service/qa_vote.go` — Toggle/upsert logic + score recalculation
- `apps/api/internal/router/router.go` — Register Q&A vote route

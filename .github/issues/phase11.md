## Phase 11 — Export API 🟢 BACKEND

**Goal:** Host can export session results (polls + Q&A) as CSV or JSON. File is uploaded to AWS S3 and a presigned download URL is returned.

**Prerequisite:** Phase 10

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/v1/sessions/:code/export?format=csv` | Host JWT | Export results as CSV |
| `GET` | `/v1/sessions/:code/export?format=json` | Host JWT | Export results as JSON |

### Export flow

1. Validate caller is session host
2. Query all data for the session:
   - Session metadata (title, code, status, created_at, closed_at)
   - All polls with their options and vote counts
   - All Q&A entries with scores, statuses, and vote counts
3. Generate file in requested format (CSV or JSON)
4. Upload to S3 bucket: `s3://{S3_BUCKET}/exports/{session-code}/{timestamp}.{format}`
5. Generate presigned URL with 15-minute expiry
6. Return `{download_url: "https://..."}` to client

### CSV format

```csv
# Session: CS101 Lecture 5
# Code: A1B2C3
# Status: archived
# Created: 2026-04-05T10:00:00Z
# Closed: 2026-04-05T11:30:00Z

## Polls
poll_question,option_label,vote_count,answer_mode
"What is Big-O?","O(1)",3,single
"What is Big-O?","O(log n)",15,single
"What is Big-O?","O(n)",8,single

## Q&A
entry_type,body,score,status,is_hidden,created_at
question,"Can you explain recursion?",12,answered,false,2026-04-05T10:15:00Z
comment,"Great lecture!",0,active,false,2026-04-05T10:20:00Z
```

### JSON format

```json
{
  "session": {
    "title": "CS101 Lecture 5",
    "code": "A1B2C3",
    "status": "archived",
    "created_at": "...",
    "closed_at": "..."
  },
  "polls": [
    {
      "question": "What is Big-O?",
      "answer_mode": "single",
      "status": "closed",
      "total_votes": 26,
      "options": [
        {"label": "O(1)", "vote_count": 3},
        {"label": "O(log n)", "vote_count": 15}
      ]
    }
  ],
  "qa_entries": [
    {"entry_type": "question", "body": "...", "score": 12, "status": "answered"}
  ]
}
```

### AWS S3 configuration

- `S3_BUCKET`, `S3_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` from env
- Use `aws-sdk-go-v2` for S3 PutObject and presigned URL generation
- Presigned URL expires in 15 minutes (configurable)

### Acceptance tests

- [ ] Export as CSV → response contains `{download_url: "https://s3.../..."}`
- [ ] Export as JSON → same response structure
- [ ] The `download_url` is a valid presigned S3 URL
- [ ] Downloading within 15 minutes → file downloads successfully
- [ ] Downloading after 15 minutes → access denied (expired)
- [ ] CSV contains headers: session info, poll questions with options and vote counts, Q&A entries with scores
- [ ] JSON is valid and contains nested structure: session → polls → options → counts + qa_entries
- [ ] Exporting an active session works (not just archived)
- [ ] Exporting an empty session (no polls, no Q&A) → valid file with just session metadata
- [ ] Only the session host can export → 403 for others
- [ ] Invalid format parameter (`?format=xml`) → 400

### Dependencies to add

- `github.com/aws/aws-sdk-go-v2` and related sub-packages

### Files to create/modify

- `apps/api/internal/handler/export.go` — Export handler
- `apps/api/internal/service/export.go` — Data aggregation, CSV/JSON generation
- `apps/api/internal/service/s3.go` — S3 upload + presigned URL generation
- `apps/api/internal/config/config.go` — Add S3 config fields
- `apps/api/internal/router/router.go` — Register export route

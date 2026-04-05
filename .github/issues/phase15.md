## Phase 15 — Q&A UI 🔵 FRONTEND

**Goal:** Audience can submit questions/comments. Host has a moderation panel with action buttons. Upvote/downvote on questions.

**Prerequisite:** Phase 13 + Phase 6 + Phase 7

### Audience view

**Submit form:**
- Text input area with character counter (max 2000)
- Toggle switch: "Question" (default) / "Comment"
- Submit button — calls `POST /v1/sessions/:code/qa`
- After submit, entry appears immediately (optimistic UI)

**Q&A feed:**
- Entries sorted by score (highest first), then creation time
- Each entry shows: body text, entry type badge, score, relative timestamp
- **Questions**: Show upvote (▲) and downvote (▼) arrows flanking the score
  - Clicking upvote when not voted → upvote (score +1, arrow highlighted)
  - Clicking upvote again → remove vote (toggle off, score -1)
  - Clicking downvote when upvoted → flip (score -2)
- **Comments**: No vote arrows shown
- Answered questions show a green "Answered" badge
- Pinned questions stick to the top
- "Load more" button at bottom for cursor pagination (20 entries per page)

### Host view — Moderation panel

Same feed as audience, plus action buttons per entry:

| Button | API Call | Visual Effect |
|--------|----------|--------------|
| ✓ Answer | `PATCH /qa/:id {action: "answer"}` | Green "Answered" badge |
| 📌 Pin | `PATCH /qa/:id {action: "pin"}` | Entry pinned to top |
| 👁 Hide | `PATCH /qa/:id {action: "hide"}` | "[Hidden]" label, dimmed. Hidden from audience |
| 👁 Unhide | `PATCH /qa/:id {action: "unhide"}` | Restored to normal |
| 🗄 Archive | `PATCH /qa/:id {action: "archive"}` | Moved to "Archived" tab |

- Hidden entries are visible to host (with "[Hidden]" indicator) but NOT to audience
- Host can switch between "Active" and "Archived" tabs

### Components to build

- `QASubmitForm` — Text area, type toggle, submit button, char counter
- `QAEntry` — Entry card with body, score, type badge, vote arrows, timestamp
- `QAVoteArrows` — Upvote/downvote with highlight state
- `QAFeed` — Sorted list with load-more pagination
- `QAModerationActions` — Host-only action buttons per entry
- `QATabBar` — "Active" / "Archived" toggle for host

### Acceptance tests

- [ ] Audience types a question and submits → it appears in the feed
- [ ] Audience toggles to "Comment" and submits → comment appears (no vote arrows)
- [ ] Questions show upvote/downvote arrows, comments do not
- [ ] Clicking upvote → score increments by 1 visually
- [ ] Clicking upvote again → score decrements (toggle off)
- [ ] Clicking downvote after upvote → score changes by -2 (flip)
- [ ] Host clicks "Mark Answered" → green "Answered" badge appears
- [ ] Host clicks "Pin" → entry moves to the top
- [ ] Host clicks "Hide" → entry disappears from audience view, stays in host panel with "[Hidden]" label
- [ ] Host clicks "Unhide" → entry reappears for audience
- [ ] Feed loads 20 entries at a time with "Load more" button (cursor pagination)

### Files to create/modify

- `apps/web/components/qa/QASubmitForm.tsx`
- `apps/web/components/qa/QAEntry.tsx`
- `apps/web/components/qa/QAVoteArrows.tsx`
- `apps/web/components/qa/QAFeed.tsx`
- `apps/web/components/qa/QAModerationActions.tsx`
- `apps/web/app/session/[code]/page.tsx` — Add Q&A tab

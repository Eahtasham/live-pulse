## Phase 14 — Poll UI 🔵 FRONTEND

**Goal:** Host can create polls within a session, activate/close them. Audience can see active polls, cast votes, and view results as bar charts.

**Prerequisite:** Phase 13 + Phase 4 + Phase 5

### Host view — Poll management

**Create Poll form:**
- Question text input (max 500 chars)
- Dynamic option inputs: start with 2, "Add Option" button adds up to 6
- Each option: text input + delete button (disabled when ≤ 2 options remain)
- Answer mode toggle: "Single answer" (radio) / "Multi-select" (checkbox)
- Optional time limit input (seconds)
- Submit → calls `POST /v1/sessions/:code/polls`

**Poll list:**
- Shows all polls (including drafts) with status badges
- Each poll card has action buttons based on status:
  - `Draft`: "Activate" button + "Edit" / "Delete"
  - `Active`: "Close" button (with confirmation)
  - `Closed`: No actions — shows final results
- Active polls show a live vote count updating in real-time (Phase 16)

### Audience view — Voting

**Active poll display:**
- Question text prominently displayed
- Options rendered as buttons:
  - `single` mode: radio-button style (selecting one deselects others)
  - `multi` mode: checkbox style (multiple selections allowed)
- "Vote" submit button (disabled until at least one option selected)
- If time limit is set: countdown timer displayed

**Post-vote state:**
- "You voted" confirmation with selected option(s) highlighted
- Bar chart showing current vote distribution
- Vote button replaced with "Already voted" indicator

### Results bar chart (shared component)

- Horizontal bar chart: option label on left, bar proportional to vote count
- Each bar shows absolute count and percentage
- Highest-voted option highlighted
- Chart updates live when new votes arrive (Phase 16 will wire this to WebSocket)

### Components to build

- `CreatePollForm` — Question, options, answer mode, time limit
- `PollCard` — Poll with status badge, action buttons (host) or vote buttons (audience)
- `PollList` — Lists all polls, filtered by visibility (audience only sees active/closed)
- `VoteButtons` — Radio or checkbox options with submit
- `ResultsChart` — Horizontal bar chart with labels, counts, percentages
- `PollTimerBadge` — Countdown for time-limited polls

### Acceptance tests

- [ ] Host clicks "Create Poll" → form appears with question + option inputs
- [ ] Can add up to 6 options, cannot submit with fewer than 2
- [ ] After creating, poll appears in the poll list as `Draft`
- [ ] Host clicks "Activate" → poll status changes to `Active`
- [ ] Audience sees the active poll with votable options
- [ ] Single-answer poll: radio buttons — selecting one deselects others
- [ ] Multi-select poll: checkboxes — can select multiple
- [ ] After voting, audience sees "You voted" confirmation and a bar chart of results
- [ ] Attempting to vote again shows "Already voted" message
- [ ] Host clicks "Close" → poll status changes to `Closed`
- [ ] Closed poll shows final results (static chart) for both host and audience
- [ ] Draft polls are NOT visible to the audience

### Files to create/modify

- `apps/web/components/poll/CreatePollForm.tsx`
- `apps/web/components/poll/PollCard.tsx`
- `apps/web/components/poll/PollList.tsx`
- `apps/web/components/poll/VoteButtons.tsx`
- `apps/web/components/poll/ResultsChart.tsx`
- `apps/web/components/poll/PollTimerBadge.tsx`
- `apps/web/app/session/[code]/page.tsx` — Add polls tab

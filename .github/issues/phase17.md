## Phase 17 — Frontend Polish & UX 🔵 FRONTEND

**Goal:** Responsive design, smooth animations, loading states, error handling, and accessibility. Make the app production-ready for real users.

**Prerequisite:** Phase 16

### Mobile-first responsive layout

The primary audience (students answering polls) will be on phones. Design for 375px width first, then scale up.

**Key breakpoints:**
- `xs` (375px): Single-column layout, full-width cards, stacked controls
- `sm` (640px): Slightly wider cards, side-by-side vote buttons
- `md` (768px): Dashboard shows 2-column grid for session cards
- `lg` (1024px+): Host view shows polls + Q&A side-by-side

### Animations & transitions

- **Vote chart bars**: CSS transition on width (300ms ease-out) when counts update
- **New Q&A entries**: Slide in from top with opacity fade (200ms)
- **Vote button press**: Scale down to 95% on press, spring back (tactile feedback)
- **Status badge changes**: Background color transition (200ms)
- **Session code**: Subtle pulse animation when first displayed (draw attention)
- **Modal open/close**: Fade + slide up (200ms)

### Loading states

| Scenario | Loading indicator |
|----------|-------------------|
| Dashboard loading sessions | Skeleton cards (3 gray rectangles pulsing) |
| Session page loading polls/QA | Skeleton list items |
| Submitting a vote | Button shows spinner, disabled |
| Creating a session | Modal button shows spinner |
| Loading more Q&A entries | "Loading..." text at bottom |

### Error handling

| Scenario | User-facing message |
|----------|---------------------|
| Vote fails (network) | Toast: "Failed to submit vote. Try again." with retry button |
| Q&A submit fails | Toast: "Failed to send. Try again." — text preserved in input |
| Session not found | Full-page: "Session not found" with "Go Home" button |
| API unreachable | Banner: "Connection lost. Reconnecting..." |
| Expired session | "This session has ended" overlay |

### Touch targets & accessibility

- All interactive elements (buttons, links, inputs): minimum 44x44px touch target
- Focus ring visible on all interactive elements for keyboard navigation
- Tab order follows visual layout
- Semantic HTML: `<main>`, `<nav>`, `<section>`, `<h1>`–`<h3>` hierarchy
- ARIA labels on icon-only buttons (e.g., copy, upvote, downvote)
- Color contrast: WCAG AA (4.5:1 text, 3:1 large text/UI components)
- Screen reader: vote buttons announce "Upvote question" / "Downvote question"

### Branding

- LivePulse logo in header (text logo is fine for MVP)
- Consistent color scheme using Tailwind CSS variables
- Session code always displayed in mono-spaced font, large size, with copy button

### Acceptance tests

- [ ] App is usable on 375px-wide screen (iPhone SE) — no horizontal scrolling
- [ ] Vote buttons are at least 44x44px (touch target guideline)
- [ ] After voting, selected option shows a visual "selected" state within 200ms
- [ ] Chart bars animate smoothly when counts change (CSS transitions)
- [ ] Loading states (spinners or skeletons) are shown while waiting for API responses
- [ ] Failed API calls show a user-friendly error message (not raw error text)
- [ ] "Join Session" input accepts codes in any case (`a1b2c3` → `A1B2C3`)
- [ ] Session code is displayed in a large, copyable, mono-spaced format
- [ ] HTML is semantically correct — proper headings, labels, ARIA attributes
- [ ] Color contrast ratios meet WCAG AA (4.5:1 for text)
- [ ] Tab navigation works through all interactive elements (keyboard accessible)

### Files to create/modify

- All existing components in `apps/web/components/` — add responsive styles, animations, loading states
- `apps/web/components/ui/Toast.tsx` — Error/success toast notifications
- `apps/web/components/ui/Skeleton.tsx` — Loading skeleton component
- `apps/web/components/ui/Spinner.tsx` — Button loading spinner
- `apps/web/app/globals.css` — Animation keyframes, custom properties
- `apps/web/app/not-found.tsx` — Custom 404 page

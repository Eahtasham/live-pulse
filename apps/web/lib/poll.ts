// ── Types ────────────────────────────────────────────────────────────

export interface PollOption {
  id: string
  label: string
  position: number
  vote_count: number
}

export interface Poll {
  id: string
  session_id: string
  question: string
  answer_mode: "single" | "multi"
  status: "draft" | "active" | "closed"
  time_limit_sec: number | null
  options: PollOption[]
  created_at: string
  updated_at: string
}

export interface VoteResponse {
  id: string
  poll_id: string
  option_id: string
  audience_uid: string
}

// ── Types ────────────────────────────────────────────────────────────

export interface QAEntry {
  id: string
  session_id: string
  author_uid: string
  entry_type: "question" | "comment"
  body: string
  score: number
  status: "visible" | "answered" | "pinned" | "archived"
  is_hidden: boolean
  created_at: string
  updated_at: string
  user_vote?: 1 | -1 | null // null = not voted, 1 = upvoted, -1 = downvoted
}

export interface QAListResponse {
  entries: QAEntry[]
  next_cursor: string
}

export interface CreateQARequest {
  entry_type: "question" | "comment"
  body: string
}

export interface ModerateQARequest {
  status?: "visible" | "answered" | "pinned" | "archived"
  is_hidden?: boolean
}

export interface QAVoteRequest {
  audience_uid: string
  value: 1 | -1
}

export interface QAVoteResponse {
  id?: string
  qa_entry_id?: string
  voter_uid?: string
  vote_value?: number
  action: "voted" | "removed"
}

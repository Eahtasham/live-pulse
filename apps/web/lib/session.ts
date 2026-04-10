import { apiFetch, API_URL } from "./api"

// ── Types ────────────────────────────────────────────────────────────

export interface Session {
  id: string
  code: string
  title: string
  status: string
  created_at: string
  host_id?: string
}

export interface JoinResponse {
  audience_uid: string
  session_title: string
}

// ── Server-side API (uses auth via apiFetch) ─────────────────────────

export async function createSession(title: string): Promise<Session> {
  const res = await apiFetch("/v1/sessions", {
    method: "POST",
    body: JSON.stringify({ title }),
  })
  if (!res.ok) throw new Error("Failed to create session")
  return res.json()
}

export async function listSessions(): Promise<Session[]> {
  const res = await apiFetch("/v1/sessions")
  if (!res.ok) throw new Error("Failed to list sessions")
  return res.json()
}

// ── Public API (no auth needed) ──────────────────────────────────────

export async function getSessionByCode(code: string): Promise<Session> {
  const res = await fetch(
    `${API_URL}/v1/sessions/${encodeURIComponent(code)}`,
    {
      headers: { "Content-Type": "application/json" },
      cache: "no-store",
    }
  )
  if (!res.ok) throw new Error("Session not found")
  return res.json()
}

export async function joinSession(
  code: string,
  clientId: string
): Promise<JoinResponse> {
  const res = await fetch(
    `${API_URL}/v1/sessions/${encodeURIComponent(code)}/join`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Client-ID": clientId,
      },
    }
  )
  if (!res.ok) throw new Error("Failed to join session")
  return res.json()
}

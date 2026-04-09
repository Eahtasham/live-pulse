import { auth } from "@/auth"

const API_URL = process.env.API_URL || "http://localhost:8080"

/**
 * Fetch wrapper that attaches the Go API JWT from the current session.
 */
export async function apiFetch(
  path: string,
  init?: RequestInit
): Promise<Response> {
  const session = await auth()
  const headers = new Headers(init?.headers)
  headers.set("Content-Type", "application/json")

  if (session?.apiToken) {
    headers.set("Authorization", `Bearer ${session.apiToken}`)
  }

  return fetch(`${API_URL}${path}`, {
    ...init,
    headers,
  })
}

// ── Session types ────────────────────────────────────────────────────

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

// ── Session API (server-side, uses auth) ─────────────────────────────

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

export async function getSessionByCode(code: string): Promise<Session> {
  const res = await fetch(`${API_URL}/v1/sessions/${encodeURIComponent(code)}`, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  })
  if (!res.ok) throw new Error("Session not found")
  return res.json()
}

export async function joinSession(
  code: string,
  clientId: string
): Promise<JoinResponse> {
  const res = await fetch(`${API_URL}/v1/sessions/${encodeURIComponent(code)}/join`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Client-ID": clientId,
    },
  })
  if (!res.ok) throw new Error("Failed to join session")
  return res.json()
}

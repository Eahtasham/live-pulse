import { auth } from "@/auth"

export const API_URL = process.env.API_URL || "http://localhost:8080"

/**
 * Fetch wrapper that attaches the Go API JWT from the current session.
 * For use in server components / server actions only.
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


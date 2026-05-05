"use client";

import { useEffect, useCallback, useRef } from "react";
import type { useWebSocket, WSMessage } from "@/hooks/use-websocket";
import type { PollOption } from "@/lib/poll";

interface VoteUpdatePayload {
  pollId: string;
  options: { id: string; label: string; vote_count: number }[];
}

interface PollVotesCallbacks {
  onVoteUpdate: (pollId: string, options: PollOption[]) => void;
}

export function usePollVotes(
  ws: ReturnType<typeof useWebSocket>,
  sessionCode: string,
  token: string | undefined,
  callbacks: PollVotesCallbacks
) {
  const cbRef = useRef(callbacks);

  useEffect(() => {
    cbRef.current = callbacks;
  }, [callbacks]);

  // Forward each vote_update directly to PollList's buffer (no batching here;
  // PollList owns the rAF render loop that coalesces at 60fps).
  useEffect(() => {
    return ws.subscribe((msg: WSMessage) => {
      if (msg.type === "vote_update") {
        const payload = msg.payload as VoteUpdatePayload;
        const options: PollOption[] = payload.options.map((o, i) => ({
          id: o.id,
          label: o.label,
          position: i,
          vote_count: o.vote_count,
        }));
        cbRef.current.onVoteUpdate(payload.pollId, options);
      }
    });
  }, [ws]);

  // On reconnect, refetch all polls
  const refetch = useCallback(async () => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    try {
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (token) headers.Authorization = `Bearer ${token}`;
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/polls?include_votes=true`,
        { headers }
      );
      if (res.ok) {
        const polls = await res.json();
        if (Array.isArray(polls)) {
          for (const poll of polls) {
            cbRef.current.onVoteUpdate(poll.id, poll.options);
          }
        }
      }
    } catch {
      // silently fail — will retry on next reconnect
    }
  }, [sessionCode, token]);

  useEffect(() => {
    return ws.onReconnect(refetch);
  }, [ws, refetch]);
}

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
  cbRef.current = callbacks;

  // Buffer: keep only the latest options per pollId, flush every 500ms
  const bufferRef = useRef<Map<string, PollOption[]>>(new Map());
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  function scheduleFlush() {
    if (timerRef.current !== null) return; // already scheduled
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      const pending = bufferRef.current;
      if (pending.size === 0) return;
      const entries = Array.from(pending.entries());
      pending.clear();
      for (const [pollId, options] of entries) {
        cbRef.current.onVoteUpdate(pollId, options);
      }
    }, 500);
  }

  // Handle WS messages — buffer updates, flush every 500ms
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
        bufferRef.current.set(payload.pollId, options);
        scheduleFlush();
      }
    });
  }, [ws]);

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current);
    };
  }, []);

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

"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CreatePollForm } from "@/components/poll/create-poll-form";
import { PollCard } from "@/components/poll/poll-card";
import type { Poll, PollOption } from "@/lib/poll";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  sessionCode: string;
  isHost: boolean;
  token?: string;
  audienceUid: string;
  onRegisterUpdater?: (updater: (pollId: string, options: PollOption[]) => void) => void;
  onAnyVote?: () => void;
}

export function PollList({ sessionCode, isHost, token, audienceUid, onRegisterUpdater, onAnyVote }: Props) {
  const [polls, setPolls] = useState<Poll[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);
  // Map of pollId → option IDs the current audience member voted for
  const [myVotes, setMyVotes] = useState<Record<string, string[]>>({});

  // --- rAF-based render loop: buffer WS updates, flush at most 60fps ---
  const pendingRef = useRef<Map<string, PollOption[]>>(new Map());

  useEffect(() => {
    let raf: number;
    const tick = () => {
      if (pendingRef.current.size > 0) {
        const batch = new Map(pendingRef.current);
        pendingRef.current.clear();
        setPolls((prev) => {
          let next = prev;
          for (const [pollId, options] of batch) {
            const idx = next.findIndex((p) => p.id === pollId);
            if (idx === -1) continue;
            if (next === prev) next = [...prev]; // copy-on-write
            next[idx] = {
              ...next[idx],
              options: next[idx].options.map((existing) => {
                const u = options.find((o) => o.id === existing.id);
                return u ? { ...existing, vote_count: u.vote_count } : existing;
              }),
            };
          }
          return next;
        });
      }
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, []);

  const fetchPolls = useCallback(async () => {
    try {
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (token) {
        headers.Authorization = `Bearer ${token}`;
      }

      // TODO: Remove ?include_votes=true when real-time WebSocket vote updates are enabled
      // Currently needed to refresh vote counts after voting (REST polling fallback)
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/polls?include_votes=true`,
        { headers }
      );

      if (res.ok) {
        const data = await res.json();
        setPolls(data ?? []);
      }
    } finally {
      setLoading(false);
    }
  }, [sessionCode, token]);

  useEffect(() => {
    fetchPolls();
  }, [fetchPolls]);

  // Fetch which polls this audience member has already voted on (source of truth)
  useEffect(() => {
    if (isHost || !audienceUid) return;
    async function fetchMyVotes() {
      try {
        const res = await fetch(
          `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/polls/votes?audience_uid=${encodeURIComponent(audienceUid)}`,
          { headers: { "Content-Type": "application/json" } }
        );
        if (res.ok) {
          const data: { poll_id: string; option_ids: string[] }[] = await res.json();
          const map: Record<string, string[]> = {};
          for (const entry of data ?? []) {
            map[entry.poll_id] = entry.option_ids;
          }
          setMyVotes(map);
        }
      } catch {
        // silent — will fall back to no pre-filled state
      }
    }
    fetchMyVotes();
  }, [sessionCode, audienceUid, isHost]);

  // Register updater: writes to pendingRef (no setState), rAF loop flushes
  const bufferUpdate = useCallback(
    (pollId: string, options: PollOption[]) => {
      pendingRef.current.set(pollId, options);
    },
    []
  );

  useEffect(() => {
    onRegisterUpdater?.(bufferUpdate);
  }, [onRegisterUpdater, bufferUpdate]);

  function handlePollCreated() {
    setShowCreateForm(false);
    fetchPolls();
  }

  function handleVoted(pollId: string, optionIds: string[]) {
    setMyVotes((prev) => ({ ...prev, [pollId]: optionIds }));
    onAnyVote?.();
  }

  if (loading) {
    return (
      <p className="text-sm text-muted-foreground">Loading polls...</p>
    );
  }

  return (
    <div className="space-y-4">
      {/* Host: Create poll button + form */}
      {isHost && (
        <>
          {showCreateForm ? (
            <Card>
              <CardHeader>
                <CardTitle>Create Poll</CardTitle>
              </CardHeader>
              <CardContent>
                <CreatePollForm
                  sessionCode={sessionCode}
                  token={token}
                  onCreated={handlePollCreated}
                  onCancel={() => setShowCreateForm(false)}
                />
              </CardContent>
            </Card>
          ) : (
            <Button onClick={() => setShowCreateForm(true)}>
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M5 12h14" />
                <path d="M12 5v14" />
              </svg>
              Create Poll
            </Button>
          )}
        </>
      )}

      {/* Polls list */}
      {polls.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <p className="text-sm text-muted-foreground">
              {isHost
                ? "No polls yet. Create one to get started!"
                : "No active polls at the moment."}
            </p>
          </CardContent>
        </Card>
      ) : (
        polls.map((poll) => (
          <PollCard
            key={poll.id}
            poll={poll}
            isHost={isHost}
            sessionCode={sessionCode}
            token={token}
            audienceUid={audienceUid}
            initialVotedOptionIds={myVotes[poll.id]}
            onVoted={(optionIds) => handleVoted(poll.id, optionIds)}
            onStatusChanged={fetchPolls}
          />
        ))
      )}
    </div>
  );
}

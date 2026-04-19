"use client";

import { useEffect, useState, useCallback } from "react";
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
}

export function PollList({ sessionCode, isHost, token, audienceUid, onRegisterUpdater }: Props) {
  const [polls, setPolls] = useState<Poll[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);

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

  // Register the live vote updater with the parent
  const updatePollOptions = useCallback(
    (pollId: string, options: PollOption[]) => {
      setPolls((prev) =>
        prev.map((p) =>
          p.id === pollId
            ? {
                ...p,
                options: p.options.map((existing) => {
                  const updated = options.find((o) => o.id === existing.id);
                  return updated
                    ? { ...existing, vote_count: updated.vote_count }
                    : existing;
                }),
              }
            : p
        )
      );
    },
    []
  );

  useEffect(() => {
    onRegisterUpdater?.(updatePollOptions);
  }, [onRegisterUpdater, updatePollOptions]);

  function handlePollCreated() {
    setShowCreateForm(false);
    fetchPolls();
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
            onStatusChanged={fetchPolls}
          />
        ))
      )}
    </div>
  );
}

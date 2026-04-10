"use client";

import { useEffect, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CreatePollForm } from "@/components/poll/create-poll-form";
import { PollCard } from "@/components/poll/poll-card";
import type { Poll } from "@/lib/poll";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  sessionCode: string;
  isHost: boolean;
  token?: string;
  audienceUid: string;
}

export function PollList({ sessionCode, isHost, token, audienceUid }: Props) {
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

      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/polls`,
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

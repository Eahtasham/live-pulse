"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { VoteButtons } from "@/components/poll/vote-buttons";
import { ResultsChart } from "@/components/poll/results-chart";
import { PollTimerBadge } from "@/components/poll/poll-timer-badge";
import type { Poll } from "@/lib/poll";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  poll: Poll;
  isHost: boolean;
  sessionCode: string;
  token?: string;
  audienceUid: string;
  initialVotedOptionIds?: string[];
  onVoted?: (optionIds: string[]) => void;
  onStatusChanged: () => void;
}

const statusColors: Record<string, "default" | "secondary" | "outline"> = {
  draft: "outline",
  active: "default",
  closed: "secondary",
};

export const PollCard = React.memo(function PollCard({
  poll,
  isHost,
  sessionCode,
  token,
  audienceUid,
  initialVotedOptionIds,
  onVoted,
  onStatusChanged,
}: Props) {
  const [loading, setLoading] = useState(false);
  const [hasVoted, setHasVoted] = useState(
    () => !!(initialVotedOptionIds && initialVotedOptionIds.length > 0)
  );
  const [votedOptionIds, setVotedOptionIds] = useState<string[]>(
    () => initialVotedOptionIds ?? []
  );
  const [confirmClose, setConfirmClose] = useState(false);

  // Sync from backend-fetched vote state (arrives after initial render)
  useEffect(() => {
    if (initialVotedOptionIds && initialVotedOptionIds.length > 0) {
      setHasVoted(true);
      setVotedOptionIds(initialVotedOptionIds);
    }
  }, [initialVotedOptionIds]);

  async function updateStatus(newStatus: string) {
    setLoading(true);
    try {
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/polls/${poll.id}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({ status: newStatus }),
        }
      );

      if (res.ok) {
        onStatusChanged();
      }
    } finally {
      setLoading(false);
      setConfirmClose(false);
    }
  }

  function handleVoted(selectedIds: string[]) {
    setHasVoted(true);
    setVotedOptionIds(selectedIds);
    onVoted?.(selectedIds);
    onStatusChanged(); // refresh to get updated vote counts
  }

  const showResults =
    poll.status === "closed" || (poll.status === "active" && hasVoted) || isHost;

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-2">
        <div className="space-y-1 min-w-0">
          <CardTitle className="text-base leading-snug">
            {poll.question}
          </CardTitle>
          <div className="flex items-center gap-2">
            <Badge variant={statusColors[poll.status] ?? "outline"}>
              {poll.status}
            </Badge>
            <span className="text-xs text-muted-foreground">
              {poll.answer_mode === "single"
                ? "Single answer"
                : "Multi-select"}
            </span>
            {poll.time_limit_sec && poll.status === "active" && (
              <PollTimerBadge
                timeLimitSec={poll.time_limit_sec}
                isActive={poll.status === "active"}
              />
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Host controls */}
        {isHost && (
          <div className="flex gap-2">
            {poll.status === "draft" && (
              <Button
                size="sm"
                onClick={() => updateStatus("active")}
                disabled={loading}
              >
                {loading ? "Activating..." : "Activate"}
              </Button>
            )}
            {poll.status === "active" && !confirmClose && (
              <Button
                size="sm"
                variant="secondary"
                onClick={() => setConfirmClose(true)}
                disabled={loading}
              >
                Close Poll
              </Button>
            )}
            {poll.status === "active" && confirmClose && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground">
                  Close this poll?
                </span>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => updateStatus("closed")}
                  disabled={loading}
                >
                  {loading ? "Closing..." : "Confirm"}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setConfirmClose(false)}
                >
                  Cancel
                </Button>
              </div>
            )}
          </div>
        )}

        {/* Audience voting (active poll, not yet voted) */}
        {!isHost && poll.status === "active" && !hasVoted && (
          <VoteButtons
            pollId={poll.id}
            sessionCode={sessionCode}
            options={poll.options}
            answerMode={poll.answer_mode}
            audienceUid={audienceUid}
            onVoted={handleVoted}
          />
        )}

        {/* Post-vote confirmation */}
        {!isHost && hasVoted && poll.status === "active" && (
          <div className="rounded-lg bg-primary/10 px-4 py-3">
            <p className="text-sm font-medium text-primary">
              You&apos;ve voted!
            </p>
          </div>
        )}

        {/* Results chart */}
        {showResults && (
          <ResultsChart
            options={poll.options}
            highlightedIds={votedOptionIds}
          />
        )}

        {/* Already voted indicator for audience on closed polls */}
        {!isHost && poll.status === "closed" && !hasVoted && (
          <ResultsChart options={poll.options} />
        )}
      </CardContent>
    </Card>
  );
});

PollCard.displayName = "PollCard";

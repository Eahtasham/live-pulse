"use client";

import { useState } from "react";
import { ChevronUp, ChevronDown } from "lucide-react";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  entryId: string;
  sessionCode: string;
  audienceUid: string;
  score: number;
  userVote: 1 | -1 | null;
  onVoted: (newScore: number, newVote: 1 | -1 | null) => void;
}

export function QAVoteArrows({
  entryId,
  sessionCode,
  audienceUid,
  score,
  userVote,
  onVoted,
}: Props) {
  const [loading, setLoading] = useState(false);

  async function handleVote(value: 1 | -1) {
    if (loading || !audienceUid) return;

    // Optimistic calculation
    let newVote: 1 | -1 | null;
    let newScore: number;

    if (userVote === value) {
      // Toggle off
      newVote = null;
      newScore = score - value;
    } else if (userVote === null) {
      // Fresh vote
      newVote = value;
      newScore = score + value;
    } else {
      // Flip
      newVote = value;
      newScore = score + value - userVote;
    }

    onVoted(newScore, newVote);
    setLoading(true);

    try {
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/qa/${entryId}/vote`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ audience_uid: audienceUid, value }),
        }
      );
      if (!res.ok) {
        // Revert on error
        onVoted(score, userVote);
      }
    } catch {
      onVoted(score, userVote);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col items-center gap-0.5 select-none">
      <button
        type="button"
        onClick={() => handleVote(1)}
        disabled={loading}
        aria-label="Upvote"
        className={`group rounded-md p-1 transition-all ${
          userVote === 1
            ? "text-primary bg-primary/10"
            : "text-muted-foreground hover:text-primary hover:bg-primary/5"
        }`}
      >
        <ChevronUp
          className={`h-5 w-5 transition-transform group-hover:scale-110 ${
            userVote === 1 ? "stroke-[3px]" : ""
          }`}
        />
      </button>

      <span
        className={`text-sm font-bold tabular-nums min-w-6 text-center leading-none ${
          userVote === 1
            ? "text-primary"
            : userVote === -1
              ? "text-destructive"
              : "text-muted-foreground"
        }`}
      >
        {score}
      </span>

      <button
        type="button"
        onClick={() => handleVote(-1)}
        disabled={loading}
        aria-label="Downvote"
        className={`group rounded-md p-1 transition-all ${
          userVote === -1
            ? "text-destructive bg-destructive/10"
            : "text-muted-foreground hover:text-destructive hover:bg-destructive/5"
        }`}
      >
        <ChevronDown
          className={`h-5 w-5 transition-transform group-hover:scale-110 ${
            userVote === -1 ? "stroke-[3px]" : ""
          }`}
        />
      </button>
    </div>
  );
}

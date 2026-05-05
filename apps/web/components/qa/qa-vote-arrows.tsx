"use client";

import { useRef, useState } from "react";
import { ChevronUp, ChevronDown } from "lucide-react";

import { Toast } from "@/components/ui/Toast";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  entryId: string;
  sessionCode: string;
  audienceUid: string;
  score: number;
  userVote: 1 | -1 | null;
  onVoted: (newScore: number, newVote: 1 | -1 | null, action?: string) => void;
  disabled?: boolean;
}

export function QAVoteArrows({
  entryId,
  sessionCode,
  audienceUid,
  score,
  userVote,
  onVoted,
  disabled = false,
}: Props) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const inFlightRef = useRef(false);

  async function handleVote(value: 1 | -1) {
    if (loading || disabled || !audienceUid || inFlightRef.current) return;
    setError("");
    inFlightRef.current = true;

    // Optimistic update
    let optimisticVote: 1 | -1 | null;
    let optimisticScore: number;

    if (userVote === value) {
      optimisticVote = null;
      optimisticScore = score - value;
    } else if (userVote === null) {
      optimisticVote = value;
      optimisticScore = score + value;
    } else {
      optimisticVote = value;
      optimisticScore = score + value - userVote;
    }

    onVoted(optimisticScore, optimisticVote);
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
        onVoted(score, userVote);
        const data = await res.json().catch(() => ({}));
        setError(data.message || "Failed to submit vote");
      } else {
        // Use actual score from server response
        const data = await res.json();
        if (data.score !== undefined) {
          onVoted(data.score, optimisticVote, data.action);
        }
      }
    } catch {
      onVoted(score, userVote);
      setError("Something went wrong. Please try again.");
    } finally {
      setLoading(false);
      inFlightRef.current = false;
    }
  }

  return (
    <div className="flex select-none flex-col items-center gap-0.5">
      <button
        type="button"
        onClick={() => handleVote(1)}
        disabled={loading || disabled}
        aria-label="Upvote"
        className={`group rounded-md p-1 transition-all ${
          disabled
            ? "cursor-not-allowed text-muted-foreground/40"
            : userVote === 1
              ? "bg-primary/10 text-primary"
              : "text-muted-foreground hover:bg-primary/5 hover:text-primary"
        }`}
      >
        <ChevronUp
          className={`h-5 w-5 transition-transform group-hover:scale-110 ${
            userVote === 1 ? "stroke-[3px]" : ""
          }`}
        />
      </button>

      <span
        className={`min-w-6 text-center text-sm font-bold tabular-nums leading-none ${
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
        disabled={loading || disabled}
        aria-label="Downvote"
        className={`group rounded-md p-1 transition-all ${
          disabled
            ? "cursor-not-allowed text-muted-foreground/40"
            : userVote === -1
              ? "bg-destructive/10 text-destructive"
              : "text-muted-foreground hover:bg-destructive/5 hover:text-destructive"
        }`}
      >
        <ChevronDown
          className={`h-5 w-5 transition-transform group-hover:scale-110 ${
            userVote === -1 ? "stroke-[3px]" : ""
          }`}
        />
      </button>

      {error ? <Toast variant="error" description={error} className="mt-2 w-64" /> : null}
    </div>
  );
}

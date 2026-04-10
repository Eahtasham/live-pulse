"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { PollOption } from "@/lib/poll";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  pollId: string;
  sessionCode: string;
  options: PollOption[];
  answerMode: "single" | "multi";
  audienceUid: string;
  onVoted: (selectedIds: string[]) => void;
}

export function VoteButtons({
  pollId,
  sessionCode,
  options,
  answerMode,
  audienceUid,
  onVoted,
}: Props) {
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  function toggleOption(id: string) {
    const next = new Set(selected);
    if (answerMode === "single") {
      next.clear();
      next.add(id);
    } else {
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
    }
    setSelected(next);
  }

  async function handleVote() {
    if (selected.size === 0) return;
    setLoading(true);
    setError("");

    try {
      // Submit votes for each selected option
      for (const optionId of selected) {
        const res = await fetch(
          `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/polls/${pollId}/vote`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              option_id: optionId,
              audience_uid: audienceUid,
            }),
          }
        );

        if (!res.ok) {
          const data = await res.json().catch(() => ({}));
          if (data.message?.includes("already voted")) {
            setError("You have already voted on this poll");
            return;
          }
          setError(data.message || "Failed to vote");
          return;
        }
      }

      onVoted(Array.from(selected));
    } catch {
      setError("Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-3">
      {error && (
        <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-950 dark:text-red-400">
          {error}
        </p>
      )}
      <div className="space-y-2">
        {options.map((option) => {
          const isSelected = selected.has(option.id);
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => toggleOption(option.id)}
              className={`flex w-full items-center gap-3 rounded-lg border px-4 py-3 text-left text-sm font-medium transition-colors ${
                isSelected
                  ? "border-primary bg-primary/10 text-primary"
                  : "border-border hover:bg-muted"
              }`}
            >
              {/* Radio or checkbox indicator */}
              <span
                className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-${
                  answerMode === "single" ? "full" : "md"
                } border-2 ${
                  isSelected
                    ? "border-primary bg-primary"
                    : "border-muted-foreground"
                }`}
              >
                {isSelected && (
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="12"
                    height="12"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="white"
                    strokeWidth="3"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M20 6 9 17l-5-5" />
                  </svg>
                )}
              </span>
              {option.label}
            </button>
          );
        })}
      </div>
      <Button
        onClick={handleVote}
        disabled={selected.size === 0 || loading}
        className="w-full"
      >
        {loading ? "Submitting..." : "Vote"}
      </Button>
    </div>
  );
}

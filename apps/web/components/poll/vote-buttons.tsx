"use client";

import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/Spinner";
import { Toast } from "@/components/ui/Toast";
import type { PollOption } from "@/lib/poll";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  pollId: string;
  sessionCode: string;
  options: PollOption[];
  answerMode: "single" | "multi";
  audienceUid: string;
  onVoted: (selectedIds: string[]) => void;
  disabled?: boolean;
}

export function VoteButtons({
  pollId,
  sessionCode,
  options,
  answerMode,
  audienceUid,
  onVoted,
  disabled = false,
}: Props) {
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const inFlightRef = useRef(false);

  function toggleOption(id: string) {
    if (disabled || loading) return;
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
    if (selected.size === 0 || disabled || loading || inFlightRef.current) return;
    inFlightRef.current = true;
    setLoading(true);
    setError("");

    try {
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/polls/${pollId}/vote`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            option_ids: Array.from(selected),
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

      onVoted(Array.from(selected));
    } catch {
      setError("Something went wrong");
    } finally {
      setLoading(false);
      inFlightRef.current = false;
    }
  }

  return (
    <div className="space-y-3">
      {disabled ? (
        <Toast
          variant="warning"
          description="Voting is closed because this session has ended."
        />
      ) : null}
      {error ? <Toast variant="error" description={error} /> : null}
      <div className="space-y-2">
        {options.map((option) => {
          const isSelected = selected.has(option.id);
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => toggleOption(option.id)}
              disabled={disabled || loading}
              className={`flex w-full items-center gap-3 rounded-lg border px-4 py-3 text-left text-sm font-medium transition-colors ${
                disabled
                  ? "cursor-not-allowed border-border/60 bg-muted/40 text-muted-foreground"
                  : isSelected
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
        disabled={selected.size === 0 || loading || disabled}
        className="w-full"
      >
        {loading ? <Spinner label="Submitting..." /> : disabled ? "Voting closed" : "Vote"}
      </Button>
    </div>
  );
}

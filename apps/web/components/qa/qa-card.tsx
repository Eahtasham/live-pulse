"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ArrowBigUp, ArrowBigDown } from "lucide-react";
import type { QAEntry } from "@/lib/qa";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  entry: QAEntry;
  sessionCode: string;
  audienceUid: string;
  isHost: boolean;
  token?: string;
  onUpdated: () => void;
}

const statusColors: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  visible: "outline",
  answered: "default",
  pinned: "secondary",
  archived: "destructive",
};

export function QACard({ entry, sessionCode, audienceUid, isHost, token, onUpdated }: Props) {
  const [loading, setLoading] = useState(false);
  const [voteLoading, setVoteLoading] = useState(false);
  const [confirmHide, setConfirmHide] = useState(false);
  
  // ALWAYS use server state as source of truth
  // Local state is only for optimistic updates during the API call
  const [optimisticVote, setOptimisticVote] = useState<1 | -1 | null>(null);
  const [optimisticScore, setOptimisticScore] = useState<number | null>(null);
  
  // Reset optimistic state when entry updates from server
  useEffect(() => {
    setOptimisticVote(null);
    setOptimisticScore(null);
  }, [entry.id, entry.user_vote, entry.score]);
  
  // Compute current state: optimistic takes precedence during API call, else server state
  const currentVote = optimisticVote !== null ? optimisticVote : (entry.user_vote ?? null);
  const currentScore = optimisticScore !== null ? optimisticScore : entry.score;

  async function moderate(action: string, isHidden?: boolean) {
    if (!token) return;
    setLoading(true);

    try {
      const body: { status?: string; is_hidden?: boolean } = {};
      if (action) body.status = action;
      if (isHidden !== undefined) body.is_hidden = isHidden;

      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/qa/${entry.id}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify(body),
        }
      );

      if (res.ok) {
        onUpdated();
      }
    } finally {
      setLoading(false);
      setConfirmHide(false);
    }
  }

  async function castVote(value: 1 | -1) {
    if (entry.entry_type !== "question" || voteLoading) return;
    
    // Must have an audience UID to vote (host or audience)
    if (!audienceUid) {
      console.error("Cannot vote: no audience UID provided");
      return;
    }
    
    setVoteLoading(true);
    
    let newVote: 1 | -1 | null;
    let newScore: number;
    
    if (currentVote === value) {
      // Toggle off - removing vote
      newVote = null;
      newScore = currentScore - value;
    } else if (currentVote === null) {
      // New vote
      newVote = value;
      newScore = currentScore + value;
    } else {
      // Changing vote (e.g., upvote -> downvote)
      newVote = value;
      newScore = currentScore + value - currentVote;
    }
    
    // Apply optimistic update
    setOptimisticVote(newVote);
    setOptimisticScore(newScore);

    try {
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/qa/${entry.id}/vote`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            audience_uid: audienceUid,
            value,
          }),
        }
      );

      if (res.ok) {
        // Refresh to get actual server state
        // This will trigger useEffect to reset optimistic state
        onUpdated();
      } else {
        // Revert on error - reset optimistic state
        setOptimisticVote(null);
        setOptimisticScore(null);
      }
    } catch {
      // Revert on network error
      setOptimisticVote(null);
      setOptimisticScore(null);
    } finally {
      setVoteLoading(false);
    }
  }

  // Hosts and audience can both vote - they just need an audienceUid
  const canVote = entry.entry_type === "question" && 
                  entry.status !== "archived" && 
                  !entry.is_hidden &&
                  !!audienceUid;

  return (
    <Card className={entry.is_hidden ? "opacity-60" : ""}>
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-2 flex-wrap">
            <Badge variant={entry.entry_type === "question" ? "default" : "outline"}>
              {entry.entry_type}
            </Badge>
            <Badge variant={statusColors[entry.status] ?? "outline"}>{entry.status}</Badge>
            {entry.is_hidden && <Badge variant="destructive">hidden</Badge>}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm">{entry.body}</p>

        {/* Reddit-style voting (only for questions) */}
        {canVote && (
          <div className="flex items-center gap-1 select-none">
            <button
              onClick={() => castVote(1)}
              disabled={voteLoading}
              className={`p-1 rounded hover:bg-muted transition-colors ${
                currentVote === 1 
                  ? "text-orange-500 hover:text-orange-600" 
                  : "text-muted-foreground hover:text-foreground"
              }`}
              aria-label="Upvote"
            >
              <ArrowBigUp 
                className={`w-6 h-6 ${currentVote === 1 ? "fill-current" : ""}`} 
              />
            </button>
            
            <span className={`font-semibold text-sm min-w-[1.5rem] text-center ${
              currentVote === 1 
                ? "text-orange-500" 
                : currentVote === -1 
                  ? "text-indigo-500" 
                  : "text-muted-foreground"
            }`}>
              {currentScore}
            </span>
            
            <button
              onClick={() => castVote(-1)}
              disabled={voteLoading}
              className={`p-1 rounded hover:bg-muted transition-colors ${
                currentVote === -1 
                  ? "text-indigo-500 hover:text-indigo-600" 
                  : "text-muted-foreground hover:text-foreground"
              }`}
              aria-label="Downvote"
            >
              <ArrowBigDown 
                className={`w-6 h-6 ${currentVote === -1 ? "fill-current" : ""}`} 
              />
            </button>
          </div>
        )}

        {/* Host moderation */}
        {isHost && (
          <div className="flex flex-wrap gap-2 pt-2 border-t">
            {entry.status !== "answered" && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => moderate("answered")}
                disabled={loading}
              >
                Mark Answered
              </Button>
            )}
            {entry.status !== "pinned" && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => moderate("pinned")}
                disabled={loading}
              >
                Pin
              </Button>
            )}
            {entry.status !== "visible" && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => moderate("visible")}
                disabled={loading}
              >
                Unpin
              </Button>
            )}
            {!confirmHide ? (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setConfirmHide(true)}
                disabled={loading}
              >
                {entry.is_hidden ? "Unhide" : "Hide"}
              </Button>
            ) : (
              <>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => moderate("", !entry.is_hidden)}
                  disabled={loading}
                >
                  Confirm
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setConfirmHide(false)}
                >
                  Cancel
                </Button>
              </>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

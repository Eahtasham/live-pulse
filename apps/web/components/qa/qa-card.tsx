"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
    if (entry.entry_type !== "question") return;
    setVoteLoading(true);

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
        onUpdated();
      }
    } finally {
      setVoteLoading(false);
    }
  }

  const canVote = entry.entry_type === "question" && entry.status !== "archived" && !entry.is_hidden;

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
          <span className="text-xs text-muted-foreground">
            Score: {entry.score}
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm">{entry.body}</p>

        {/* Voting buttons (only for questions) */}
        {canVote && (
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => castVote(1)}
              disabled={voteLoading}
            >
              ▲ Upvote
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => castVote(-1)}
              disabled={voteLoading}
            >
              ▼ Downvote
            </Button>
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

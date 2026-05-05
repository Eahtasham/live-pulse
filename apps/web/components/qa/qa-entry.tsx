"use client";

import { useState, useEffect } from "react";
import { Badge } from "@/components/ui/badge";
import { QAVoteArrows } from "@/components/qa/qa-vote-arrows";
import { QAModerationActions } from "@/components/qa/qa-moderation-actions";
import {
  MessageCircleQuestion,
  MessageSquareText,
  Pin,
  CheckCircle2,
  EyeOff,
  Clock,
} from "lucide-react";
import type { QAEntry as QAEntryType } from "@/lib/qa";

interface Props {
  entry: QAEntryType;
  sessionCode: string;
  audienceUid: string;
  isHost: boolean;
  token?: string;
  sessionEnded?: boolean;
  onUpdated: () => void;
}

function relativeTime(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function QAEntryCard({
  entry,
  sessionCode,
  audienceUid,
  isHost,
  token,
  sessionEnded = false,
  onUpdated,
}: Props) {
  const [currentScore, setCurrentScore] = useState(entry.score);
  const [currentVote, setCurrentVote] = useState<1 | -1 | null>(
    entry.user_vote ?? null
  );

  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      setCurrentScore(entry.score);
      setCurrentVote(entry.user_vote ?? null);
    });

    return () => cancelAnimationFrame(frame);
  }, [entry.score, entry.user_vote]);

  function handleVoted(newScore: number, newVote: 1 | -1 | null) {
    setCurrentScore(newScore);
    setCurrentVote(newVote);
  }

  const isQuestion = entry.entry_type === "question";
  const isPinned = entry.status === "pinned";
  const isAnswered = entry.status === "answered";
  const isArchived = entry.status === "archived";
  const isHidden = entry.is_hidden;

  const canVote =
    isQuestion && !isArchived && !isHidden && !!audienceUid && !sessionEnded;

  return (
    <div
      className={`group relative rounded-xl border transition-all ${
        isPinned
          ? "border-amber-400/40 bg-amber-50/30 dark:border-amber-500/20 dark:bg-amber-950/10"
          : isHidden
            ? "border-border/50 bg-muted/30 opacity-60"
            : "border-border bg-card hover:border-border/80 hover:shadow-sm"
      }`}
    >
      {/* Pinned indicator strip */}
      {isPinned && (
        <div className="absolute top-0 left-0 right-0 h-0.5 rounded-t-xl bg-linear-to-r from-amber-400 to-amber-500" />
      )}
      {isAnswered && (
        <div className="absolute top-0 left-0 right-0 h-0.5 rounded-t-xl bg-linear-to-r from-emerald-400 to-emerald-500" />
      )}

      <div className="flex gap-3 p-4">
        {/* Vote column — only for questions */}
        {isQuestion ? (
          <div className="shrink-0 pt-0.5">
            {canVote ? (
              <QAVoteArrows
                entryId={entry.id}
                sessionCode={sessionCode}
                audienceUid={audienceUid}
                score={currentScore}
                userVote={currentVote}
                disabled={sessionEnded}
                onVoted={handleVoted}
              />
            ) : (
              <div className="flex flex-col items-center">
                <span className="text-sm font-bold tabular-nums text-muted-foreground min-w-6 text-center">
                  {currentScore}
                </span>
                <span className="text-[10px] text-muted-foreground/60">
                  votes
                </span>
              </div>
            )}
          </div>
        ) : null}

        {/* Content */}
        <div className="flex-1 min-w-0 space-y-2">
          {/* Badges row */}
          <div className="flex items-center gap-1.5 flex-wrap">
            {isQuestion ? (
              <Badge
                variant="outline"
                className="gap-1 text-[10px] font-semibold uppercase tracking-wider border-primary/30 text-primary"
              >
                <MessageCircleQuestion className="h-3 w-3" />
                Question
              </Badge>
            ) : (
              <Badge
                variant="outline"
                className="gap-1 text-[10px] font-semibold uppercase tracking-wider"
              >
                <MessageSquareText className="h-3 w-3" />
                Comment
              </Badge>
            )}

            {isPinned && (
              <Badge
                variant="secondary"
                className="gap-1 text-[10px] font-semibold uppercase tracking-wider bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400 border-0"
              >
                <Pin className="h-3 w-3" />
                Pinned
              </Badge>
            )}

            {isAnswered && (
              <Badge
                variant="secondary"
                className="gap-1 text-[10px] font-semibold uppercase tracking-wider bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400 border-0"
              >
                <CheckCircle2 className="h-3 w-3" />
                Answered
              </Badge>
            )}

            {isHidden && isHost && (
              <Badge
                variant="destructive"
                className="gap-1 text-[10px] font-semibold uppercase tracking-wider"
              >
                <EyeOff className="h-3 w-3" />
                Hidden
              </Badge>
            )}

            {isArchived && (
              <Badge
                variant="secondary"
                className="gap-1 text-[10px] font-semibold uppercase tracking-wider"
              >
                Archived
              </Badge>
            )}
          </div>

          {/* Body text */}
          <p className="text-sm leading-relaxed text-foreground/90 whitespace-pre-wrap wrap-break-word">
            {entry.body}
          </p>

          {/* Footer: timestamp + moderation */}
          <div className="flex items-center justify-between gap-2">
            <span className="flex items-center gap-1 text-[11px] text-muted-foreground/70">
              <Clock className="h-3 w-3" />
              {relativeTime(entry.created_at)}
            </span>

            {isHost && token && !sessionEnded ? (
              <QAModerationActions
                entryId={entry.id}
                sessionCode={sessionCode}
                token={token}
                status={entry.status}
                isHidden={entry.is_hidden}
                onModerated={onUpdated}
              />
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}

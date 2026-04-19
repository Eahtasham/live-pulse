"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  CheckCircle2,
  Pin,
  EyeOff,
  Eye,
  Archive,
  Loader2,
} from "lucide-react";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  entryId: string;
  sessionCode: string;
  token: string;
  status: string;
  isHidden: boolean;
  onModerated: () => void;
}

export function QAModerationActions({
  entryId,
  sessionCode,
  token,
  status,
  isHidden,
  onModerated,
}: Props) {
  const [loading, setLoading] = useState<string | null>(null);

  async function moderate(body: { status?: string; is_hidden?: boolean }, actionKey: string) {
    setLoading(actionKey);
    try {
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/qa/${entryId}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify(body),
        }
      );
      if (res.ok) onModerated();
    } finally {
      setLoading(null);
    }
  }

  const isLoading = loading !== null;

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {status !== "answered" && (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => moderate({ status: "answered" }, "answer")}
          disabled={isLoading}
          className="h-7 gap-1.5 px-2 text-xs text-accent-foreground hover:bg-accent hover:text-accent-foreground"
        >
          {loading === "answer" ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <CheckCircle2 className="h-3.5 w-3.5" />
          )}
          Answer
        </Button>
      )}

      {status !== "pinned" ? (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => moderate({ status: "pinned" }, "pin")}
          disabled={isLoading}
          className="h-7 gap-1.5 px-2 text-xs hover:bg-amber-500/10 hover:text-amber-600 dark:hover:text-amber-400"
        >
          {loading === "pin" ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Pin className="h-3.5 w-3.5" />
          )}
          Pin
        </Button>
      ) : (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => moderate({ status: "visible" }, "unpin")}
          disabled={isLoading}
          className="h-7 gap-1.5 px-2 text-xs text-amber-600 hover:bg-amber-500/10 dark:text-amber-400"
        >
          {loading === "unpin" ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Pin className="h-3.5 w-3.5" />
          )}
          Unpin
        </Button>
      )}

      {!isHidden ? (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => moderate({ is_hidden: true }, "hide")}
          disabled={isLoading}
          className="h-7 gap-1.5 px-2 text-xs hover:bg-destructive/10 hover:text-destructive"
        >
          {loading === "hide" ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <EyeOff className="h-3.5 w-3.5" />
          )}
          Hide
        </Button>
      ) : (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => moderate({ is_hidden: false }, "unhide")}
          disabled={isLoading}
          className="h-7 gap-1.5 px-2 text-xs text-destructive hover:bg-destructive/10"
        >
          {loading === "unhide" ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Eye className="h-3.5 w-3.5" />
          )}
          Unhide
        </Button>
      )}

      {status !== "archived" && (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => moderate({ status: "archived" }, "archive")}
          disabled={isLoading}
          className="h-7 gap-1.5 px-2 text-xs hover:bg-muted hover:text-muted-foreground"
        >
          {loading === "archive" ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Archive className="h-3.5 w-3.5" />
          )}
          Archive
        </Button>
      )}
    </div>
  );
}

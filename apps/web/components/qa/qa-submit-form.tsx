"use client";

import { useState, useRef, useEffect } from "react";
import { MessageCircleQuestion, MessageSquareText, Send } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Spinner } from "@/components/ui/Spinner";
import { Toast } from "@/components/ui/Toast";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const MAX_CHARS = 2000;

interface Props {
  sessionCode: string;
  audienceUid: string;
  onSubmitted: (entry: import("@/lib/qa").QAEntry) => void;
  disabled?: boolean;
}

export function QASubmitForm({ sessionCode, audienceUid, onSubmitted, disabled = false }: Props) {
  const [entryType, setEntryType] = useState<"question" | "comment">("question");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    textareaRef.current?.focus();
  }, []);

  const charCount = body.length;
  const charRatio = charCount / MAX_CHARS;
  const isOverLimit = charCount > MAX_CHARS;
  const canSubmit = body.trim().length > 0 && !isOverLimit && !loading && !disabled;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;
    setLoading(true);
    setError("");

    try {
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/qa`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Audience-UID": audienceUid,
          },
          body: JSON.stringify({ entry_type: entryType, body: body.trim() }),
        }
      );

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.message || "Failed to submit");
        return;
      }

      const entry = await res.json();
      setBody("");
      onSubmitted(entry);
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  if (disabled) {
    return (
      <Toast
        variant="warning"
        title="Session closed"
        description="New questions and comments are disabled now that the host has ended the session."
      />
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div className="flex items-center gap-1 rounded-lg bg-muted/60 p-1">
        <button
          type="button"
          onClick={() => setEntryType("question")}
          className={`flex flex-1 items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-all ${
            entryType === "question"
              ? "bg-primary text-primary-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <MessageCircleQuestion className="h-4 w-4" />
          Question
        </button>
        <button
          type="button"
          onClick={() => setEntryType("comment")}
          className={`flex flex-1 items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-all ${
            entryType === "comment"
              ? "bg-secondary text-secondary-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <MessageSquareText className="h-4 w-4" />
          Comment
        </button>
      </div>

      <div className="relative">
        <Textarea
          ref={textareaRef}
          placeholder={
            entryType === "question"
              ? "What would you like to ask?"
              : "Share your thoughts..."
          }
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={3}
          className="resize-none pr-4 pb-8 text-sm"
        />
        <div className="absolute right-3 bottom-2 flex items-center gap-2">
          <span
            className={`text-xs font-mono tabular-nums transition-colors ${
              isOverLimit
                ? "font-semibold text-destructive"
                : charRatio > 0.9
                  ? "text-amber-500"
                  : "text-muted-foreground"
            }`}
          >
            {charCount}/{MAX_CHARS}
          </span>
        </div>
      </div>

      {error ? <Toast variant="error" description={error} /> : null}

      <div className="flex justify-end">
        <Button type="submit" disabled={!canSubmit} size="sm" className="gap-2">
          {loading ? <Spinner className="size-4" /> : <Send className="h-4 w-4" />}
          {loading ? "Sending..." : "Submit"}
        </Button>
      </div>
    </form>
  );
}

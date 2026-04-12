"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  sessionCode: string;
  audienceUid: string;
  onSubmitted: () => void;
  onCancel?: () => void;
}

export function QAForm({ sessionCode, audienceUid, onSubmitted, onCancel }: Props) {
  const [entryType, setEntryType] = useState<"question" | "comment">("question");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!body.trim()) return;

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
          body: JSON.stringify({
            entry_type: entryType,
            body: body.trim(),
          }),
        }
      );

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.message || "Failed to submit");
        return;
      }

      setBody("");
      onSubmitted();
    } catch {
      setError("Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label>Type</Label>
        <div className="flex gap-2">
          <Button
            type="button"
            variant={entryType === "question" ? "default" : "outline"}
            onClick={() => setEntryType("question")}
            className="flex-1"
          >
            Question
          </Button>
          <Button
            type="button"
            variant={entryType === "comment" ? "default" : "outline"}
            onClick={() => setEntryType("comment")}
            className="flex-1"
          >
            Comment
          </Button>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="body">
          {entryType === "question" ? "Your Question" : "Your Comment"}
        </Label>
        <Input
          id="body"
          value={body}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setBody(e.target.value)}
          placeholder={
            entryType === "question"
              ? "What would you like to ask?"
              : "Share your thoughts..."
          }
          maxLength={2000}
        />
        <p className="text-xs text-muted-foreground text-right">
          {body.length}/2000
        </p>
      </div>

      {error && <p className="text-sm text-red-500">{error}</p>}

      <div className="flex gap-2">
        <Button type="submit" disabled={loading || !body.trim()}>
          {loading ? "Submitting..." : "Submit"}
        </Button>
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        )}
      </div>
    </form>
  );
}

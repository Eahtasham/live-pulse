"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  sessionCode: string;
  token?: string;
  onCreated: () => void;
  onCancel: () => void;
}

interface OptionInput {
  label: string;
  position: number;
}

export function CreatePollForm({
  sessionCode,
  token,
  onCreated,
  onCancel,
}: Props) {
  const [question, setQuestion] = useState("");
  const [answerMode, setAnswerMode] = useState<"single" | "multi">("single");
  const [timeLimitSec, setTimeLimitSec] = useState("");
  const [options, setOptions] = useState<OptionInput[]>([
    { label: "", position: 0 },
    { label: "", position: 1 },
  ]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  function addOption() {
    if (options.length >= 6) return;
    setOptions([...options, { label: "", position: options.length }]);
  }

  function removeOption(index: number) {
    if (options.length <= 2) return;
    const updated = options
      .filter((_, i) => i !== index)
      .map((o, i) => ({ ...o, position: i }));
    setOptions(updated);
  }

  function updateOptionLabel(index: number, label: string) {
    const updated = [...options];
    updated[index] = { ...updated[index], label };
    setOptions(updated);
  }

  const canSubmit =
    question.trim().length > 0 &&
    options.length >= 2 &&
    options.every((o) => o.label.trim().length > 0) &&
    !loading;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;
    setLoading(true);
    setError("");

    try {
      const body: Record<string, unknown> = {
        question: question.trim(),
        answer_mode: answerMode,
        options: options.map((o) => ({
          label: o.label.trim(),
          position: o.position,
        })),
      };

      const tl = parseInt(timeLimitSec, 10);
      if (!isNaN(tl) && tl > 0) {
        body.time_limit_sec = tl;
      }

      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/polls`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify(body),
        }
      );

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.message || "Failed to create poll");
        return;
      }

      onCreated();
    } catch {
      setError("Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && (
        <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-950 dark:text-red-400">
          {error}
        </p>
      )}

      <div className="space-y-2">
        <Label htmlFor="poll-question">Question</Label>
        <Input
          id="poll-question"
          placeholder="Enter your question..."
          value={question}
          onChange={(e) => setQuestion(e.target.value.slice(0, 500))}
          required
          autoFocus
        />
        <p className="text-xs text-muted-foreground">
          {question.length}/500 characters
        </p>
      </div>

      <div className="space-y-2">
        <Label>Options</Label>
        {options.map((option, index) => (
          <div key={index} className="flex items-center gap-2">
            <Input
              placeholder={`Option ${index + 1}`}
              value={option.label}
              onChange={(e) =>
                updateOptionLabel(index, e.target.value.slice(0, 200))
              }
              required
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => removeOption(index)}
              disabled={options.length <= 2}
              aria-label="Remove option"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M18 6 6 18" />
                <path d="m6 6 12 12" />
              </svg>
            </Button>
          </div>
        ))}
        {options.length < 6 && (
          <Button type="button" variant="outline" size="sm" onClick={addOption}>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M5 12h14" />
              <path d="M12 5v14" />
            </svg>
            Add Option
          </Button>
        )}
      </div>

      <div className="space-y-2">
        <Label>Answer Mode</Label>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => setAnswerMode("single")}
            className={`flex-1 rounded-lg border px-3 py-2 text-sm font-medium transition-colors ${
              answerMode === "single"
                ? "border-primary bg-primary/10 text-primary"
                : "border-border text-muted-foreground hover:bg-muted"
            }`}
          >
            Single answer
          </button>
          <button
            type="button"
            onClick={() => setAnswerMode("multi")}
            className={`flex-1 rounded-lg border px-3 py-2 text-sm font-medium transition-colors ${
              answerMode === "multi"
                ? "border-primary bg-primary/10 text-primary"
                : "border-border text-muted-foreground hover:bg-muted"
            }`}
          >
            Multi-select
          </button>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="time-limit">Time limit (seconds, optional)</Label>
        <Input
          id="time-limit"
          type="number"
          min="1"
          placeholder="No limit"
          value={timeLimitSec}
          onChange={(e) => setTimeLimitSec(e.target.value)}
        />
      </div>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={!canSubmit}>
          {loading ? "Creating..." : "Create Poll"}
        </Button>
      </div>
    </form>
  );
}

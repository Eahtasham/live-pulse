"use client";

import { useEffect, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { QAForm } from "@/components/qa/qa-form";
import { QACard } from "@/components/qa/qa-card";
import type { QAEntry, QAListResponse } from "@/lib/qa";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  sessionCode: string;
  isHost: boolean;
  token?: string;
  audienceUid: string;
}

export function QAList({ sessionCode, isHost, token, audienceUid }: Props) {
  const [entries, setEntries] = useState<QAEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [nextCursor, setNextCursor] = useState<string>("");

  const fetchEntries = useCallback(async () => {
    try {
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/qa?limit=50`,
        {
          headers: {
            "X-Audience-UID": audienceUid,
          },
        }
      );

      if (res.ok) {
        const data: QAListResponse = await res.json();
        setEntries(data.entries ?? []);
        setNextCursor(data.next_cursor ?? "");
      }
    } finally {
      setLoading(false);
    }
  }, [sessionCode, audienceUid]);

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries]);

  function handleSubmitted() {
    setShowForm(false);
    fetchEntries();
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading Q&A...</p>;
  }

  return (
    <div className="space-y-4">
      {/* Submit Q&A button */}
      {showForm ? (
        <Card>
          <CardHeader>
            <CardTitle>Ask a Question</CardTitle>
          </CardHeader>
          <CardContent>
            <QAForm
              sessionCode={sessionCode}
              audienceUid={audienceUid}
              onSubmitted={handleSubmitted}
              onCancel={() => setShowForm(false)}
            />
          </CardContent>
        </Card>
      ) : (
        <Button onClick={() => setShowForm(true)}>
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
            <path d="M5 12h14" />
            <path d="M12 5v14" />
          </svg>
          Ask a Question
        </Button>
      )}

      {/* Q&A entries list */}
      {entries.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <p className="text-sm text-muted-foreground">
              No questions yet. Be the first to ask!
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {entries.map((entry) => (
            <QACard
              key={entry.id}
              entry={entry}
              sessionCode={sessionCode}
              audienceUid={audienceUid}
              isHost={isHost}
              token={token}
              onUpdated={fetchEntries}
            />
          ))}
          {nextCursor && (
            <Button variant="outline" className="w-full">
              Load More
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

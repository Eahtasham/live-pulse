"use client";

import { useEffect, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { QASubmitForm } from "@/components/qa/qa-submit-form";
import { QAEntryCard } from "@/components/qa/qa-entry";
import {
  MessageCircleQuestion,
  ChevronDown,
  Loader2,
  Inbox,
} from "lucide-react";
import type { QAEntry, QAListResponse } from "@/lib/qa";
import { Spinner } from "@/components/ui/Spinner";
import { Toast } from "@/components/ui/Toast";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const PAGE_SIZE = 20;

interface Props {
  sessionCode: string;
  isHost: boolean;
  token?: string;
  audienceUid: string;
  sessionEnded?: boolean;
  onRegisterCallbacks?: (callbacks: {
    addEntry: (entry: QAEntry) => void;
    updateEntry: (id: string, updates: Partial<QAEntry>) => void;
    replaceEntries: (entries: QAEntry[]) => void;
  }) => void;
}

export function QAFeed({ sessionCode, isHost, token, audienceUid, sessionEnded = false, onRegisterCallbacks }: Props) {
  const [entries, setEntries] = useState<QAEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [nextCursor, setNextCursor] = useState("");
  const [hostTab, setHostTab] = useState<"active" | "archived">("active");
  const [showForm, setShowForm] = useState(false);

  const fetchEntries = useCallback(
    async (cursor?: string) => {
      const isMore = !!cursor;
      if (isMore) setLoadingMore(true);
      else setLoading(true);

      try {
        const params = new URLSearchParams({ limit: String(PAGE_SIZE) });
        if (cursor) params.set("cursor", cursor);

        const headers: Record<string, string> = {};
        if (audienceUid) headers["X-Audience-UID"] = audienceUid;

        const res = await fetch(
          `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/qa?${params}`,
          { headers }
        );

        if (res.ok) {
          const data: QAListResponse = await res.json();
          const newEntries = data.entries ?? [];
          if (isMore) {
            setEntries((prev) => [...prev, ...newEntries]);
          } else {
            setEntries(newEntries);
          }
          setNextCursor(data.next_cursor ?? "");
        }
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [sessionCode, audienceUid]
  );

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries]);

  // Register live update callbacks with parent
  const addEntry = useCallback((entry: QAEntry) => {
    setEntries((prev) => {
      // Deduplicate — if entry already exists, skip
      if (prev.some((e) => e.id === entry.id)) return prev;
      return [...prev, entry];
    });
  }, []);

  const updateEntry = useCallback((id: string, updates: Partial<QAEntry>) => {
    setEntries((prev) =>
      prev.map((e) => (e.id === id ? { ...e, ...updates } : e))
    );
  }, []);

  const replaceEntries = useCallback((nextEntries: QAEntry[]) => {
    setEntries(nextEntries);
  }, []);

  useEffect(() => {
    onRegisterCallbacks?.({ addEntry, updateEntry, replaceEntries });
  }, [onRegisterCallbacks, addEntry, updateEntry, replaceEntries]);

  useEffect(() => {
    if (sessionEnded) setShowForm(false);
  }, [sessionEnded]);

  function handleSubmitted(newEntry: QAEntry) {
    // Optimistic: insert at the end (it'll sort correctly on next fetch)
    setEntries((prev) => [...prev, newEntry]);
    setShowForm(false);
  }

  function handleLoadMore() {
    if (nextCursor) fetchEntries(nextCursor);
  }

  // Sort: pinned first, then by score desc, then creation time
  const sortedEntries = [...entries].sort((a, b) => {
    const aPinned = a.status === "pinned" ? 1 : 0;
    const bPinned = b.status === "pinned" ? 1 : 0;
    if (aPinned !== bPinned) return bPinned - aPinned;
    if (a.score !== b.score) return b.score - a.score;
    return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
  });

  // Filter for host tabs
  const visibleEntries = isHost
    ? sortedEntries.filter((e) =>
        hostTab === "active"
          ? e.status !== "archived"
          : e.status === "archived"
      )
    : sortedEntries;

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner label="Loading Q&amp;A" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {sessionEnded ? (
        <Toast
          variant="warning"
          description="This session has ended. Questions, comments, and votes are now read only."
        />
      ) : null}

      {/* Submit form toggle */}
      {!sessionEnded && showForm ? (
        <div className="rounded-xl border border-primary/20 bg-card p-4 shadow-sm">
          <QASubmitForm
            sessionCode={sessionCode}
            audienceUid={audienceUid}
            disabled={sessionEnded}
            onSubmitted={handleSubmitted}
          />
          <button
            onClick={() => setShowForm(false)}
            className="mt-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            Cancel
          </button>
        </div>
      ) : !sessionEnded ? (
        <button
          onClick={() => setShowForm(true)}
          className="flex w-full items-center gap-3 rounded-xl border border-dashed border-border bg-card/50 px-4 py-3 text-sm text-muted-foreground transition-all hover:border-primary/40 hover:bg-card hover:text-foreground hover:shadow-sm"
        >
          <MessageCircleQuestion className="h-5 w-5 text-primary/60" />
          Ask a question or leave a comment...
        </button>
      ) : null}

      {/* Host tab bar */}
      {isHost && (
        <Tabs
          value={hostTab}
          onValueChange={(v) => setHostTab(v as "active" | "archived")}
        >
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="active" className="text-sm">
              Active
              <span className="ml-1.5 rounded-full bg-primary/10 px-1.5 py-0.5 text-[10px] font-bold text-primary tabular-nums">
                {sortedEntries.filter((e) => e.status !== "archived").length}
              </span>
            </TabsTrigger>
            <TabsTrigger value="archived" className="text-sm">
              Archived
              <span className="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-bold text-muted-foreground tabular-nums">
                {sortedEntries.filter((e) => e.status === "archived").length}
              </span>
            </TabsTrigger>
          </TabsList>
        </Tabs>
      )}

      {/* Entries list */}
      {visibleEntries.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-12 text-center">
          <Inbox className="h-10 w-10 text-muted-foreground/40" />
          <p className="mt-3 text-sm font-medium text-muted-foreground">
            {isHost && hostTab === "archived"
              ? "No archived entries"
              : "No questions yet"}
          </p>
          <p className="mt-1 text-xs text-muted-foreground/60">
            {isHost && hostTab === "archived"
              ? "Archived entries will appear here"
              : "Be the first to ask something!"}
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {visibleEntries.map((entry) => (
            <QAEntryCard
              key={entry.id}
              entry={entry}
              sessionCode={sessionCode}
              audienceUid={audienceUid}
              isHost={isHost}
              token={token}
              sessionEnded={sessionEnded}
              onUpdated={fetchEntries}
            />
          ))}
        </div>
      )}

      {/* Load more */}
      {nextCursor && (
        <Button
          variant="outline"
          className="w-full gap-2"
          onClick={handleLoadMore}
          disabled={loadingMore}
        >
          {loadingMore ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <ChevronDown className="h-4 w-4" />
          )}
          {loadingMore ? "Loading..." : "Load more"}
        </Button>
      )}
    </div>
  );
}

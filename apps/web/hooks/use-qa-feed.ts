"use client";

import { useEffect, useCallback, useRef } from "react";
import type { useWebSocket, WSMessage } from "@/hooks/use-websocket";
import type { QAEntry } from "@/lib/qa";

interface NewQuestionPayload {
  id: string;
  entry_type: "question";
  body: string;
  score: number;
  author_uid: string;
  created_at: string;
}

interface NewCommentPayload {
  id: string;
  entry_type: "comment";
  body: string;
  author_uid: string;
  created_at: string;
}

interface QAUpdatePayload {
  id: string;
  status: string;
  is_hidden: boolean;
  score: number;
}

interface QAFeedCallbacks {
  onNewEntry: (entry: QAEntry) => void;
  onEntryUpdate: (id: string, updates: Partial<QAEntry>) => void;
}

export function useQAFeed(
  ws: ReturnType<typeof useWebSocket>,
  sessionCode: string,
  audienceUid: string,
  callbacks: QAFeedCallbacks
) {
  const cbRef = useRef(callbacks);
  cbRef.current = callbacks;

  // Buffer for coalescing rapid updates, flush every 500ms
  const newEntriesBuffer = useRef<Map<string, QAEntry>>(new Map());
  const updatesBuffer = useRef<Map<string, Partial<QAEntry>>>(new Map());
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  function scheduleFlush() {
    if (timerRef.current !== null) return;
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      const newEntries = Array.from(newEntriesBuffer.current.values());
      const updates = Array.from(updatesBuffer.current.entries());
      newEntriesBuffer.current.clear();
      updatesBuffer.current.clear();
      for (const entry of newEntries) {
        cbRef.current.onNewEntry(entry);
      }
      for (const [id, partial] of updates) {
        cbRef.current.onEntryUpdate(id, partial);
      }
    }, 500);
  }

  useEffect(() => {
    return ws.subscribe((msg: WSMessage) => {
      if (msg.type === "new_question") {
        const p = msg.payload as NewQuestionPayload;
        newEntriesBuffer.current.set(p.id, {
          id: p.id,
          session_id: "",
          author_uid: p.author_uid,
          entry_type: "question",
          body: p.body,
          score: p.score,
          status: "visible",
          is_hidden: false,
          created_at: p.created_at,
          updated_at: p.created_at,
        });
        scheduleFlush();
      }

      if (msg.type === "new_comment") {
        const p = msg.payload as NewCommentPayload;
        newEntriesBuffer.current.set(p.id, {
          id: p.id,
          session_id: "",
          author_uid: p.author_uid,
          entry_type: "comment",
          body: p.body,
          score: 0,
          status: "visible",
          is_hidden: false,
          created_at: p.created_at,
          updated_at: p.created_at,
        });
        scheduleFlush();
      }

      if (msg.type === "qa_update") {
        const p = msg.payload as QAUpdatePayload;
        // Merge with any existing buffered update for same entry
        const existing = updatesBuffer.current.get(p.id) ?? {};
        updatesBuffer.current.set(p.id, {
          ...existing,
          status: p.status as QAEntry["status"],
          is_hidden: p.is_hidden,
          score: p.score,
        });
        scheduleFlush();
      }
    });
  }, [ws]);

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current);
    };
  }, []);

  // On reconnect, refetch full Q&A list
  const refetch = useCallback(async () => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    try {
      const headers: Record<string, string> = {};
      if (audienceUid) headers["X-Audience-UID"] = audienceUid;
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(sessionCode)}/qa?limit=100`,
        { headers }
      );
      if (res.ok) {
        const data = await res.json();
        // Fully replace entries via individual updates — not ideal but ensures consistency
        const entries: QAEntry[] = data.entries ?? [];
        for (const entry of entries) {
          cbRef.current.onNewEntry(entry);
        }
      }
    } catch {
      // silently fail
    }
  }, [sessionCode, audienceUid]);

  useEffect(() => {
    return ws.onReconnect(refetch);
  }, [ws, refetch]);
}

"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useSession } from "next-auth/react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PollList } from "@/components/poll/poll-list";
import { QAFeed } from "@/components/qa/qa-feed";
import { ShareModal, ShareSessionButton } from "@/components/session/share-modal";
import { useWebSocket } from "@/hooks/use-websocket";
import { useSessionStatus } from "@/hooks/use-session-status";
import { usePollVotes } from "@/hooks/use-poll-votes";
import { useQAFeed } from "@/hooks/use-qa-feed";
import { getStableClientId } from "@/lib/fingerprint";
import type { PollOption } from "@/lib/poll";
import type { QAEntry } from "@/lib/qa";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface SessionData {
  id: string;
  code: string;
  title: string;
  status: string;
  host_id?: string;
}

export function SessionJoinView({ code }: { code: string }) {
  const { data: authSession } = useSession();
  const [pageStatus, setPageStatus] = useState<"loading" | "joined" | "error">(
    "loading"
  );
  const [session, setSession] = useState<SessionData | null>(null);
  const [audienceUid, setAudienceUid] = useState("");
  const [error, setError] = useState("");
  const [isHost, setIsHost] = useState(false);
  const [activeTab, setActiveTab] = useState<"polls" | "qa">("polls");
  const [confirmEnd, setConfirmEnd] = useState(false);
  const [ending, setEnding] = useState(false);
  const [showShareAfterVote, setShowShareAfterVote] = useState(false);

  // WebSocket connection
  const ws = useWebSocket(code);
  const { sessionEnded } = useSessionStatus(ws);

  // Refs for child updaters
  const pollUpdaterRef = useRef<((pollId: string, options: PollOption[]) => void) | null>(null);
  const qaCallbacksRef = useRef<{
    addEntry: (entry: QAEntry) => void;
    updateEntry: (id: string, updates: Partial<QAEntry>) => void;
  } | null>(null);

  // Wire poll votes hook
  usePollVotes(ws, code, authSession?.apiToken, {
    onVoteUpdate: (pollId, options) => {
      pollUpdaterRef.current?.(pollId, options);
    },
  });

  // Wire QA feed hook
  useQAFeed(ws, code, audienceUid, {
    onNewEntry: (entry) => {
      qaCallbacksRef.current?.addEntry(entry);
    },
    onEntryUpdate: (id, updates) => {
      qaCallbacksRef.current?.updateEntry(id, updates);
    },
  });

  const handlePollRegister = useCallback(
    (updater: (pollId: string, options: PollOption[]) => void) => {
      pollUpdaterRef.current = updater;
    },
    []
  );

  const handleQARegister = useCallback(
    (cbs: { addEntry: (entry: QAEntry) => void; updateEntry: (id: string, updates: Partial<QAEntry>) => void }) => {
      qaCallbacksRef.current = cbs;
    },
    []
  );

  async function handleEndSession() {
    if (!authSession?.apiToken) return;
    setEnding(true);
    try {
      const res = await fetch(
        `${apiUrl}/v1/sessions/${encodeURIComponent(code)}/close`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${authSession.apiToken}`,
          },
        }
      );
      if (res.ok) {
        setSession((prev) => (prev ? { ...prev, status: "closed" } : prev));
        setConfirmEnd(false);
      }
    } finally {
      setEnding(false);
    }
  }

  const checkHost = useCallback(
    (sessionData: SessionData) => {
      if (authSession?.apiToken && sessionData.host_id) {
        try {
          const payload = JSON.parse(
            atob(authSession.apiToken.split(".")[1])
          );
          if (payload.user_id === sessionData.host_id) {
            setIsHost(true);
          }
        } catch {
          // ignore decode errors
        }
      }
    },
    [authSession?.apiToken]
  );

  useEffect(() => {
    async function join() {
      try {
        // First get session details
        const sessionRes = await fetch(
          `${apiUrl}/v1/sessions/${encodeURIComponent(code)}`,
          { headers: { "Content-Type": "application/json" } }
        );

        if (!sessionRes.ok) {
          setError("Session not found");
          setPageStatus("error");
          return;
        }

        const sessionData: SessionData = await sessionRes.json();
        setSession(sessionData);
        checkHost(sessionData);

        // Join the session with a device fingerprint (survives incognito)
        const clientId = await getStableClientId();
        const joinRes = await fetch(
          `${apiUrl}/v1/sessions/${encodeURIComponent(code)}/join`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "X-Client-ID": clientId,
            },
          }
        );

        if (!joinRes.ok) {
          setError("Session not found");
          setPageStatus("error");
          return;
        }

        const joinData = await joinRes.json();
        setAudienceUid(joinData.audience_uid);
        setPageStatus("joined");
      } catch {
        setError("Unable to connect");
        setPageStatus("error");
      }
    }

    join();
  }, [code, checkHost]);

  useEffect(() => {
    if (session) checkHost(session);
  }, [authSession, session, checkHost]);

  if (pageStatus === "loading") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-muted-foreground">Joining session...</p>
      </div>
    );
  }

  if (pageStatus === "error") {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4">
        <h1 className="text-2xl font-bold text-destructive">{error}</h1>
        <p className="text-muted-foreground">
          Check your session code and try again.
        </p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Session header */}
      <div className="border-b border-border bg-card">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-4 py-4">
          <div>
            <h1 className="text-xl font-bold">{session?.title}</h1>
            <div className="mt-1 flex items-center gap-2">
              <span className="font-mono text-sm font-semibold tracking-wider text-muted-foreground">
                {code}
              </span>
              <Badge
                variant={
                  session?.status === "active" ? "default" : "secondary"
                }
              >
                {session?.status}
              </Badge>
              {isHost && <Badge variant="outline">Host</Badge>}
              {/* Connection indicator */}
              <span
                className={`inline-block h-2 w-2 rounded-full ${
                  ws.state === "connected"
                    ? "bg-green-500"
                    : ws.state === "connecting"
                      ? "bg-yellow-500 animate-pulse"
                      : "bg-red-500"
                }`}
                title={ws.state}
              />
            </div>
          </div>
          {isHost && !sessionEnded && session?.status === "active" && (
            <div className="flex items-center gap-2">
              {confirmEnd ? (
                <>
                  <span className="text-sm text-muted-foreground">End session?</span>
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={handleEndSession}
                    disabled={ending}
                  >
                    {ending ? "Ending..." : "Confirm"}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setConfirmEnd(false)}
                  >
                    Cancel
                  </Button>
                </>
              ) : (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setConfirmEnd(true)}
                >
                  End Session
                </Button>
              )}
            </div>
          )}
          {!isHost && (
            <div className="flex items-center gap-2">
              <ShareSessionButton
                sessionCode={code}
                sessionTitle={session?.title ?? ""}
              />
              <p className="hidden sm:block text-xs text-muted-foreground">
                ID:{" "}
                <code className="font-mono">{audienceUid.slice(0, 8)}…</code>
              </p>
            </div>
          )}
          {isHost && (
            <ShareSessionButton
              sessionCode={code}
              sessionTitle={session?.title ?? ""}
            />
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-border bg-card">
        <div className="mx-auto flex max-w-4xl px-4">
          <button
            onClick={() => setActiveTab("polls")}
            className={`border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
              activeTab === "polls"
                ? "border-primary text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            Polls
          </button>
          <button
            onClick={() => setActiveTab("qa")}
            className={`border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
              activeTab === "qa"
                ? "border-primary text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            Q&amp;A
          </button>
        </div>
      </div>

      {/* Tab content */}
      <div className="mx-auto max-w-4xl px-4 py-6">
        {sessionEnded && (
          <div className="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-6 text-center">
            <p className="text-lg font-semibold text-destructive">
              Session has ended
            </p>
            <p className="mt-1 text-sm text-muted-foreground">
              The host has closed this session. No further interactions are possible.
            </p>
          </div>
        )}

        {activeTab === "polls" && (
          <PollList
            sessionCode={code}
            isHost={isHost}
            token={authSession?.apiToken}
            audienceUid={audienceUid}
            onRegisterUpdater={handlePollRegister}
            onAnyVote={() => setShowShareAfterVote(true)}
          />
        )}
        {activeTab === "qa" && (
          <QAFeed
            sessionCode={code}
            isHost={isHost}
            token={authSession?.apiToken}
            audienceUid={audienceUid}
            onRegisterCallbacks={handleQARegister}
          />
        )}
      </div>

      {/* Share modal shown after voting */}
      <ShareModal
        open={showShareAfterVote}
        onOpenChange={setShowShareAfterVote}
        sessionCode={code}
        sessionTitle={session?.title ?? ""}
        afterVote
      />
    </div>
  );
}

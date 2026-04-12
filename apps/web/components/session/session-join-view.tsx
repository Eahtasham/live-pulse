"use client";

import { useEffect, useState, useCallback } from "react";
import { useSession } from "next-auth/react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PollList } from "@/components/poll/poll-list";
import { QAList } from "@/components/qa/qa-list";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

function getClientId(): string {
  const key = "livepulse_client_id";
  let id = localStorage.getItem(key);
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem(key, id);
  }
  return id;
}

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

        // Join the session
        const clientId = getClientId();
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
            </div>
          </div>
          {!isHost && (
            <p className="text-xs text-muted-foreground">
              ID:{" "}
              <code className="font-mono">{audienceUid.slice(0, 8)}…</code>
            </p>
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
        {activeTab === "polls" && (
          <PollList
            sessionCode={code}
            isHost={isHost}
            token={authSession?.apiToken}
            audienceUid={audienceUid}
          />
        )}
        {activeTab === "qa" && (
          <QAList
            sessionCode={code}
            isHost={isHost}
            token={authSession?.apiToken}
            audienceUid={audienceUid}
          />
        )}
      </div>
    </div>
  );
}

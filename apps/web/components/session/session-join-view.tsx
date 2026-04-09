"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

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

export function SessionJoinView({ code }: { code: string }) {
  const [status, setStatus] = useState<"loading" | "joined" | "error">(
    "loading"
  );
  const [sessionTitle, setSessionTitle] = useState("");
  const [audienceUid, setAudienceUid] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    async function join() {
      try {
        const clientId = getClientId();
        const res = await fetch(
          `${apiUrl}/v1/sessions/${encodeURIComponent(code)}/join`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "X-Client-ID": clientId,
            },
          }
        );

        if (!res.ok) {
          setError("Session not found");
          setStatus("error");
          return;
        }

        const data = await res.json();
        setSessionTitle(data.session_title);
        setAudienceUid(data.audience_uid);
        setStatus("joined");
      } catch {
        setError("Unable to connect");
        setStatus("error");
      }
    }

    join();
  }, [code]);

  if (status === "loading") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-muted-foreground">Joining session...</p>
      </div>
    );
  }

  if (status === "error") {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4">
        <h1 className="text-2xl font-bold text-destructive">
          {error}
        </h1>
        <p className="text-muted-foreground">
          Check your session code and try again.
        </p>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">{sessionTitle}</CardTitle>
          <p className="text-sm text-muted-foreground">
            Session code:{" "}
            <span className="font-mono font-semibold tracking-wider">
              {code}
            </span>
          </p>
        </CardHeader>
        <CardContent className="space-y-4 text-center">
          <div className="rounded-lg bg-primary/10 px-4 py-3">
            <p className="text-sm font-medium text-primary">
              You&apos;ve joined the session!
            </p>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            Polls and Q&amp;A will appear here when the host starts them.
          </p>
          <p className="text-xs text-muted-foreground">
            Your audience ID:{" "}
            <code className="font-mono">{audienceUid.slice(0, 8)}…</code>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

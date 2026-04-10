"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { Session } from "@/lib/session";

export function SessionCard({ session }: { session: Session }) {
  const [copied, setCopied] = useState(false);

  const joinUrl =
    typeof window !== "undefined"
      ? `${window.location.origin}/session/${session.code}`
      : `/session/${session.code}`;

  function handleCopy() {
    navigator.clipboard.writeText(joinUrl).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-2">
        <div className="space-y-1">
          <CardTitle className="text-lg">{session.title}</CardTitle>
          <p className="text-sm text-muted-foreground">
            Code:{" "}
            <span className="font-mono font-semibold tracking-wider">
              {session.code}
            </span>
          </p>
        </div>
        <Badge variant={session.status === "active" ? "default" : "secondary"}>
          {session.status}
        </Badge>
      </CardHeader>
      <CardContent className="flex items-center gap-2">
        <code className="flex-1 truncate rounded-md bg-muted px-3 py-2 text-sm">
          {joinUrl}
        </code>
        <Button variant="outline" size="sm" onClick={handleCopy}>
          {copied ? (
            <>
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
                <path d="M20 6 9 17l-5-5" />
              </svg>
              Copied
            </>
          ) : (
            <>
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
                <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
                <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
              </svg>
              Copy Link
            </>
          )}
        </Button>
      </CardContent>
    </Card>
  );
}

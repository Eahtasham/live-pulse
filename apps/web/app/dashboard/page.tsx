"use client";

import { useSession, signOut } from "next-auth/react";
import { useEffect, useState, useCallback } from "react";
import { ThemeToggle } from "@/components/theme-toggle";
import { CreateSessionDialog } from "@/components/session/create-session-dialog";
import { SessionCard } from "@/components/session/session-card";
import type { Session } from "@/lib/session";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function DashboardPage() {
  const { data: authSession, status } = useSession();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchSessions = useCallback(async () => {
    if (!authSession?.apiToken) return;
    try {
      const res = await fetch(`${apiUrl}/v1/sessions`, {
        headers: {
          Authorization: `Bearer ${authSession.apiToken}`,
          "Content-Type": "application/json",
        },
      });
      if (res.ok) {
        const data = await res.json();
        setSessions(data ?? []);
      }
    } finally {
      setLoading(false);
    }
  }, [authSession?.apiToken]);

  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  if (status === "loading") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-zinc-500">Loading...</p>
      </div>
    );
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Host Dashboard</h1>
          <p className="mt-2 text-gray-500">
            Manage your sessions, polls, and Q&amp;A
          </p>
        </div>
        <div className="flex items-center gap-4">
          {authSession?.user && (
            <span className="text-sm text-zinc-600 dark:text-zinc-400">
              {authSession.user.email}
            </span>
          )}
          <ThemeToggle />
          <button
            onClick={() => signOut({ callbackUrl: "/login" })}
            className="rounded-lg border border-zinc-300 px-4 py-2 text-sm font-medium transition-colors hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800"
          >
            Sign out
          </button>
        </div>
      </div>

      <div className="mt-8 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Your Sessions</h2>
        <CreateSessionDialog
          token={authSession?.apiToken ?? ""}
          onCreated={fetchSessions}
        />
      </div>

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {loading ? (
          <p className="text-sm text-muted-foreground col-span-full">
            Loading sessions...
          </p>
        ) : sessions.length === 0 ? (
          <p className="text-sm text-muted-foreground col-span-full">
            No sessions yet. Create one to get started!
          </p>
        ) : (
          sessions.map((s) => <SessionCard key={s.id} session={s} />)
        )}
      </div>
    </div>
  );
}

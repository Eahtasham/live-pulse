"use client";

import { useSession, signOut } from "next-auth/react";
import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { ArrowRight, CheckCircle2, Clock3, LayoutDashboard } from "lucide-react";
import { Brand } from "@/components/brand";
import { ThemeToggle } from "@/components/theme-toggle";
import { CreateSessionDialog } from "@/components/session/create-session-dialog";
import { SessionCard } from "@/components/session/session-card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { Session } from "@/lib/session";
import { PageWrapper } from "@/components/page-wrapper";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function DashboardPage() {
  const { data: authSession, status } = useSession();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);

  const totalSessions = sessions.length;
  const activeSessions = sessions.filter((session) => session.status === "active").length;
  const inactiveSessions = totalSessions - activeSessions;

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
      <div className="flex min-h-screen items-center justify-center bg-background px-6">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <CardTitle className="text-xl">Loading dashboard</CardTitle>
            <CardDescription>
              We&apos;re checking your session and restoring your host workspace.
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    );
  }

  return (
    <main className="relative isolate min-h-screen overflow-hidden px-6 py-6 lg:px-8">
      <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        <div className="absolute left-[-12%] top-[-10%] h-80 w-80 rounded-full bg-primary/15 blur-3xl dark:bg-primary/20" />
        <div className="absolute right-[-12%] top-[8%] h-80 w-80 rounded-full bg-cyan-500/10 blur-3xl dark:bg-cyan-400/10" />
      </div>

      <PageWrapper className="flex flex-col gap-8">
        <header className="flex items-center justify-between rounded-xl border border-border/70 bg-card/85 p-4 shadow-sm backdrop-blur">
          <div className="flex items-center gap-3">
            <Brand size="sm" href="/" />
            <h1 className="text-xl font-semibold tracking-tight">Dashboard</h1>
          </div>

          <div className="flex items-center gap-3">
            {authSession?.user?.email && (
              <Badge variant="outline" className="rounded-full border-border/70 bg-background/70 px-3 py-1 text-sm">
                {authSession.user.email}
              </Badge>
            )}
            <ThemeToggle />
            <Button onClick={() => signOut({ callbackUrl: "/login" })} variant="secondary" size="sm">
              Sign out
            </Button>
          </div>
        </header>

        <div className="grid gap-4 md:grid-cols-3">
          {[
            {
              icon: LayoutDashboard,
              label: "Total sessions",
              value: totalSessions,
              detail: "Sessions created from this account",
            },
            {
              icon: CheckCircle2,
              label: "Active rooms",
              value: activeSessions,
              detail: "Ready for audience interaction",
            },
            {
              icon: Clock3,
              label: "Inactive rooms",
              value: inactiveSessions,
              detail: "Closed or parked sessions",
            },
          ].map((stat) => {
            const Icon = stat.icon;
            return (
              <Card key={stat.label} className="bg-card/90">
                <CardHeader className="space-y-4 pb-0">
                  <div className="flex items-center justify-between gap-3">
                    <span className="flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                      <Icon className="size-5" />
                    </span>
                    <Badge variant="outline" className="rounded-full px-2.5 py-1 text-[10px] uppercase tracking-[0.2em]">
                      overview
                    </Badge>
                  </div>
                  <CardTitle className="text-sm font-medium uppercase tracking-[0.2em] text-muted-foreground">
                    {stat.label}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 pt-4">
                  <p className="text-4xl font-semibold tracking-tight">{stat.value}</p>
                  <CardDescription>{stat.detail}</CardDescription>
                </CardContent>
              </Card>
            );
          })}
        </div>

        <section className="flex flex-col gap-4 rounded-[1.75rem] border border-border/70 bg-card/85 p-6 shadow-sm backdrop-blur lg:flex-row lg:items-end lg:justify-between">
          <div className="space-y-2">
            <p className="text-sm font-semibold uppercase tracking-[0.22em] text-muted-foreground">
              sessions
            </p>
            <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
              Your latest rooms live here.
            </h2>
            <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
              Create a new session, copy its join link, and move straight into a live audience experience.
            </p>
          </div>

          <CreateSessionDialog
            token={authSession?.apiToken ?? ""}
            onCreated={fetchSessions}
          />
        </section>

        <section>
          {loading ? (
            <Card className="bg-card/90">
              <CardContent className="py-12 text-center text-sm text-muted-foreground">
                Loading sessions...
              </CardContent>
            </Card>
          ) : sessions.length === 0 ? (
            <Card className="bg-card/90">
              <CardHeader className="space-y-2 text-center">
                <CardTitle className="text-2xl">No sessions yet</CardTitle>
                <CardDescription>
                  Create your first room to start running polls and Q&A.
                </CardDescription>
              </CardHeader>
              <CardContent className="flex justify-center pb-6">
                <div className="flex flex-wrap items-center justify-center gap-3">
                  <CreateSessionDialog token={authSession?.apiToken ?? ""} onCreated={fetchSessions} />
                  <Button asChild variant="outline">
                    <Link href="/session/AAAAAA">
                      See a session view
                      <ArrowRight className="size-4" />
                    </Link>
                  </Button>
                </div>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {sessions.map((session) => (
                <SessionCard key={session.id} session={session} />
              ))}
            </div>
          )}
        </section>
      </PageWrapper>
    </main>
  );
}

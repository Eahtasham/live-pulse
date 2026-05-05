"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useSession } from "next-auth/react";
import {
  ArrowRight,
  BarChart3,
  LayoutDashboard,
  MessageSquareText,
  ShieldCheck,
  Sparkles,
  Users,
  Zap,
} from "lucide-react";
import { Brand } from "@/components/brand";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ThemeToggle } from "@/components/theme-toggle";
import { PageWrapper } from "@/components/page-wrapper";

const featureCards = [
  {
    title: "Live polls that feel immediate",
    description:
      "Launch polls, watch results animate in real time, and keep the audience focused on the moment.",
    icon: BarChart3,
  },
  {
    title: "Q&A built for moderation",
    description:
      "Capture questions and comments in one feed with host controls that stay out of the way until needed.",
    icon: MessageSquareText,
  },
  {
    title: "Host controls without clutter",
    description:
      "Create sessions, manage polls, and share links from a dashboard that feels more like a command center.",
    icon: LayoutDashboard,
  },
  {
    title: "Theme-aware across every surface",
    description:
      "Light, dark, and system themes are part of the product instead of a bolted-on toggle.",
    icon: Sparkles,
  },
];

const workflowSteps = [
  {
    step: "01",
    title: "Create a session",
    description:
      "Hosts spin up a room, share the code, and bring the audience in within seconds.",
  },
  {
    step: "02",
    title: "Run polls and Q&A",
    description:
      "Audience responses update live while the host keeps the room moderated and readable.",
  },
  {
    step: "03",
    title: "Close with clarity",
    description:
      "Wrap with results, archived questions, and a clean record of what happened in the session.",
  },
];

export default function Home() {
  const { data: session } = useSession();
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const router = useRouter();

  function handleJoin(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = code.trim().toUpperCase();
    if (trimmed.length === 6) {
      setError("");
      router.push(`/session/${trimmed}`);
      return;
    }

    setError("Enter a 6-character session code.");
  }

  return (
    <main className="relative isolate overflow-hidden">
      <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        <div className="absolute left-[-10%] top-[-12%] h-96 w-96 rounded-full bg-primary/15 blur-3xl dark:bg-primary/20" />
        <div className="absolute right-[-8%] top-[10%] h-80 w-80 rounded-full bg-cyan-500/10 blur-3xl dark:bg-cyan-400/10" />
        <div className="absolute bottom-[-14%] left-[22%] h-72 w-72 rounded-full bg-emerald-500/10 blur-3xl dark:bg-emerald-400/10" />
      </div>

      <PageWrapper className="flex items-center justify-between gap-4">
        <Brand href="/" />

        <div className="flex items-center gap-2 sm:gap-3">
          <Link
            href="#features"
            className="hidden rounded-full px-3 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground md:inline-flex"
          >
            Features
          </Link>
          <ThemeToggle />
          {session ? (
            <Button asChild size="sm" className="hidden sm:inline-flex">
              <Link href="/dashboard">Go to dashboard</Link>
            </Button>
          ) : (
            <>
              <Button asChild variant="outline" size="sm">
                <Link href="/login">Sign in</Link>
              </Button>
              <Button asChild size="sm" className="hidden sm:inline-flex">
                <Link href="/dashboard">Open dashboard</Link>
              </Button>
            </>
          )}
        </div>
      </PageWrapper>

      <PageWrapper className="grid gap-12 pb-16 pt-6 lg:grid-cols-[1.15fr_0.85fr] lg:items-center lg:pb-20 lg:pt-10">
        <div className="space-y-8">
          <div className="space-y-5">
            <Badge variant="outline" className="w-fit rounded-full border-primary/20 bg-background/80 px-3 py-1 text-primary shadow-sm backdrop-blur">
              live sessions that look and feel premium
            </Badge>
            <div className="space-y-4">
              <h1 className="max-w-3xl text-5xl font-semibold leading-[0.95] tracking-tight text-balance text-foreground sm:text-6xl lg:text-7xl">
                Run polls and Q&A with a front end that keeps up with the room.
              </h1>
              <p className="max-w-2xl text-base leading-7 text-muted-foreground sm:text-lg">
                LivePulse gives hosts a sleek control center and audiences a fast,
                low-friction way to join, vote, and ask questions without losing the
                rhythm of the session.
              </p>
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <Button asChild size="lg" className="gap-2">
              <Link href="#join">
                Join a live session
                <ArrowRight className="size-4" />
              </Link>
            </Button>
            <Button asChild variant="outline" size="lg">
              <Link href="/dashboard">Create and manage rooms</Link>
            </Button>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            {[
              {
                icon: Users,
                label: "Audience friendly",
                value: "No account needed to join",
              },
              {
                icon: Zap,
                label: "Realtime by default",
                value: "Polls and Q&A update instantly",
              },
              {
                icon: ShieldCheck,
                label: "Host control",
                value: "Moderation stays in your hands",
              },
            ].map((item) => {
              const Icon = item.icon;
              return (
                <div
                  key={item.label}
                  className="rounded-2xl border border-border/70 bg-card/80 p-4 shadow-sm backdrop-blur"
                >
                  <div className="flex items-center gap-3">
                    <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                      <Icon className="size-5" />
                    </span>
                    <div className="min-w-0">
                      <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                        {item.label}
                      </p>
                      <p className="mt-1 text-sm font-medium leading-5 text-foreground">
                        {item.value}
                      </p>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        <Card
          id="join"
          className="relative border-border/70 bg-card/90 shadow-[0_28px_70px_-34px_rgba(15,23,42,0.42)]"
        >
          <CardHeader className="space-y-4">
            <div className="flex items-center justify-between gap-3">
              <Badge
                variant="secondary"
                className="rounded-full px-3 py-1 text-xs uppercase tracking-[0.18em]"
              >
                join live
              </Badge>
              <Badge
                variant="outline"
                className="rounded-full border-primary/20 bg-primary/5 px-3 py-1 text-primary"
              >
                6-character code
              </Badge>
            </div>
            <div className="space-y-2">
              <CardTitle className="text-2xl leading-tight">
                Enter a session code and jump straight into the room.
              </CardTitle>
              <CardDescription className="text-base leading-7">
                Audience members join instantly. Hosts can create sessions from the
                dashboard and share them with a single link.
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="space-y-4">
            <form onSubmit={handleJoin} className="space-y-3">
              <div className="space-y-2">
                <Input
                  type="text"
                  value={code}
                  onChange={(e) => {
                    setCode(
                      e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6)
                    );
                    if (error) setError("");
                  }}
                  placeholder="A1B2C3"
                  maxLength={6}
                  aria-invalid={!!error}
                  className="h-14 rounded-2xl border-border/70 bg-background/80 text-center font-mono text-2xl tracking-[0.45em] shadow-sm backdrop-blur placeholder:tracking-[0.25em]"
                />
                {error ? (
                  <p className="text-sm text-destructive">{error}</p>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No login required for audience members.
                  </p>
                )}
              </div>

              <Button type="submit" size="lg" className="h-14 w-full gap-2 rounded-2xl">
                Join session
                <ArrowRight className="size-4" />
              </Button>
            </form>

            <div className="grid gap-2 sm:grid-cols-3">
              {[
                "Realtime voting",
                "Moderated Q&A",
                "Theme-aware UI",
              ].map((item) => (
                <div
                  key={item}
                  className="rounded-2xl border border-border/70 bg-muted/50 px-3 py-2 text-center text-sm font-medium text-muted-foreground"
                >
                  {item}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </PageWrapper>

      <PageWrapper className="pb-20">
        <div className="mb-6 flex items-end justify-between gap-4">
          <div className="space-y-2">
            <p className="text-sm font-semibold uppercase tracking-[0.22em] text-muted-foreground">
              Product features
            </p>
            <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">
              Built for live moments that need momentum.
            </h2>
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {featureCards.map((feature) => {
            const Icon = feature.icon;
            return (
              <Card key={feature.title} className="h-full bg-card/90">
                <CardHeader className="space-y-4">
                  <div className="flex items-start justify-between gap-3">
                    <span className="flex size-12 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                      <Icon className="size-6" />
                    </span>
                    <Badge variant="outline" className="rounded-full px-2.5 py-1 text-[10px] uppercase tracking-[0.2em]">
                      live
                    </Badge>
                  </div>
                  <CardTitle className="text-xl leading-tight">
                    {feature.title}
                  </CardTitle>
                  <CardDescription className="text-sm leading-6">
                    {feature.description}
                  </CardDescription>
                </CardHeader>
              </Card>
            );
          })}
        </div>
      </PageWrapper>

      <PageWrapper className="pb-24">
        <div className="grid gap-4 lg:grid-cols-3">
          {workflowSteps.map((step) => (
            <Card key={step.step} className="h-full bg-card/90">
              <CardHeader className="space-y-4">
                <div className="flex items-center justify-between gap-3">
                  <Badge variant="secondary" className="rounded-full px-3 py-1 text-xs tracking-[0.2em]">
                    {step.step}
                  </Badge>
                  <Sparkles className="size-4 text-primary" />
                </div>
                <CardTitle className="text-xl leading-tight">
                  {step.title}
                </CardTitle>
                <CardDescription className="text-sm leading-6">
                  {step.description}
                </CardDescription>
              </CardHeader>
            </Card>
          ))}
        </div>

        <div className="mt-6 flex flex-col items-start justify-between gap-4 rounded-[1.75rem] border border-border/70 bg-card/80 px-6 py-6 shadow-sm backdrop-blur md:flex-row md:items-center">
          <div className="space-y-2">
            <p className="text-sm font-semibold uppercase tracking-[0.22em] text-muted-foreground">
              ready to host
            </p>
            <p className="text-lg font-medium text-foreground">
              Create a session, open polls, and keep the audience engaged from a single flow.
            </p>
          </div>
          <div className="flex gap-3">
            {session ? (
              <Button asChild size="lg">
                <Link href="/dashboard">Go to dashboard</Link>
              </Button>
            ) : (
              <>
                <Button asChild variant="outline" size="lg">
                  <Link href="/login">Sign in</Link>
                </Button>
                <Button asChild size="lg">
                  <Link href="/dashboard">Go to dashboard</Link>
                </Button>
              </>
            )}
          </div>
        </div>
      </PageWrapper>
    </main>
  );
}

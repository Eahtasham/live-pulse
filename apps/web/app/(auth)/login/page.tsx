"use client";

import { signIn } from "next-auth/react";
import { useSession } from "next-auth/react";
import { isTemp } from "tempmail-checker";
import { useEffect, useState } from "react";
import {
  ArrowRight,
  CheckCircle2,
  Globe2,
  LockKeyhole,
  MessageSquareText,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { Brand } from "../../../components/brand";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ThemeToggle } from "@/components/theme-toggle";
import { PageWrapper } from "@/components/page-wrapper";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function LoginPage() {
  const { status } = useSession();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const router = useRouter();

  const isRegisterMode = mode === "register";

  useEffect(() => {
    if (status === "authenticated") {
      router.replace("/dashboard");
    }
  }, [router, status]);

  if (status === "authenticated") {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-6">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <CardTitle className="text-xl">Redirecting to dashboard</CardTitle>
            <CardDescription>
              You&apos;re already signed in.
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    );
  }

  async function handleSubmit(e: React.SyntheticEvent, modeOverride?: "login" | "register") {
    e.preventDefault();
    setLoading(true);
    setError("");

    const currentMode = modeOverride || mode;

    if (isTemp(email)) {
      setError("Disposable email addresses are not allowed");
      setLoading(false);
      return;
    }

    if (currentMode === "register") {
      // Register via Go API, then sign in via NextAuth
      try {
        const res = await fetch(`${apiUrl}/v1/auth/register`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, name, password }),
        });

        if (!res.ok) {
          const data = await res.json();
          setError(data.message || "Registration failed");
          setLoading(false);
          return;
        }

        // Registration succeeded — now log in via NextAuth
        await signIn("email-login", {
          email,
          password,
          callbackUrl: "/dashboard",
        });
      } catch {
        setError("Something went wrong");
        setLoading(false);
      }
    } else {
      // Login via NextAuth credentials provider
      const result = await signIn("email-login", {
        email,
        password,
        redirect: false,
      });

      if (result?.error) {
        setError("Invalid email or password");
        setLoading(false);
      } else {
        window.location.href = "/dashboard";
      }
    }
  }

  return (
    <main className="relative isolate min-h-screen overflow-hidden px-6 py-6 lg:px-8">
      <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        <div className="absolute left-[-12%] top-[-10%] h-80 w-80 rounded-full bg-primary/15 blur-3xl dark:bg-primary/20" />
        <div className="absolute right-[-10%] top-[12%] h-72 w-72 rounded-full bg-cyan-500/10 blur-3xl dark:bg-cyan-400/10" />
      </div>

      <PageWrapper className="flex items-center justify-between gap-4">
        <Brand href="/" />

        <div className="flex items-center gap-2">
          <ThemeToggle />
        </div>
      </PageWrapper>

      <PageWrapper className="grid gap-10 py-12 lg:grid-cols-[1.15fr_0.85fr] lg:items-center lg:py-16">
        <section className="space-y-8">
          <div className="space-y-5">
            <Badge variant="outline" className="w-fit rounded-full border-primary/20 bg-background/80 px-3 py-1 text-primary shadow-sm backdrop-blur">
              secure host access
            </Badge>
            <div className="space-y-4">
              <h1 className="max-w-2xl text-5xl font-semibold leading-[0.95] tracking-tight text-balance sm:text-6xl lg:text-7xl">
                {isRegisterMode
                  ? "Create your host account and get your first room live."
                  : "Sign in and open the control room for your next session."}
              </h1>
              <p className="max-w-xl text-base leading-7 text-muted-foreground sm:text-lg">
                {isRegisterMode
                  ? "Use your email to create a host account, then jump straight into the dashboard to create sessions and moderate live interaction."
                  : "Use Google or your email credentials to access the dashboard and session controls."}
              </p>
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            {[
              {
                icon: LockKeyhole,
                title: "Protected login",
                text: "Google or email with server-side auth",
              },
              {
                icon: MessageSquareText,
                title: "Fast moderation",
                text: "Keep the room clear and responsive",
              },
              {
                icon: Globe2,
                title: "System theme",
                text: "Matches the rest of the app instantly",
              },
            ].map((item) => {
              const Icon = item.icon;
              return (
                <div
                  key={item.title}
                  className="rounded-2xl border border-border/70 bg-card/80 p-4 shadow-sm backdrop-blur"
                >
                  <div className="flex items-center gap-3">
                    <span className="flex size-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                      <Icon className="size-5" />
                    </span>
                    <div>
                      <p className="text-sm font-medium text-foreground">
                        {item.title}
                      </p>
                      <p className="text-sm text-muted-foreground">{item.text}</p>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </section>

        <Card className="border-border/70 bg-card/90 shadow-[0_28px_70px_-34px_rgba(15,23,42,0.42)]">
          <CardHeader className="space-y-4 pb-0">
            <div className="flex items-center justify-between gap-3">
              <Badge variant="secondary" className="rounded-full px-3 py-1 text-xs uppercase tracking-[0.18em]">
                {isRegisterMode ? "create account" : "sign in"}
              </Badge>
              <Badge variant="outline" className="rounded-full border-primary/20 bg-primary/5 px-3 py-1 text-primary">
                1 minute setup
              </Badge>
            </div>
            <div className="space-y-2">
              <CardTitle className="text-2xl leading-tight">
                {isRegisterMode
                  ? "Start hosting in a couple of steps."
                  : "Welcome back, host."}
              </CardTitle>
              <CardDescription className="text-base leading-7">
                {isRegisterMode
                  ? "Create a host profile, connect your sign-in method, and you will land on the dashboard immediately."
                  : "Use your email credentials to access the dashboard and session controls."}
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="space-y-6 pt-6">
            <Button
              type="button"
              onClick={() => signIn("google", { callbackUrl: "/dashboard" })}
              variant="outline"
              className="h-12 w-full justify-center gap-3 rounded-2xl border-border/70 bg-background/80 text-sm font-medium shadow-sm backdrop-blur"
            >
              <svg className="size-5" viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
                  fill="#4285F4"
                />
                <path
                  d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                  fill="#34A853"
                />
                <path
                  d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                  fill="#FBBC05"
                />
                <path
                  d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                  fill="#EA4335"
                />
              </svg>
              Continue with Google
            </Button>

            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-border/70" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-card px-2 text-muted-foreground">or</span>
              </div>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="rounded-2xl border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                  {error}
                </div>
              )}

              <div className="space-y-2">
                <div className="flex items-center justify-between gap-3">
                  <Label htmlFor="email">Email</Label>
                  <span className="text-xs text-muted-foreground">Required</span>
                </div>
                <Input
                  id="email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="leomessi@goat.com"
                  className="h-12 rounded-2xl border-border/70 bg-background/80 px-4 shadow-sm backdrop-blur"
                />
              </div>

              {isRegisterMode && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between gap-3">
                    <Label htmlFor="name">Name</Label>
                    <span className="text-xs text-muted-foreground">Displayed to hosts</span>
                  </div>
                  <Input
                    id="name"
                    type="text"
                    required
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Lionel Messi"
                    className="h-12 rounded-2xl border-border/70 bg-background/80 px-4 shadow-sm backdrop-blur"
                  />
                </div>
              )}

              <div className="space-y-2">
                <div className="flex items-center justify-between gap-3">
                  <Label htmlFor="password">Password</Label>
                  <span className="text-xs text-muted-foreground">At least 8 characters</span>
                </div>
                <Input
                  id="password"
                  type="password"
                  required
                  minLength={8}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="h-12 rounded-2xl border-border/70 bg-background/80 px-4 shadow-sm backdrop-blur"
                />
              </div>

              <div className="flex items-center justify-between gap-3 rounded-2xl border border-border/70 bg-muted/40 p-1">
                <Button
                  type="button"
                  variant={mode === "login" ? "default" : "ghost"}
                  className="h-10 flex-1 rounded-xl"
                  onClick={(e) => {
                    e.preventDefault();
                    setMode("login");
                    setError("");
                    handleSubmit(e, "login");
                  }}
                  disabled={loading}
                >
                  {loading && mode === "login" ? "Signing in..." : "Sign in"}
                </Button>
                <Button
                  type="button"
                  variant={isRegisterMode ? "default" : "ghost"}
                  className="h-10 flex-1 rounded-xl"
                  onClick={(e) => {
                    e.preventDefault();
                    setMode("register");
                    setError("");
                    handleSubmit(e, "register");
                  }}
                  disabled={loading}
                >
                  {loading && isRegisterMode ? "Creating account..." : "Sign up"}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </PageWrapper>
    </main>
  );
}

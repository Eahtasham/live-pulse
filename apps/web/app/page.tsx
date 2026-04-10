"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ThemeToggle } from "@/components/theme-toggle";

export default function Home() {
  const [code, setCode] = useState("");
  const router = useRouter();

  function handleJoin(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = code.trim().toUpperCase();
    if (trimmed.length === 6) {
      router.push(`/session/${trimmed}`);
    }
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background p-4">
      <div className="absolute right-4 top-4">
        <ThemeToggle />
      </div>

      <div className="flex w-full max-w-md flex-col items-center gap-8">
        <div className="text-center">
          <h1 className="text-4xl font-bold tracking-tight">LivePulse</h1>
          <p className="mt-2 text-lg text-muted-foreground">
            Real-time audience engagement
          </p>
        </div>

        <form onSubmit={handleJoin} className="flex w-full gap-2">
          <input
            type="text"
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6))}
            placeholder="Enter session code"
            maxLength={6}
            className="flex-1 rounded-lg border border-input bg-background px-4 py-3 text-center font-mono text-lg font-semibold tracking-widest placeholder:font-sans placeholder:text-sm placeholder:font-normal placeholder:tracking-normal focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/50"
          />
          <button
            type="submit"
            disabled={code.trim().length !== 6}
            className="rounded-lg bg-primary px-6 py-3 font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            Join
          </button>
        </form>

        <div className="relative w-full">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-border" />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-muted-foreground">or</span>
          </div>
        </div>

        <div className="flex gap-3">
          <Link
            href="/login"
            className="rounded-lg border border-border px-6 py-2.5 text-sm font-medium transition-colors hover:bg-muted"
          >
            Sign in
          </Link>
          <Link
            href="/dashboard"
            className="rounded-lg bg-secondary px-6 py-2.5 text-sm font-medium text-secondary-foreground transition-colors hover:bg-secondary/80"
          >
            Host Dashboard
          </Link>
        </div>
      </div>
    </div>
  );
}

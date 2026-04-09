"use client";

import { useSession, signOut } from "next-auth/react";
import { ThemeToggle } from "@/components/theme-toggle";

export default function DashboardPage() {
  const { data: session, status } = useSession();

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
          {session?.user && (
            <span className="text-sm text-zinc-600 dark:text-zinc-400">
              {session.user.email}
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
    </div>
  );
}

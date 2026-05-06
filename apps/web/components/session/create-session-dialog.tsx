"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Copy, Check, Plus } from "lucide-react";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface Props {
  token: string;
  onCreated: () => void;
}

interface CreatedSession {
  id: string;
  code: string;
  title: string;
  status: string;
}

export function CreateSessionDialog({ token, onCreated }: Props) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [created, setCreated] = useState<CreatedSession | null>(null);
  const [copied, setCopied] = useState(false);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const res = await fetch(`${apiUrl}/v1/sessions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ title: title.trim() }),
      });

      if (!res.ok) {
        setError("Failed to create session");
        setLoading(false);
        return;
      }

      const data = await res.json();
      setCreated(data);
      onCreated();
    } catch {
      setError("Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  function handleClose() {
    setOpen(false);
    setTitle("");
    setCreated(null);
    setCopied(false);
    setError("");
  }

  function handleCopyLink() {
    if (!created) return;
    const link = `${window.location.origin}/session/${created.code}`;
    navigator.clipboard.writeText(link).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) handleClose();
        else setOpen(true);
      }}
    >
      <DialogTrigger asChild>
        <Button>
          <Plus className="w-4 h-4 mr-2" />
          New Session
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {created ? "Session Created!" : "Create a new session"}
          </DialogTitle>
        </DialogHeader>

        {created ? (
          <div className="space-y-6 py-2">
            <div className="flex flex-col items-center justify-center space-y-3 rounded-xl border border-border/50 bg-muted/30 p-6">
              <p className="text-sm font-medium text-muted-foreground">Session Code</p>
              <p className="font-mono text-4xl font-bold tracking-widest">
                {created.code}
              </p>
            </div>

            <div className="space-y-3">
              <Label htmlFor="session-link">Share link</Label>
              <div className="flex gap-2">
                <div className="flex-1 rounded-md border border-input bg-muted/50 px-3 py-2 text-sm">
                  <span className="truncate block">
                    {typeof window !== "undefined"
                      ? `${window.location.origin}/session/${created.code}`
                      : `/session/${created.code}`}
                  </span>
                </div>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={handleCopyLink}
                  className="shrink-0"
                >
                  {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                </Button>
              </div>
            </div>

            <div className="flex gap-2 pt-2">
              <Button variant="outline" onClick={handleClose} className="flex-1">
                Close
              </Button>
              <Button
                onClick={() => router.push(`/session/${created.code}`)}
                className="flex-1"
              >
                Go to Session
              </Button>
            </div>
          </div>
        ) : (
          <form onSubmit={handleCreate} className="space-y-6 py-2">
            {error && (
              <div className="rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                {error}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="session-title">Session title</Label>
              <Input
                id="session-title"
                placeholder="e.g. CS101 Lecture 5"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
                autoFocus
                className="h-11"
              />
            </div>
            <div className="flex gap-2 pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={handleClose}
                className="flex-1"
              >
                Cancel
              </Button>
              <Button type="submit" disabled={loading || !title.trim()} className="flex-1">
                {loading ? "Creating..." : "Create"}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
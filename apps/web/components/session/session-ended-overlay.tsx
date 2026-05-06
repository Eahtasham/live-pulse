import Link from "next/link";
import { Clock3 } from "lucide-react";

import { ShareSessionButton } from "@/components/session/share-modal";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

interface SessionEndedOverlayProps {
  sessionCode: string;
  sessionTitle: string;
  closedAt?: string | null;
}

function relativeTime(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const minutes = Math.floor(diff / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function SessionEndedOverlay({ sessionCode, sessionTitle, closedAt }: SessionEndedOverlayProps) {
  return (
    <div className="absolute inset-0 z-20 flex items-center justify-center bg-background/80 px-4 py-8 backdrop-blur-sm">
      <Card className="w-full max-w-2xl border-destructive/20 bg-card/95 shadow-2xl">
        <CardHeader className="space-y-4 text-center">
          <Badge variant="destructive" className="mx-auto w-fit rounded-full px-3 py-1 text-xs uppercase tracking-[0.2em]">
            Session ended
          </Badge>
          <CardTitle className="text-3xl tracking-tight">{sessionTitle} is closed</CardTitle>
          <CardDescription className="mx-auto max-w-xl text-base leading-7">
            {closedAt ? (
              <span className="inline-flex items-center gap-2">
                <Clock3 className="size-4" />
                Closed {relativeTime(closedAt)}
              </span>
            ) : (
              "The host has closed this session. Polls and Q&A are now read only."
            )}
          </CardDescription>
        </CardHeader>

        <CardContent className="flex flex-col gap-3 sm:flex-row sm:justify-center">
          <ShareSessionButton sessionCode={sessionCode} sessionTitle={sessionTitle} />
          <Button asChild variant="outline" className="h-11 rounded-2xl">
            <Link href="/dashboard">Back to dashboard</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
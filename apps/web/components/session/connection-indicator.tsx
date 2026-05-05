import { Loader2, Wifi, WifiOff } from "lucide-react";

import type { ConnectionState } from "@/hooks/use-websocket";
import { cn } from "@/lib/utils";

interface ConnectionIndicatorProps {
  state: ConnectionState;
  sessionEnded?: boolean;
  className?: string;
}

export function ConnectionIndicator({ state, sessionEnded = false, className }: ConnectionIndicatorProps) {
  if (sessionEnded) {
    return (
      <span
        className={cn(
          "inline-flex items-center gap-2 rounded-full border border-destructive/20 bg-destructive/10 px-3 py-1 text-xs font-medium text-destructive",
          className
        )}
      >
        <span className="h-2 w-2 rounded-full bg-destructive" />
        Session ended
      </span>
    );
  }

  const copy =
    state === "connected"
      ? {
          label: "Live",
          icon: <Wifi className="size-3.5" />,
          className: "border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
          dot: "bg-emerald-500",
        }
      : state === "connecting"
        ? {
            label: "Connecting",
            icon: <Loader2 className="size-3.5 animate-spin" />,
            className: "border-amber-500/20 bg-amber-500/10 text-amber-700 dark:text-amber-300",
            dot: "bg-amber-500",
          }
        : {
            label: "Reconnecting",
            icon: <WifiOff className="size-3.5" />,
            className: "border-muted-foreground/20 bg-muted/70 text-muted-foreground",
            dot: "bg-muted-foreground",
          };

  return (
    <span
      className={cn(
        "inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium",
        copy.className,
        className
      )}
      aria-live="polite"
    >
      <span className={cn("h-2 w-2 rounded-full", copy.dot)} />
      {copy.icon}
      {copy.label}
    </span>
  );
}
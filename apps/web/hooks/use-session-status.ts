"use client";

import { useEffect, useState } from "react";
import type { useWebSocket } from "@/hooks/use-websocket";

interface SessionClosedPayload {
  code: string;
  closed_at: string;
}

export function useSessionStatus(
  ws: ReturnType<typeof useWebSocket>
) {
  const [sessionEnded, setSessionEnded] = useState(false);
  const [closedAt, setClosedAt] = useState<string | null>(null);

  useEffect(() => {
    return ws.subscribe((msg) => {
      if (msg.type === "session_closed") {
        const payload = msg.payload as SessionClosedPayload;
        setSessionEnded(true);
        setClosedAt(payload.closed_at);
      }
    });
  }, [ws]);

  return { sessionEnded, closedAt };
}

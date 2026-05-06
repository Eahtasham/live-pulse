"use client";

import { useEffect, useState, useRef } from "react";
import { Badge } from "@/components/ui/badge";

interface Props {
  timeLimitSec: number;
  isActive: boolean;
}

export function PollTimerBadge({ timeLimitSec, isActive }: Props) {
  const [remaining, setRemaining] = useState(timeLimitSec);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    const frame = requestAnimationFrame(() => setRemaining(timeLimitSec));

    if (isActive) {
      intervalRef.current = setInterval(() => {
        setRemaining((prev) => {
          if (prev <= 1) {
            if (intervalRef.current) clearInterval(intervalRef.current);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    }

    return () => {
      cancelAnimationFrame(frame);
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [timeLimitSec, isActive]);

  if (!isActive || remaining <= 0) return null;

  const minutes = Math.floor(remaining / 60);
  const seconds = remaining % 60;
  const timeStr = minutes > 0 ? `${minutes}:${seconds.toString().padStart(2, "0")}` : `${seconds}s`;

  return (
    <Badge
      variant={remaining <= 10 ? "destructive" : "outline"}
      className="font-mono"
    >
      {timeStr}
    </Badge>
  );
}

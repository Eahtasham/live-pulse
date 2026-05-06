import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";

interface SpinnerProps {
  className?: string;
  label?: string;
}

export function Spinner({ className, label }: SpinnerProps) {
  return (
    <span className="inline-flex items-center gap-2 text-sm text-muted-foreground">
      <Loader2 className={cn("size-4 animate-spin", className)} aria-hidden="true" />
      {label ? <span>{label}</span> : null}
    </span>
  );
}
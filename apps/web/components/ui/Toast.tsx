import type { ReactNode } from "react";
import { AlertCircle, CheckCircle2, Info, XCircle } from "lucide-react";

import { cn } from "@/lib/utils";

type ToastVariant = "default" | "success" | "warning" | "error";

interface ToastProps {
  title?: string;
  description: string;
  variant?: ToastVariant;
  className?: string;
  action?: ReactNode;
}

const variantStyles: Record<ToastVariant, { wrapper: string; icon: ReactNode }> = {
  default: {
    wrapper: "border-border/70 bg-card/90 text-foreground",
    icon: <Info className="size-4" />,
  },
  success: {
    wrapper: "border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
    icon: <CheckCircle2 className="size-4" />,
  },
  warning: {
    wrapper: "border-amber-500/20 bg-amber-500/10 text-amber-700 dark:text-amber-300",
    icon: <AlertCircle className="size-4" />,
  },
  error: {
    wrapper: "border-destructive/20 bg-destructive/10 text-destructive",
    icon: <XCircle className="size-4" />,
  },
};

export function Toast({ title, description, variant = "default", className, action }: ToastProps) {
  const styles = variantStyles[variant];

  return (
    <div
      aria-live="polite"
      className={cn(
        "flex items-start gap-3 rounded-2xl border px-4 py-3 shadow-sm",
        styles.wrapper,
        className
      )}
    >
      <span className="mt-0.5 shrink-0">{styles.icon}</span>
      <div className="min-w-0 flex-1 space-y-0.5">
        {title ? <p className="text-sm font-semibold leading-5">{title}</p> : null}
        <p className="text-sm leading-6 opacity-90">{description}</p>
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </div>
  );
}
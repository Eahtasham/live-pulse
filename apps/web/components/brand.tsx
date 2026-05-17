import Link from "next/link";
import { Zap } from "lucide-react";

import { cn } from "@/lib/utils";

interface BrandProps {
  href?: string;
  className?: string;
  size?: "default" | "sm";
}

export function Brand({ href, className, size = "default" }: BrandProps) {
  const rootClassName = cn(
    "group flex items-center gap-2.5",
    size === "sm" && "gap-2",
    className
  );

  const markClassName = cn(
    "flex items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm transition-transform duration-150 group-hover:scale-105",
    size === "sm" ? "size-8" : "size-9"
  );

  const wordmark = (
    <>
      <span className={markClassName}>
        <Zap className={size === "sm" ? "size-4" : "size-4.5"} />
      </span>
      <div>
        <p
          className={cn(
            "font-semibold tracking-tight text-foreground",
            size === "sm" ? "text-sm" : "text-base"
          )}
        >
          LivePulse
        </p>
        {size !== "sm" && (
          <p className="text-xs text-muted-foreground">
            real-time polls &amp; Q&amp;A
          </p>
        )}
      </div>
    </>
  );

  if (!href) {
    return <div className={rootClassName}>{wordmark}</div>;
  }

  return (
    <Link href={href} className={rootClassName}>
      {wordmark}
    </Link>
  );
}
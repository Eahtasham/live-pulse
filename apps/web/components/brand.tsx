import Link from "next/link";
import { Rocket } from "lucide-react";

import { cn } from "@/lib/utils";

interface BrandProps {
  href?: string;
  className?: string;
  size?: "default" | "sm";
}

export function Brand({ href, className, size = "default" }: BrandProps) {
  const rootClassName = cn(
    "group flex items-center gap-3",
    size === "sm" && "gap-2",
    className
  );

  const markClassName = cn(
    "flex items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-[0_18px_36px_-20px_rgba(16,185,129,0.7)] transition-transform duration-200 group-hover:-translate-y-0.5",
    size === "sm" ? "size-10" : "size-11"
  );

  const wordmark = (
    <>
      <span className={markClassName}>
        <Rocket className={size === "sm" ? "size-4" : "size-5"} />
      </span>
      <div>
        <p
          className={cn(
            "font-semibold tracking-[0.18em] uppercase text-muted-foreground",
            size === "sm" ? "text-xs" : "text-sm"
          )}
        >
          LivePulse
        </p>
        <p
          className={cn(
            "text-muted-foreground",
            size === "sm" ? "text-xs" : "text-sm"
          )}
        >
          real-time polls and q&a
        </p>
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
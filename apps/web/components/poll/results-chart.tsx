"use client";

import type { PollOption } from "@/lib/poll";

interface Props {
  options: PollOption[];
  highlightedIds?: string[];
}

export function ResultsChart({ options, highlightedIds }: Props) {
  const totalVotes = options.reduce((sum, o) => sum + o.vote_count, 0);
  const maxVotes = Math.max(...options.map((o) => o.vote_count), 1);

  return (
    <div className="space-y-2">
      {options.map((option) => {
        const percentage =
          totalVotes > 0 ? (option.vote_count / totalVotes) * 100 : 0;
        const barWidth =
          maxVotes > 0 ? (option.vote_count / maxVotes) * 100 : 0;
        const isHighest =
          totalVotes > 0 && option.vote_count === maxVotes;
        const isHighlighted = highlightedIds?.includes(option.id);

        return (
          <div key={option.id} className="space-y-1">
            <div className="flex items-center justify-between text-sm">
              <span
                className={`font-medium ${
                  isHighlighted ? "text-primary" : ""
                }`}
              >
                {option.label}
                {isHighlighted && (
                  <span className="ml-1 text-xs text-primary">✓</span>
                )}
              </span>
              <span className="text-muted-foreground">
                {option.vote_count} ({percentage.toFixed(0)}%)
              </span>
            </div>
            <div className="h-6 w-full overflow-hidden rounded-md bg-muted">
              <div
                className={`h-full rounded-md transition-all duration-500 ${
                  isHighest && totalVotes > 0
                    ? "bg-primary"
                    : "bg-primary/50"
                }`}
                style={{ width: `${barWidth}%` }}
              />
            </div>
          </div>
        );
      })}
      <p className="text-xs text-muted-foreground">
        {totalVotes} total vote{totalVotes !== 1 ? "s" : ""}
      </p>
    </div>
  );
}

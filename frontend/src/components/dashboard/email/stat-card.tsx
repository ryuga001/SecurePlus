import type { LucideIcon } from "lucide-react";

import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export function StatCard({
  label,
  value,
  hint,
  icon: Icon,
  tone = "default",
  hero = false,
}: {
  label: string;
  value: string;
  hint: string;
  icon: LucideIcon;
  tone?: "default" | "blocked" | "flagged";
  hero?: boolean;
}) {
  return (
    <div className="flex flex-col justify-between rounded-lg border bg-surface p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <span className="text-xs font-medium tracking-wide text-text-tertiary uppercase">
            {label}
          </span>

          <p
            className={cn(
              "mt-2 font-mono font-semibold tracking-tight tabular-nums",
              hero ? "text-[40px] leading-none" : "text-[28px] leading-8",
            )}
          >
            {value}
          </p>
        </div>

        <span
          aria-hidden
          className={cn(
            "flex size-9 shrink-0 items-center justify-center rounded-lg bg-surface-container",
            tone === "blocked" && "text-chart-1",
            tone === "flagged" && "text-chart-2",
            tone === "default" && "text-text-tertiary",
          )}
        >
          <Icon className="size-5" />
        </span>
      </div>

      <span className="mt-4 block text-xs text-muted-foreground">{hint}</span>
    </div>
  );
}

export function StatCardSkeleton({ hero = false }: { hero?: boolean }) {
  return (
    <div className="flex flex-col justify-between rounded-lg border bg-surface p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-3 w-24" />
          <Skeleton className={hero ? "h-10 w-20" : "h-8 w-16"} />
        </div>
        <Skeleton className="size-9 rounded-lg" />
      </div>
      <Skeleton className="mt-4 h-3 w-36" />
    </div>
  );
}

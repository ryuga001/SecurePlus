import * as React from "react";

import { cn } from "@/lib/utils";

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      aria-hidden
      className={cn(
        "relative isolate overflow-hidden rounded-md bg-surface-container-high/70",
        "before:absolute before:inset-0 before:-translate-x-full before:bg-linear-to-r before:from-transparent before:via-white/50 before:to-transparent dark:before:via-white/[0.06]",
        "motion-safe:before:animate-[skeleton-shimmer_1.6s_ease-in-out_infinite] motion-reduce:before:hidden",
        className
      )}
      {...props}
    />
  );
}

export { Skeleton };

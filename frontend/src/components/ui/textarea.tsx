"use client";

import * as React from "react";

import { cn } from "@/lib/utils";

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "min-h-24 w-full min-w-0 rounded-md border border-input bg-background px-3 py-3 text-sm text-foreground transition-colors shadow-none outline-none placeholder:text-text-disabled focus-visible:border-primary focus-visible:shadow-[0_0_0_2px_var(--primary-container)] disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-muted disabled:opacity-50 aria-invalid:border-error",
        className
      )}
      {...props}
    />
  );
}

export { Textarea };
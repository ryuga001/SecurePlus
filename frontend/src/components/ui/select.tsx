"use client";

import * as React from "react";

import { cn } from "@/lib/utils";

export type SelectOption = { label: string; value: string };

function Select({
  className,
  options,
  placeholder,
  children,
  ...props
}: React.ComponentProps<"select"> & { options?: SelectOption[]; placeholder?: string }) {
  return (
    <select
      data-slot="select"
      className={cn(
        "h-10 w-full min-w-0 rounded-md border border-input bg-background px-3 text-sm text-foreground transition-colors shadow-none outline-none focus-visible:border-primary focus-visible:shadow-[0_0_0_2px_var(--primary-container)] disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-muted disabled:opacity-50 aria-invalid:border-error",
        className
      )}
      {...props}
    >
      {placeholder !== undefined ? <option value="">{placeholder}</option> : null}
      {options?.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
      {children}
    </select>
  );
}

export { Select };
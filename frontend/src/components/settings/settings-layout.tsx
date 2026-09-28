"use client";

import * as React from "react";

import { Label } from "@/components/ui/label";

export function SettingsRow({
  id,
  label,
  hint,
  error,
  children,
}: {
  id: string;
  label: string;
  hint?: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="grid gap-2 md:grid-cols-[minmax(0,220px)_1fr] md:items-start md:gap-8">
      <div className="flex flex-col gap-1">
        <Label htmlFor={id}>{label}</Label>
        {hint ? (
          <p id={`${id}-hint`} className="text-xs text-muted-foreground">
            {hint}
          </p>
        ) : null}
      </div>

      <div className="flex flex-col gap-1.5 md:max-w-xl">
        {children}
        {error ? (
          <p id={`${id}-error`} role="alert" className="text-xs text-error-text">
            {error}
          </p>
        ) : null}
      </div>
    </div>
  );
}

export function SettingsActionBar({
  dirty,
  unsavedLabel,
  children,
}: {
  dirty: boolean;
  unsavedLabel: string;
  children: React.ReactNode;
}) {
  return (
    <div
      data-slot="settings-action-bar"
      className="sticky bottom-0 -mx-6 -mb-6 flex min-h-16 flex-wrap items-center justify-between gap-3 border-t border-border bg-surface px-6 py-4 shadow-[0_-2px_8px_rgba(15,23,42,0.08)]"
    >
      <span aria-live="polite" className="flex min-h-5 items-center gap-2 text-sm text-foreground">
        {dirty ? (
          <>
            <span className="size-2 rounded-full bg-amber-500" />
            {unsavedLabel}
          </>
        ) : null}
      </span>

      <span className="flex items-center gap-2">{children}</span>
    </div>
  );
}

export function useUnsavedChangesWarning(active: boolean) {
  React.useEffect(() => {
    if (!active) return;

    const warn = (event: BeforeUnloadEvent) => event.preventDefault();

    window.addEventListener("beforeunload", warn);

    return () => window.removeEventListener("beforeunload", warn);
  }, [active]);
}

"use client";

import { Plus, X } from "lucide-react";
import * as React from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

export function TagInput({
  values,
  onChange,
  placeholder,
  addLabel = "Add",
  emptyMessage = "Nothing added yet",
  disabled,
  className,
}: {
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
  addLabel?: string;
  emptyMessage?: string;
  disabled?: boolean;
  className?: string;
}) {
  const [draft, setDraft] = React.useState("");

  function add() {
    const value = draft.trim().toLowerCase();
    if (!value) return;

    if (!values.includes(value)) onChange([...values, value]);
    setDraft("");
  }

  function remove(value: string) {
    onChange(values.filter((current) => current !== value));
  }

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div className="flex gap-2">
        <Input
          value={draft}
          placeholder={placeholder}
          disabled={disabled}
          spellCheck={false}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === ",") {
              event.preventDefault();
              add();
            }
          }}
        />
        <Button type="button" variant="secondary" className="shrink-0" disabled={disabled} onClick={add}>
          <Plus />
          {addLabel}
        </Button>
      </div>

      {values.length === 0 ? (
        <p className="text-xs text-muted-foreground">{emptyMessage}</p>
      ) : (
        <div className="flex flex-wrap gap-1.5">
          {values.map((value) => (
            <span
              key={value}
              className="inline-flex h-5.5 items-center gap-1.5 rounded-md border border-border bg-surface-container py-1 pr-1 pl-2.5 font-mono text-xs"
            >
              {value}
              <button
                type="button"
                aria-label={`Remove ${value}`}
                disabled={disabled}
                onClick={() => remove(value)}
                className="text-text-secondary transition-colors hover:text-destructive"
              >
                <X className="size-3.5" />
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

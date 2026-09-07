"use client";

import { Search } from "lucide-react";
import * as React from "react";

import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

export type CheckboxOption = {
  id: number;
  label: string;
  hint?: string;
};

export function CheckboxList({
  options,
  selected,
  onToggle,
  emptyMessage = "Nothing to choose from yet",
  searchPlaceholder,
  className,
  disabled,
}: {
  options: CheckboxOption[];
  selected: number[];
  onToggle: (id: number, checked: boolean) => void;
  emptyMessage?: string;
  searchPlaceholder?: string;
  className?: string;
  disabled?: boolean;
}) {
  const [search, setSearch] = React.useState("");

  const term = search.trim().toLowerCase();
  const visible = term
    ? options.filter((option) => option.label.toLowerCase().includes(term))
    : options;

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      {searchPlaceholder ? (
        <div className="relative">
          <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="h-9 pl-9"
            placeholder={searchPlaceholder}
            value={search}
            disabled={disabled}
            onChange={(event) => setSearch(event.target.value)}
          />
        </div>
      ) : null}

      <div className="max-h-44 overflow-y-auto border">
        {visible.length === 0 ? (
          <p className="px-3 py-6 text-center text-sm text-muted-foreground">
            {options.length === 0 ? emptyMessage : "No matches"}
          </p>
        ) : (
          visible.map((option) => {
            const checked = selected.includes(option.id);

            return (
              <label
                key={option.id}
                className={cn(
                  "flex cursor-pointer items-center gap-2.5 border-b px-3 py-2.5 text-sm last:border-b-0",
                  checked ? "bg-primary/8" : "hover:bg-muted/60"
                )}
              >
                <input
                  type="checkbox"
                  className="size-4 accent-[var(--primary)]"
                  checked={checked}
                  disabled={disabled}
                  onChange={(event) => onToggle(option.id, event.target.checked)}
                />
                <span className="min-w-0 flex-1 truncate">{option.label}</span>
                {option.hint ? (
                  <span className="shrink-0 text-xs text-muted-foreground">{option.hint}</span>
                ) : null}
              </label>
            );
          })
        )}
      </div>

      <p className="text-xs text-muted-foreground">{selected.length} selected</p>
    </div>
  );
}

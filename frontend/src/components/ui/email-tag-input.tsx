"use client";

import { Plus, X } from "lucide-react";
import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

const EMAIL_PATTERN = /^[^\s@,;<>]+@[^\s@,;<>]+\.[^\s@,;<>]{2,}$/;

function normalizeEmail(raw: string) {
  const angled = raw.match(/<([^>]+)>/);
  const value = angled?.[1] ?? raw;

  return value
    .trim()
    .replace(/^[<"']+/, "")
    .replace(/[>"',;.]+$/, "")
    .toLowerCase();
}

function splitEntries(raw: string) {
  return raw.split(/[\s,;\n\r\t]+/).filter(Boolean);
}

export type EmailTagInputLabels = {
  add: string;
  empty: string;
  count: (count: number) => string;
  remove: (value: string) => string;
  clear: string;
  showAll: (count: number) => string;
  showFewer: string;
  invalid: (value: string) => string;
  skippedInvalid: (count: number) => string;
  skippedDuplicate: (count: number) => string;
};

export function EmailTagInput({
  id,
  values,
  onChange,
  labels,
  placeholder,
  collapseAfter = 25,
  max = 200,
  disabled,
  invalid,
  className,
}: {
  id?: string;
  values: string[];
  onChange: (values: string[]) => void;
  labels: EmailTagInputLabels;
  placeholder?: string;
  collapseAfter?: number;
  max?: number;
  disabled?: boolean;
  invalid?: boolean;
  className?: string;
}) {
  const [draft, setDraft] = React.useState("");
  const [notice, setNotice] = React.useState("");
  const [expanded, setExpanded] = React.useState(false);

  const noticeId = id ? `${id}-notice` : undefined;

  function addMany(raw: string) {
    const entries = splitEntries(raw);
    if (entries.length === 0) return;

    const existing = new Set(values);
    const accepted: string[] = [];

    let invalidCount = 0;
    let duplicateCount = 0;
    let lastInvalid = "";

    for (const entry of entries) {
      const value = normalizeEmail(entry);

      if (!EMAIL_PATTERN.test(value)) {
        invalidCount++;
        lastInvalid = entry.trim();
        continue;
      }

      if (existing.has(value)) {
        duplicateCount++;
        continue;
      }

      if (values.length + accepted.length >= max) {
        invalidCount++;
        continue;
      }

      existing.add(value);
      accepted.push(value);
    }

    if (accepted.length > 0) onChange([...values, ...accepted]);

    const messages: string[] = [];

    if (invalidCount > 0) {
      messages.push(
        entries.length === 1
          ? labels.invalid(lastInvalid)
          : labels.skippedInvalid(invalidCount),
      );
    }

    if (duplicateCount > 0) messages.push(labels.skippedDuplicate(duplicateCount));

    setNotice(messages.join(" "));

    if (invalidCount === 0) setDraft("");
  }

  const visible = expanded ? values : values.slice(0, collapseAfter);
  const hidden = values.length - visible.length;

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div className="flex gap-2">
        <Input
          id={id}
          value={draft}
          placeholder={placeholder}
          disabled={disabled}
          spellCheck={false}
          aria-invalid={invalid || undefined}
          aria-describedby={notice ? noticeId : undefined}
          onChange={(event) => {
            setDraft(event.target.value);
            setNotice("");
          }}
          onPaste={(event) => {
            const pasted = event.clipboardData.getData("text");
            if (!/[\s,;\n]/.test(pasted)) return;

            event.preventDefault();
            addMany(pasted);
          }}
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === "," || event.key === ";") {
              event.preventDefault();
              addMany(draft);
              return;
            }

            if (event.key === "Backspace" && draft === "" && values.length > 0) {
              event.preventDefault();
              onChange(values.slice(0, -1));
            }
          }}
        />

        <Button
          type="button"
          variant="secondary"
          className="shrink-0"
          disabled={disabled || !draft.trim()}
          onClick={() => addMany(draft)}
        >
          <Plus />
          {labels.add}
        </Button>
      </div>

      {notice ? (
        <p id={noticeId} role="status" className="text-xs text-error-text">
          {notice}
        </p>
      ) : null}

      {values.length === 0 ? (
        <p className="text-xs text-muted-foreground">{labels.empty}</p>
      ) : (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs text-muted-foreground tabular-nums">
              {labels.count(values.length)}
            </span>

            <Button
              type="button"
              size="sm"
              variant="ghost"
              disabled={disabled}
              onClick={() => {
                onChange([]);
                setNotice("");
              }}
            >
              {labels.clear}
            </Button>
          </div>

          <div className="max-h-56 overflow-y-auto rounded-md border p-2">
            <div className="flex flex-wrap gap-1.5">
              {visible.map((value) => (
                <Badge key={value} variant="secondary" className="gap-1 pr-1 font-mono">
                  <span className="max-w-56 truncate">{value}</span>
                  <button
                    type="button"
                    aria-label={labels.remove(value)}
                    disabled={disabled}
                    onClick={() => onChange(values.filter((item) => item !== value))}
                    className="rounded-full p-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-40"
                  >
                    <X className="size-3" />
                  </button>
                </Badge>
              ))}
            </div>
          </div>

          {values.length > collapseAfter ? (
            <button
              type="button"
              className="self-start text-xs font-medium text-primary hover:underline"
              onClick={() => setExpanded((current) => !current)}
            >
              {expanded ? labels.showFewer : labels.showAll(hidden)}
            </button>
          ) : null}
        </div>
      )}
    </div>
  );
}

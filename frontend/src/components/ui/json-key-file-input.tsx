"use client";

import { CloudUpload, FileCode, X } from "lucide-react";
import * as React from "react";

import { Button } from "@/components/ui/button";
import {
  parseServiceAccountKey,
  type ServiceAccountKey,
  type ServiceAccountKeyError,
} from "@/lib/gcp-service-account";
import { cn } from "@/lib/utils";

const ALLOWED_TYPES = ["application/json", "text/json", "text/plain"];

export type JsonKeyFileInputLabels = {
  dropzone: string;
  browse: string;
  hint: string;
  replace: string;
  remove: string;
  projectId: string;
  clientEmail: string;
  errors: Record<ServiceAccountKeyError, string>;
};

export function JsonKeyFileInput({
  id,
  value,
  onChange,
  onError,
  labels,
  storedDerived,
  disabled,
  invalid,
  maxBytes = 64 * 1024,
  className,
}: {
  id: string;
  value: ServiceAccountKey | null;
  onChange: (value: ServiceAccountKey | null) => void;
  onError?: (message: string) => void;
  labels: JsonKeyFileInputLabels;
  storedDerived?: { projectId: string; clientEmail: string };
  disabled?: boolean;
  invalid?: boolean;
  maxBytes?: number;
  className?: string;
}) {
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = React.useState(false);

  function fail(error: ServiceAccountKeyError) {
    onError?.(labels.errors[error]);
  }

  function readFile(file: File) {
    if (file.size > maxBytes) {
      fail("tooLarge");
      return;
    }

    if (file.type !== "" && !ALLOWED_TYPES.includes(file.type)) {
      fail("wrongType");
      return;
    }

    const reader = new FileReader();

    reader.onerror = () => fail("unreadable");
    reader.onload = () => {
      const text = typeof reader.result === "string" ? reader.result : "";
      const result = parseServiceAccountKey(file.name, text);

      if (!result.ok) {
        fail(result.error);
        return;
      }

      onChange(result.value);
    };

    reader.readAsText(file);
  }

  const derived = value
    ? { projectId: value.projectId, clientEmail: value.clientEmail }
    : storedDerived;

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <input
        ref={inputRef}
        id={id}
        type="file"
        className="sr-only"
        accept="application/json,.json"
        disabled={disabled}
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (file) readFile(file);
          event.target.value = "";
        }}
      />

      {value ? (
        <div className="flex items-center gap-2 rounded-lg border p-3">
          <FileCode className="size-4 shrink-0 text-muted-foreground" />
          <span className="min-w-0 flex-1 truncate text-sm font-medium">{value.fileName}</span>

          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={disabled}
            onClick={() => inputRef.current?.click()}
          >
            {labels.replace}
          </Button>

          <button
            type="button"
            aria-label={labels.remove}
            disabled={disabled}
            onClick={() => onChange(null)}
            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-40"
          >
            <X className="size-4" />
          </button>
        </div>
      ) : (
        <label
          htmlFor={id}
          onDragEnter={(event) => {
            event.preventDefault();
            setDragging(true);
          }}
          onDragOver={(event) => event.preventDefault()}
          onDragLeave={(event) => {
            if (event.currentTarget.contains(event.relatedTarget as Node)) return;
            setDragging(false);
          }}
          onDrop={(event) => {
            event.preventDefault();
            setDragging(false);

            const file = event.dataTransfer.files?.[0];
            if (file) readFile(file);
          }}
          className={cn(
            "flex min-h-32 cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-6 py-6 text-center transition-colors",
            dragging ? "border-primary bg-primary/5" : "hover:bg-surface-container-low",
            invalid && "border-error",
            disabled && "pointer-events-none opacity-50",
          )}
        >
          <CloudUpload className="size-5 text-muted-foreground" />
          <span className="text-sm font-medium">{labels.dropzone}</span>
          <span className="text-xs text-muted-foreground">{labels.hint}</span>
        </label>
      )}

      {derived ? (
        <dl className="grid gap-1 rounded-md border bg-surface-container-low p-3 text-xs">
          <div className="flex gap-2">
            <dt className="w-28 shrink-0 text-muted-foreground">{labels.projectId}</dt>
            <dd className="min-w-0 flex-1 truncate font-mono select-all">{derived.projectId}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="w-28 shrink-0 text-muted-foreground">{labels.clientEmail}</dt>
            <dd className="min-w-0 flex-1 truncate font-mono select-all">{derived.clientEmail}</dd>
          </div>
        </dl>
      ) : null}
    </div>
  );
}

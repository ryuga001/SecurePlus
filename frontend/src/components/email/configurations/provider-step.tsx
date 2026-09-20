"use client";

import { cn } from "@/lib/utils";
import type { EmailProvider } from "@/store/api/email-configurations-api";

import { providerMeta, providerOrder } from "./provider-meta";

export function ProviderStep({
  value,
  onSelect,
}: {
  value: EmailProvider | null;
  onSelect: (provider: EmailProvider) => void;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {providerOrder.map((provider) => {
        const meta = providerMeta[provider];
        const selected = value === provider;

        return (
          <button
            key={provider}
            type="button"
            aria-pressed={selected}
            onClick={() => onSelect(provider)}
            className={cn(
              "flex items-start gap-3 border p-4 text-left transition-colors hover:border-primary hover:bg-primary/5 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
              selected && "border-primary bg-primary/8"
            )}
          >
            <span className="flex size-10 shrink-0 items-center justify-center bg-primary/10 font-heading text-lg font-semibold text-primary">
              {meta.monogram}
            </span>

            <span className="flex flex-col gap-1">
              <span className="text-sm font-medium">{meta.label}</span>
              <span className="text-xs leading-relaxed text-muted-foreground">
                {meta.description}
              </span>
            </span>
          </button>
        );
      })}
    </div>
  );
}

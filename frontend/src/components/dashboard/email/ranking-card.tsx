"use client";

import { Inbox } from "lucide-react";
import { useTranslations } from "next-intl";

import { useAuth } from "@/components/auth-provider";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCount } from "@/lib/dashboard";

export type RankingRow = {
  key: string;
  label: string;
  badge?: string;
  hint?: string;
  count: number;
};

export function RankingCard({
  title,
  subtitle,
  rows,
  emptyMessage,
}: {
  title: string;
  subtitle: string;
  rows: RankingRow[];
  emptyMessage: string;
}) {
  const t = useTranslations("dashboard.email");
  const { identity } = useAuth();
  const language = identity?.branding?.language ?? "ENGLISH";

  const peak = rows.reduce((highest, row) => Math.max(highest, row.count), 0);

  return (
    <section className="flex flex-col rounded-lg border bg-surface p-5">
      <header className="mb-5 flex flex-col">
        <div className="flex items-center justify-between gap-2">
          <h3 className="truncate font-heading text-base leading-6 font-semibold">{title}</h3>
          <span className="shrink-0 font-mono text-[11px] tracking-wide text-text-tertiary">
            {t("topFive")}
          </span>
        </div>
        <span className="text-xs text-muted-foreground">{subtitle}</span>
      </header>

      {rows.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 rounded-md border border-dashed py-8 text-center">
          <Inbox className="size-5 text-text-tertiary" />
          <span className="text-xs text-muted-foreground">{emptyMessage}</span>
        </div>
      ) : (
        <ol className="flex flex-col gap-4">
          {rows.map((row) => (
            <li key={row.key} className="flex flex-col gap-1.5">
              <div className="flex items-center justify-between gap-2">
                <div className="flex min-w-0 items-center gap-1.5">
                  <span className="truncate text-[13px] font-medium" title={row.label}>
                    {row.label}
                  </span>
                  {row.badge ? (
                    <Badge variant="secondary" size="sm" className="shrink-0 font-mono uppercase">
                      {row.badge}
                    </Badge>
                  ) : null}
                </div>

                <span className="shrink-0 font-mono text-xs font-semibold tabular-nums">
                  {formatCount(row.count, language)}
                </span>
              </div>

              <div className="h-2.5 w-full overflow-hidden bg-surface-container">
                <div
                  className="h-full rounded-r-lg bg-chart-1"
                  style={{ width: `${peak > 0 ? Math.max((row.count / peak) * 100, 2) : 0}%` }}
                />
              </div>

              {row.hint ? (
                <span className="text-[10px] text-text-tertiary">{row.hint}</span>
              ) : null}
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

export function RankingCardSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <section className="flex flex-col rounded-lg border bg-surface p-5">
      <div className="mb-5 flex flex-col gap-2">
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-3 w-52" />
      </div>

      <div className="flex flex-col gap-4">
        {Array.from({ length: rows }, (_, index) => (
          <div key={index} className="flex flex-col gap-1.5">
            <div className="flex items-center justify-between">
              <Skeleton className="h-3.5 w-32" />
              <Skeleton className="h-3.5 w-8" />
            </div>
            <Skeleton className="h-2.5 w-full" />
          </div>
        ))}
      </div>
    </section>
  );
}

"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { useAuth } from "@/components/auth-provider";
import { Skeleton } from "@/components/ui/skeleton";
import { axisTicks, bucketLabel, formatCompact, formatCount, niceCeiling } from "@/lib/dashboard";
import { cn } from "@/lib/utils";
import type { AnalyticsPeriod, TrendPoint } from "@/store/api/email-analytics-api";

const PLOT_HEIGHT = 208;
const SEGMENT_GAP = 2;

export function IncidentTrend({
  points,
  bucket,
}: {
  points: TrendPoint[];
  bucket: AnalyticsPeriod;
}) {
  const t = useTranslations("dashboard.email.trend");
  const { identity } = useAuth();
  const [hovered, setHovered] = React.useState<number | null>(null);

  const timezone = identity?.branding?.timezone ?? "UTC";
  const language = identity?.branding?.language ?? "ENGLISH";

  const peak = points.reduce((highest, point) => Math.max(highest, point.total), 0);
  const max = niceCeiling(peak);
  const ticks = axisTicks(max);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-col">
          <h2 className="font-heading text-base leading-6 font-semibold">{t("title")}</h2>
          <span className="text-xs text-muted-foreground">{t("subtitle")}</span>
        </div>

        <div className="flex items-center gap-4 rounded-md border bg-surface-container-low px-3 py-1.5">
          <LegendKey className="bg-chart-1" label={t("blocked")} />
          <LegendKey className="bg-chart-2" label={t("flagged")} />
        </div>
      </div>

      <div className="relative flex w-full gap-2">
        <div
          aria-hidden
          className="flex w-9 shrink-0 flex-col justify-between text-right font-mono text-[11px] tabular-nums text-text-tertiary"
          style={{ height: PLOT_HEIGHT }}
        >
          {ticks.map((tick) => (
            <span key={tick} className="leading-none">
              {formatCompact(tick, language)}
            </span>
          ))}
        </div>

        <div className="relative min-w-0 flex-1">
          <div
            aria-hidden
            className="absolute inset-x-0 top-0 flex flex-col justify-between"
            style={{ height: PLOT_HEIGHT }}
          >
            {ticks.map((tick, index) => (
              <div
                key={tick}
                className={cn("h-px w-full", index === ticks.length - 1 ? "bg-border" : "bg-border-subtle")}
              />
            ))}
          </div>

          <ol
            className="relative flex items-end justify-between gap-1.5"
            style={{ height: PLOT_HEIGHT }}
            onMouseLeave={() => setHovered(null)}
          >
            {points.map((point, index) => (
              <Bar
                key={point.bucket_start}
                point={point}
                max={max}
                active={hovered === index}
                label={bucketLabel(point.bucket_start, bucket, timezone, language)}
                onEnter={() => setHovered(index)}
                onLeave={() => setHovered(null)}
              />
            ))}
          </ol>

          <div className="mt-2 flex justify-between gap-1.5">
            {points.map((point, index) => (
              <span
                key={point.bucket_start}
                className={cn(
                  "min-w-0 flex-1 truncate text-center font-mono text-[10px] tabular-nums",
                  hovered === index ? "font-semibold text-primary" : "text-text-tertiary",
                )}
              >
                {bucketLabel(point.bucket_start, bucket, timezone, language)}
              </span>
            ))}
          </div>
        </div>
      </div>

      <table className="sr-only">
        <caption>{t("title")}</caption>
        <thead>
          <tr>
            <th scope="col">{t("bucket")}</th>
            <th scope="col">{t("blocked")}</th>
            <th scope="col">{t("flagged")}</th>
            <th scope="col">{t("total")}</th>
          </tr>
        </thead>
        <tbody>
          {points.map((point) => (
            <tr key={point.bucket_start}>
              <th scope="row">{bucketLabel(point.bucket_start, bucket, timezone, language)}</th>
              <td>{formatCount(point.blocked, language)}</td>
              <td>{formatCount(point.flagged, language)}</td>
              <td>{formatCount(point.total, language)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Bar({
  point,
  max,
  active,
  label,
  onEnter,
  onLeave,
}: {
  point: TrendPoint;
  max: number;
  active: boolean;
  label: string;
  onEnter: () => void;
  onLeave: () => void;
}) {
  const t = useTranslations("dashboard.email.trend");
  const { identity } = useAuth();
  const language = identity?.branding?.language ?? "ENGLISH";

  const scale = (value: number) => (value / max) * PLOT_HEIGHT;
  const blocked = point.blocked > 0 ? Math.max(scale(point.blocked), 3) : 0;
  const flagged = point.flagged > 0 ? Math.max(scale(point.flagged), 3) : 0;

  return (
    <li className="relative flex min-w-0 flex-1 justify-center self-stretch">
      <button
        type="button"
        onMouseEnter={onEnter}
        onFocus={onEnter}
        onBlur={onLeave}
        aria-label={t("barLabel", {
          bucket: label,
          blocked: formatCount(point.blocked, language),
          flagged: formatCount(point.flagged, language),
        })}
        className="flex h-full w-full max-w-6 cursor-pointer flex-col justify-end rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        {flagged > 0 ? (
          <span
            aria-hidden
            className={cn("w-full rounded-t-lg bg-chart-2", active && "brightness-110")}
            style={{ height: flagged }}
          />
        ) : null}

        {flagged > 0 && blocked > 0 ? <span aria-hidden style={{ height: SEGMENT_GAP }} /> : null}

        {blocked > 0 ? (
          <span
            aria-hidden
            className={cn("w-full bg-chart-1", flagged === 0 && "rounded-t-lg", active && "brightness-110")}
            style={{ height: blocked }}
          />
        ) : null}
      </button>

      {active && point.total > 0 ? (
        <div
          role="tooltip"
          className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-2 w-44 -translate-x-1/2 rounded-lg border bg-surface-container-highest p-2 shadow-e2"
        >
          <div className="flex items-center justify-between gap-2 border-b pb-1">
            <span className="truncate font-mono text-[10px] text-text-secondary">{label}</span>
            <span className="font-mono text-[11px] font-semibold tabular-nums">
              {t("totalValue", { count: formatCount(point.total, language) })}
            </span>
          </div>

          <dl className="mt-1 flex flex-col gap-0.5">
            <TooltipRow swatch="bg-chart-1" label={t("blocked")} value={formatCount(point.blocked, language)} />
            <TooltipRow swatch="bg-chart-2" label={t("flagged")} value={formatCount(point.flagged, language)} />
          </dl>
        </div>
      ) : null}
    </li>
  );
}

function TooltipRow({ swatch, label, value }: { swatch: string; label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-2 text-[11px]">
      <dt className="flex items-center gap-1.5 text-text-secondary">
        <span aria-hidden className={cn("size-1.5 rounded-full", swatch)} />
        {label}
      </dt>
      <dd className="font-mono font-medium tabular-nums">{value}</dd>
    </div>
  );
}

function LegendKey({ className, label }: { className: string; label: string }) {
  return (
    <span className="flex items-center gap-1.5 text-xs">
      <span aria-hidden className={cn("size-2.5 rounded-[2px]", className)} />
      {label}
    </span>
  );
}

export function IncidentTrendSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-3 w-64" />
        </div>
        <Skeleton className="h-8 w-40" />
      </div>

      <div className="flex gap-2" style={{ height: PLOT_HEIGHT }}>
        <Skeleton className="w-9 shrink-0" />
        <Skeleton className="flex-1" />
      </div>
    </div>
  );
}

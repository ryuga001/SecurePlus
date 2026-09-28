"use client";

import { useTranslations } from "next-intl";

import { ANALYTICS_PERIODS } from "@/lib/dashboard";
import { cn } from "@/lib/utils";
import type { AnalyticsPeriod } from "@/store/api/email-analytics-api";

export function PeriodToggle({
  value,
  onChange,
  disabled,
}: {
  value: AnalyticsPeriod;
  onChange: (period: AnalyticsPeriod) => void;
  disabled?: boolean;
}) {
  const t = useTranslations("dashboard.email.period");

  return (
    <div className="flex flex-col items-start gap-1 sm:items-end">
      <div
        role="group"
        aria-label={t("label")}
        className="inline-flex items-center gap-0.5 rounded-lg border bg-surface-container-low p-1"
      >
        {ANALYTICS_PERIODS.map((period) => {
          const active = period === value;

          return (
            <button
              key={period}
              type="button"
              disabled={disabled}
              aria-pressed={active}
              onClick={() => onChange(period)}
              className={cn(
                "rounded-md px-3 py-1 text-xs font-medium transition-colors disabled:pointer-events-none disabled:opacity-60",
                active
                  ? "bg-surface text-foreground shadow-e1"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {t(`short.${period}`)}
            </button>
          );
        })}
      </div>

      <span className="font-mono text-[11px] tracking-wide text-text-tertiary">
        {t(`caption.${value}`)}
      </span>
    </div>
  );
}

"use client";

import { Ban, Flag, Inbox } from "lucide-react";
import { useTranslations } from "next-intl";
import { useRouter } from "next/navigation";

import { useAuth } from "@/components/auth-provider";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { relativeTime } from "@/lib/dashboard";
import { cn } from "@/lib/utils";
import {
  useListEmailIncidentsQuery,
  type EmailIncidentListItem,
} from "@/store/api/email-incidents-api";

const RECENT_LIMIT = 5;

export function RecentActivity() {
  const t = useTranslations("dashboard.email.activity");
  const router = useRouter();
  const { identity } = useAuth();
  const language = identity?.branding?.language ?? "ENGLISH";

  const { data, isLoading, isError } = useListEmailIncidentsQuery({
    page: 1,
    pageSize: RECENT_LIMIT,
    sortBy: "created_at",
    sortDir: "desc",
    filters: {},
  });

  const incidents = data?.items ?? [];

  return (
    <section className="flex flex-col rounded-lg border bg-surface p-5">
      <header className="mb-4 flex items-center justify-between gap-3">
        <div className="flex min-w-0 flex-col">
          <h2 className="font-heading text-base leading-6 font-semibold">{t("title")}</h2>
          <span className="text-xs text-muted-foreground">{t("subtitle")}</span>
        </div>

        <Button
          type="button"
          size="sm"
          variant="ghost"
          onClick={() => router.push("/admin/email/audits/incidents")}
        >
          {t("viewAll")}
        </Button>
      </header>

      {isLoading ? <ActivityRows /> : null}

      {!isLoading && (isError || incidents.length === 0) ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed py-10 text-center">
          <Inbox className="size-5 text-text-tertiary" />
          <span className="text-xs text-muted-foreground">
            {isError ? t("loadFailed") : t("empty")}
          </span>
        </div>
      ) : null}

      {!isLoading && !isError && incidents.length > 0 ? (
        <ul className="flex flex-col divide-y">
          {incidents.map((incident) => (
            <ActivityRow key={incident.correlation_id} incident={incident} language={language} />
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function ActivityRow({
  incident,
  language,
}: {
  incident: EmailIncidentListItem;
  language: Parameters<typeof relativeTime>[1];
}) {
  const t = useTranslations("dashboard.email.activity");
  const blocked = incident.effective_action === "BLOCK";
  const Icon = blocked ? Ban : Flag;

  return (
    <li className="flex items-center justify-between gap-3 py-3">
      <div className="flex min-w-0 items-center gap-3">
        <span
          aria-hidden
          className={cn(
            "flex size-8 shrink-0 items-center justify-center rounded-md bg-surface-container",
            blocked ? "text-chart-1" : "text-chart-2",
          )}
        >
          <Icon className="size-4" />
        </span>

        <div className="flex min-w-0 flex-col">
          <span className="truncate text-[13px] font-medium">
            {blocked ? t("blockedTitle") : t("flaggedTitle")}
          </span>
          <span className="truncate text-[10px] text-text-tertiary">
            {t("detail", { sender: incident.from, matches: incident.match_count })}
          </span>
        </div>
      </div>

      <span className="shrink-0 font-mono text-[10px] text-text-tertiary">
        {relativeTime(incident.created_at, language)}
      </span>
    </li>
  );
}

function ActivityRows() {
  return (
    <div className="flex flex-col divide-y">
      {Array.from({ length: 3 }, (_, index) => (
        <div key={index} className="flex items-center justify-between gap-3 py-3">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <Skeleton className="size-8 shrink-0 rounded-md" />
            <div className="flex min-w-0 flex-1 flex-col gap-1.5">
              <Skeleton className="h-3.5 w-48" />
              <Skeleton className="h-3 w-32" />
            </div>
          </div>
          <Skeleton className="h-3 w-14" />
        </div>
      ))}
    </div>
  );
}

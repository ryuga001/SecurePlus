"use client";

import { Ban, CircleAlert, Flag, ShieldAlert } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";

import { useAuth } from "@/components/auth-provider";
import { IncidentTrend, IncidentTrendSkeleton } from "@/components/dashboard/email/incident-trend";
import { PeriodToggle } from "@/components/dashboard/email/period-toggle";
import {
  RankingCard,
  RankingCardSkeleton,
  type RankingRow,
} from "@/components/dashboard/email/ranking-card";
import { RecentActivity } from "@/components/dashboard/email/recent-activity";
import { StatCard, StatCardSkeleton } from "@/components/dashboard/email/stat-card";
import { Button } from "@/components/ui/button";
import { usePrivileges } from "@/hooks/use-privileges";
import { apiErrorMessage } from "@/lib/api-error";
import { formatCount } from "@/lib/dashboard";
import {
  useGetEmailAnalyticsQuery,
  type AnalyticsPeriod,
} from "@/store/api/email-analytics-api";

const INCIDENT_VIEW = "admin.email.incident.view";

export default function DashboardPage() {
  const t = useTranslations("dashboard");
  const email = useTranslations("dashboard.email");
  const table = useTranslations("table");
  const { identity } = useAuth();

  const { has, loading: privilegesLoading } = usePrivileges();
  const [period, setPeriod] = React.useState<AnalyticsPeriod>("week");

  const canView = has(INCIDENT_VIEW);

  const { data, isLoading, isFetching, isError, error, refetch } = useGetEmailAnalyticsQuery(period, {
    skip: privilegesLoading || !canView,
  });

  const language = identity?.branding?.language ?? "ENGLISH";
  const pending = privilegesLoading || isLoading;

  const senderRows: RankingRow[] = (data?.top_users ?? []).map((row) => ({
    key: row.email,
    label: row.email,
    count: row.total,
    hint: email("senders.split", {
      blocked: formatCount(row.blocked, language),
      flagged: formatCount(row.flagged, language),
    }),
  }));

  const policyRows: RankingRow[] = (data?.top_policies ?? []).map((row) => ({
    key: String(row.policy_id),
    label: row.policy_name,
    count: row.count,
  }));

  const ruleRows: RankingRow[] = (data?.top_rules ?? []).map((row) => ({
    key: String(row.rule_id),
    label: row.rule_name,
    badge: row.rule_type,
    count: row.count,
  }));

  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-col justify-between gap-4 md:flex-row md:items-end">
        <div>
          <h1 className="font-heading text-[28px] leading-9 font-semibold tracking-[-0.01em]">
            {t("welcome", { name: identity?.user.first_name ?? "" })}
          </h1>
          <p className="mt-1 text-sm leading-5 text-muted-foreground">{t("subheading")}</p>
        </div>

        {canView ? (
          <PeriodToggle value={period} onChange={setPeriod} disabled={isFetching} />
        ) : null}
      </header>

      {!privilegesLoading && !canView ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed bg-surface p-10 text-center">
          <ShieldAlert className="size-6 text-text-tertiary" />
          <p className="text-sm font-medium">{email("denied.title")}</p>
          <p className="max-w-md text-sm text-muted-foreground">{email("denied.description")}</p>
        </div>
      ) : null}

      {canView && isError ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-lg border bg-surface p-10 text-center">
          <CircleAlert className="size-6 text-error" />
          <p className="max-w-md text-sm text-muted-foreground">
            {apiErrorMessage(error, email("loadFailed"))}
          </p>
          <Button type="button" variant="outline" onClick={() => refetch()}>
            {table("tryAgain")}
          </Button>
        </div>
      ) : null}

      {canView && !isError ? (
        <div className="flex flex-col gap-6">
          <section
            aria-label={email("summary.label")}
            className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"
          >
            {pending ? (
              <>
                <StatCardSkeleton hero />
                <StatCardSkeleton />
                <StatCardSkeleton />
              </>
            ) : (
              <>
                <StatCard
                  hero
                  icon={ShieldAlert}
                  label={email("summary.total")}
                  value={formatCount(data?.summary.total ?? 0, language)}
                  hint={email("summary.totalHint")}
                />
                <StatCard
                  tone="blocked"
                  icon={Ban}
                  label={email("summary.blocked")}
                  value={formatCount(data?.summary.blocked ?? 0, language)}
                  hint={email("summary.blockedHint")}
                />
                <StatCard
                  tone="flagged"
                  icon={Flag}
                  label={email("summary.flagged")}
                  value={formatCount(data?.summary.flagged ?? 0, language)}
                  hint={email("summary.flaggedHint")}
                />
              </>
            )}
          </section>

          <section className="rounded-lg border bg-surface p-5">
            {pending || !data ? (
              <IncidentTrendSkeleton />
            ) : (
              <IncidentTrend points={data.trend.points} bucket={data.trend.bucket} />
            )}
          </section>

          <div className="grid gap-4 lg:grid-cols-[repeat(3,minmax(0,1fr))]">
            {pending ? (
              <>
                <RankingCardSkeleton />
                <RankingCardSkeleton />
                <RankingCardSkeleton />
              </>
            ) : (
              <>
                <RankingCard
                  title={email("senders.title")}
                  subtitle={email("senders.subtitle")}
                  rows={senderRows}
                  emptyMessage={email("senders.empty")}
                />
                <RankingCard
                  title={email("policies.title")}
                  subtitle={email("policies.subtitle")}
                  rows={policyRows}
                  emptyMessage={email("policies.empty")}
                />
                <RankingCard
                  title={email("rules.title")}
                  subtitle={email("rules.subtitle")}
                  rows={ruleRows}
                  emptyMessage={email("rules.empty")}
                />
              </>
            )}
          </div>

          <RecentActivity />
        </div>
      ) : null}
    </div>
  );
}

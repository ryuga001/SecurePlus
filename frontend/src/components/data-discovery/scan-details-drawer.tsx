"use client";

import { CircleAlert, FileSearch, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import * as React from "react";

import { useAuth } from "@/components/auth-provider";
import { StatusBadge } from "@/components/data-table/status-badge";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { apiErrorMessage } from "@/lib/api-error";
import {
  SCAN_POLL_INTERVAL,
  formatBytes,
  formatDuration,
  isScanActive,
} from "@/lib/data-discovery";
import { formatInZone } from "@/lib/datetime";
import {
  dataDiscoveryScansApi,
  useGetDiscoveryScanQuery,
  type DiscoveryScan,
} from "@/store/api/data-discovery-scans-api";

export function useLiveDiscoveryScan(scanId: number | null) {
  const skip = scanId === null;
  const cached = dataDiscoveryScansApi.endpoints.getDiscoveryScan.useQueryState(scanId ?? 0, { skip });
  const live = !cached.data || isScanActive(cached.data.status);

  return useGetDiscoveryScanQuery(scanId ?? 0, {
    skip,
    pollingInterval: live ? SCAN_POLL_INTERVAL : 0,
    skipPollingIfUnfocused: true,
  });
}

export function useScanErrorLabel() {
  const t = useTranslations("data-discovery.scans.errors");

  return React.useCallback(
    (code: string | null | undefined) => {
      if (!code) return "";

      return t.has(code) ? t(code) : code;
    },
    [t],
  );
}

export function useZonedDate() {
  const { identity } = useAuth();
  const timezone = identity?.branding?.timezone ?? "UTC";
  const language = identity?.branding?.language ?? "ENGLISH";

  return React.useCallback(
    (value: string | null | undefined) => (value ? formatInZone(value, timezone, language) : ""),
    [timezone, language],
  );
}

export function scanElapsed(scan: Pick<DiscoveryScan, "started_at" | "finished_at">) {
  if (!scan.started_at) return "";

  const end = scan.finished_at ? new Date(scan.finished_at).getTime() : Date.now();

  return formatDuration(end - new Date(scan.started_at).getTime());
}

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <p className="text-xs font-medium tracking-wide text-text-tertiary uppercase">{label}</p>
      <div className="mt-1 truncate text-sm">{value === "" || value === undefined ? "—" : value}</div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="border-t pt-4">
      <h3 className="mb-3 text-sm font-semibold">{title}</h3>
      {children}
    </section>
  );
}

function Metric({ label, value, tone }: { label: string; value: string; tone?: "error" | "primary" }) {
  return (
    <div className="border bg-surface-container-low p-3">
      <p className="text-xs font-medium tracking-wide text-text-tertiary uppercase">{label}</p>
      <p
        className={
          tone === "error"
            ? "mt-1 text-lg font-semibold text-error-text"
            : tone === "primary"
              ? "mt-1 text-lg font-semibold text-primary"
              : "mt-1 text-lg font-semibold"
        }
      >
        {value}
      </p>
    </div>
  );
}

function Details({ scan }: { scan: DiscoveryScan }) {
  const t = useTranslations("data-discovery.scans.details");
  const router = useRouter();
  const errorLabel = useScanErrorLabel();
  const formatDate = useZonedDate();

  const counters = scan.counters;
  const finishedTargets = counters.completed_targets + counters.failed_targets;
  const targetProgress =
    counters.total_targets > 0 ? Math.round((finishedTargets / counters.total_targets) * 100) : 0;
  const number = (value: number) => value.toLocaleString();

  return (
    <div className="mt-6 flex flex-col gap-6">
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <Field label={t("policy")} value={scan.policy_name} />
        <Field label={t("status")} value={<StatusBadge status={scan.status} />} />
        <Field label={t("error")} value={errorLabel(scan.error_code)} />
        <Field label={t("requested")} value={formatDate(scan.created_at)} />
        <Field label={t("started")} value={formatDate(scan.started_at)} />
        <Field label={t("finished")} value={formatDate(scan.finished_at)} />
        <Field label={t("duration")} value={scanElapsed(scan)} />
      </div>

      <Section title={t("progress")}>
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">
              {t("targetsDone", { done: finishedTargets, total: counters.total_targets })}
            </span>
            <span className="font-medium">{targetProgress}%</span>
          </div>

          <div className="h-2 w-full overflow-hidden bg-surface-container-low">
            <div
              className={isScanActive(scan.status) ? "h-full bg-primary transition-all" : "h-full bg-primary"}
              style={{ width: `${targetProgress}%` }}
            />
          </div>

          {isScanActive(scan.status) ? (
            <p className="flex items-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="size-3 animate-spin" />
              {t("live")}
            </p>
          ) : null}
        </div>
      </Section>

      <Section title={t("files")}>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Metric label={t("discovered")} value={number(counters.files_discovered)} />
          <Metric label={t("supported")} value={number(counters.files_supported)} />
          <Metric label={t("skipped")} value={number(counters.files_skipped)} />
          <Metric label={t("processed")} value={number(counters.files_processed)} />
          <Metric label={t("succeeded")} value={number(counters.files_succeeded)} />
          <Metric
            label={t("failed")}
            value={number(counters.files_failed)}
            tone={counters.files_failed > 0 ? "error" : undefined}
          />
          <Metric
            label={t("findings")}
            value={number(counters.findings_total)}
            tone={counters.findings_total > 0 ? "primary" : undefined}
          />
          <Metric label={t("scanned")} value={formatBytes(counters.bytes_processed)} />
        </div>
      </Section>

      <Section title={t("targets", { count: scan.targets.length })}>
        {scan.targets.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("noTargets")}</p>
        ) : (
          <div className="overflow-x-auto border">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="border-b bg-surface-container-low text-left">
                  {[
                    t("target"),
                    t("status"),
                    t("discovered"),
                    t("skipped"),
                    t("succeeded"),
                    t("failed"),
                    t("findings"),
                    t("error"),
                  ].map((label) => (
                    <th
                      key={label}
                      className="px-3 py-2 text-xs font-semibold whitespace-nowrap text-text-tertiary uppercase"
                    >
                      {label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {scan.targets.map((target) => (
                  <tr key={target.position} className="border-b last:border-b-0">
                    <td className="max-w-56 truncate px-3 py-2 font-mono text-xs" title={target.target}>
                      {target.target}
                    </td>
                    <td className="px-3 py-2">
                      <StatusBadge status={target.status} />
                    </td>
                    <td className="px-3 py-2 text-right">{number(target.files_discovered)}</td>
                    <td className="px-3 py-2 text-right">{number(target.files_skipped)}</td>
                    <td className="px-3 py-2 text-right">{number(target.files_succeeded)}</td>
                    <td className="px-3 py-2 text-right">{number(target.files_failed)}</td>
                    <td className="px-3 py-2 text-right">{number(target.findings_total)}</td>
                    <td className="max-w-48 px-3 py-2 text-muted-foreground">
                      {errorLabel(target.error_code) || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Section>

      <div className="flex justify-end border-t pt-4">
        <Button
          type="button"
          disabled={counters.files_processed === 0}
          onClick={() => router.push(`/admin/data-discovery/scans/${scan.id}`)}
        >
          <FileSearch />
          {t("viewFiles")}
        </Button>
      </div>
    </div>
  );
}

export function ScanDetailsDrawer({
  scanId,
  onOpenChange,
}: {
  scanId: number | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("data-discovery.scans.details");

  const { data, isLoading, isError, error } = useLiveDiscoveryScan(scanId);

  return (
    <DrawerWrapper
      open={scanId !== null}
      title={data ? t("title", { id: data.id }) : t("titleFallback")}
      description={t("description")}
      onClose={() => onOpenChange(false)}
      width="3xl"
    >
      {isLoading ? (
        <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
          <Loader2 className="size-6 animate-spin text-primary" />
          <p className="text-sm">{t("loading")}</p>
        </div>
      ) : null}

      {!isLoading && isError ? (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <CircleAlert className="size-6 text-error-text" />
          <p className="max-w-md text-sm text-muted-foreground">
            {apiErrorMessage(error, t("loadFailed"))}
          </p>
        </div>
      ) : null}

      {!isLoading && !isError && data ? <Details scan={data} /> : null}
    </DrawerWrapper>
  );
}

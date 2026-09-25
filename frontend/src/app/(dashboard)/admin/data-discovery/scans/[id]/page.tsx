"use client";

import { ArrowLeft, CircleAlert, ListFilter, Loader2 } from "lucide-react";
import { useParams, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import * as React from "react";

import { FileResultDrawer } from "@/components/data-discovery/file-result-drawer";
import {
  EMPTY_FILE_RESULT_FILTERS,
  FileResultFilterDrawer,
  activeFileResultFilterCount,
  toFileResultFilterParams,
  type FileResultFilters,
} from "@/components/data-discovery/file-result-filter-drawer";
import {
  scanElapsed,
  useLiveDiscoveryScan,
  useScanErrorLabel,
} from "@/components/data-discovery/scan-details-drawer";
import { AccessDenied } from "@/components/dashboard/access-denied";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import { StatusBadge } from "@/components/data-table/status-badge";
import type {
  ColumnConfig,
  DataTableQueryArgs,
  TableAction,
} from "@/components/data-table/types";
import { Button } from "@/components/ui/button";
import { usePrivileges } from "@/hooks/use-privileges";
import { apiErrorMessage } from "@/lib/api-error";
import {
  SCAN_POLL_INTERVAL,
  formatBytes,
  formatDuration,
  isScanActive,
} from "@/lib/data-discovery";
import { DISCOVERY_SCAN_PRIVILEGES } from "@/lib/privileges";
import {
  useListDiscoveryScanFilesQuery,
  type DiscoveryFileResult,
  type DiscoveryScan,
} from "@/store/api/data-discovery-scans-api";

function Summary({ scan }: { scan: DiscoveryScan }) {
  const t = useTranslations("data-discovery.scans.details");
  const errorLabel = useScanErrorLabel();
  const counters = scan.counters;

  const items: { label: string; value: React.ReactNode }[] = [
    { label: t("status"), value: <StatusBadge status={scan.status} /> },
    { label: t("processed"), value: counters.files_processed.toLocaleString() },
    { label: t("succeeded"), value: counters.files_succeeded.toLocaleString() },
    { label: t("failed"), value: counters.files_failed.toLocaleString() },
    { label: t("findings"), value: counters.findings_total.toLocaleString() },
    { label: t("scanned"), value: formatBytes(counters.bytes_processed) },
    { label: t("duration"), value: scanElapsed(scan) || "—" },
  ];

  return (
    <div className="mb-4 flex flex-col gap-3 border bg-surface-container-low p-4">
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-7">
        {items.map((item) => (
          <div key={item.label} className="min-w-0">
            <p className="text-xs font-medium tracking-wide text-text-tertiary uppercase">{item.label}</p>
            <div className="mt-1 truncate text-sm font-medium">{item.value}</div>
          </div>
        ))}
      </div>

      {scan.error_code ? (
        <p className="text-sm text-error-text">{errorLabel(scan.error_code)}</p>
      ) : null}

      {isScanActive(scan.status) ? (
        <p className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="size-3 animate-spin" />
          {t("live")}
        </p>
      ) : null}
    </div>
  );
}

export default function DiscoveryScanFilesPage() {
  const t = useTranslations("data-discovery.scans");
  const shared = useTranslations("data-discovery");
  const common = useTranslations("common");
  const table = useTranslations("table");
  const router = useRouter();
  const errorLabel = useScanErrorLabel();

  const params = useParams<{ id: string }>();
  const scanId = Number(params.id);
  const validId = Number.isInteger(scanId) && scanId > 0;

  const {
    has,
    loading: privilegesLoading,
    isError: privilegesError,
    refetch: refetchPrivileges,
  } = usePrivileges();

  const canView = has(DISCOVERY_SCAN_PRIVILEGES.view);

  const [filtersOpen, setFiltersOpen] = React.useState(false);
  const [applied, setApplied] = React.useState<FileResultFilters>(EMPTY_FILE_RESULT_FILTERS);
  const [selected, setSelected] = React.useState<DiscoveryFileResult | null>(null);

  const {
    data: scan,
    isLoading: scanLoading,
    isError: scanError,
    error: scanFailure,
  } = useLiveDiscoveryScan(validId && canView ? scanId : null);

  const live = !scan || isScanActive(scan.status);

  const appliedParams = React.useMemo(() => toFileResultFilterParams(applied), [applied]);
  const appliedKey = React.useMemo(() => JSON.stringify(appliedParams), [appliedParams]);
  const activeCount = activeFileResultFilterCount(applied);

  const query = React.useCallback(
    (args: DataTableQueryArgs) =>
      useListDiscoveryScanFilesQuery(
        { ...args, scanId, filters: { ...args.filters, ...appliedParams } },
        { pollingInterval: live ? SCAN_POLL_INTERVAL : 0, skipPollingIfUnfocused: true },
      ),
    [appliedParams, scanId, live],
  );

  const targets = React.useMemo(
    () => new Map((scan?.targets ?? []).map((target) => [target.position, target.target])),
    [scan],
  );

  const columns: ColumnConfig<DiscoveryFileResult>[] = [
    {
      key: "file_name",
      label: t("files.columns.file"),
      render: (value, row) => (
        <div className="flex min-w-0 max-w-md flex-col">
          <span className="truncate font-medium text-foreground" title={String(value)}>
            {String(value)}
          </span>
          <span className="truncate font-mono text-xs text-muted-foreground" title={targets.get(row.target_position)}>
            {targets.get(row.target_position) ?? "—"}
          </span>
        </div>
      ),
    },
    {
      key: "status",
      label: t("files.columns.status"),
      render: (_value, row) => (
        <div className="flex min-w-0 flex-col items-start gap-1">
          <StatusBadge status={row.status} />
          {row.error_code ? (
            <span className="max-w-48 truncate text-xs text-muted-foreground">
              {errorLabel(row.error_code)}
            </span>
          ) : null}
        </div>
      ),
    },
    {
      key: "findings_total",
      label: t("files.columns.findings"),
      align: "right",
      render: (value) => {
        const count = Number(value);

        return (
          <span className={count > 0 ? "font-semibold text-primary" : "text-muted-foreground"}>
            {count.toLocaleString()}
          </span>
        );
      },
    },
    {
      key: "extension",
      label: t("files.columns.type"),
      render: (value) => <span className="uppercase">{String(value) || "—"}</span>,
    },
    {
      key: "size_bytes",
      label: t("files.columns.size"),
      align: "right",
      render: (value) => formatBytes(Number(value)),
    },
    {
      key: "duration_ms",
      label: t("files.columns.took"),
      align: "right",
      render: (value) => formatDuration(Number(value)),
    },
    {
      key: "processed_at",
      label: t("files.columns.processed"),
      type: "datetime",
    },
  ];

  const actions: TableAction[] = [
    {
      key: "filters",
      label: activeCount > 0 ? `${t("files.filters.open")} (${activeCount})` : t("files.filters.open"),
      icon: ListFilter,
      variant: "outline",
      onClick: () => setFiltersOpen(true),
    },
  ];

  const back = (
    <Button type="button" variant="outline" onClick={() => router.push("/admin/data-discovery/scans")}>
      <ArrowLeft />
      {t("files.back")}
    </Button>
  );

  function body() {
    if (privilegesLoading) {
      return (
        <div className="flex min-h-64 items-center justify-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          {common("loading")}
        </div>
      );
    }

    if (privilegesError) {
      return (
        <div className="flex min-h-64 flex-col items-center justify-center gap-3 text-sm">
          <CircleAlert className="size-5 text-error" />
          <p className="text-muted-foreground">{shared("privilegesFailed")}</p>
          <Button type="button" variant="outline" onClick={() => refetchPrivileges()}>
            {table("tryAgain")}
          </Button>
        </div>
      );
    }

    if (!canView) {
      return (
        <AccessDenied title={shared("denied.title")} description={shared("denied.description")} />
      );
    }

    if (!validId || scanError) {
      return (
        <div className="flex min-h-64 flex-col items-center justify-center gap-3 text-center text-sm">
          <CircleAlert className="size-5 text-error" />
          <p className="max-w-md text-muted-foreground">
            {validId ? apiErrorMessage(scanFailure, t("details.loadFailed")) : t("files.invalidScan")}
          </p>
        </div>
      );
    }

    if (scanLoading || !scan) {
      return (
        <div className="flex min-h-64 items-center justify-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          {t("details.loading")}
        </div>
      );
    }

    return (
      <>
        <Summary scan={scan} />

        <DataTable
          key={appliedKey}
          query={query}
          columns={columns}
          filters={[]}
          actions={actions}
          getRowId={(row) => row.id}
          onRowClick={setSelected}
          defaultPageSize={25}
          emptyMessage={activeCount > 0 ? t("files.emptyFiltered") : t("files.empty")}
        />

        <FileResultFilterDrawer
          open={filtersOpen}
          onOpenChange={setFiltersOpen}
          value={applied}
          onApply={setApplied}
        />

        <FileResultDrawer
          file={selected}
          target={selected ? targets.get(selected.target_position) : undefined}
          onOpenChange={(open) => {
            if (!open) setSelected(null);
          }}
        />
      </>
    );
  }

  return (
    <Consolepage
      heading={scan ? t("files.heading", { id: scan.id }) : t("files.headingFallback")}
      subheading={scan?.policy_name ?? t("files.subheading")}
      actions={back}
      data={body()}
    />
  );
}

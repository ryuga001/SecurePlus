"use client";

import {
  CircleAlert,
  Eye,
  FileSearch,
  ListFilter,
  Loader2,
  MoreHorizontal,
  Play,
  RotateCcw,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import {
  ScanDetailsDrawer,
  scanElapsed,
  useScanErrorLabel,
} from "@/components/data-discovery/scan-details-drawer";
import { ScanDrawer } from "@/components/data-discovery/scan-drawer";
import {
  EMPTY_SCAN_FILTERS,
  ScanFilterDrawer,
  activeScanFilterCount,
  toScanFilterParams,
  type ScanFilters,
} from "@/components/data-discovery/scan-filter-drawer";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { usePrivileges } from "@/hooks/use-privileges";
import { apiErrorMessage } from "@/lib/api-error";
import { SCAN_POLL_INTERVAL, isScanActive } from "@/lib/data-discovery";
import { DISCOVERY_SCAN_PRIVILEGES } from "@/lib/privileges";
import {
  dataDiscoveryScansApi,
  useCreateDiscoveryScanMutation,
  useListDiscoveryScansQuery,
  type DiscoveryScanListItem,
} from "@/store/api/data-discovery-scans-api";

function useLiveScanList(args: DataTableQueryArgs) {
  const cached = dataDiscoveryScansApi.endpoints.listDiscoveryScans.useQueryState(args);
  const live = (cached.data?.items ?? []).some((item) => isScanActive(item.status));

  return useListDiscoveryScansQuery(args, {
    pollingInterval: live ? SCAN_POLL_INTERVAL : 0,
    skipPollingIfUnfocused: true,
  });
}

export default function DiscoveryScansPage() {
  const t = useTranslations("data-discovery.scans");
  const shared = useTranslations("data-discovery");
  const common = useTranslations("common");
  const table = useTranslations("table");
  const router = useRouter();
  const errorLabel = useScanErrorLabel();

  const {
    has,
    loading: privilegesLoading,
    isError: privilegesError,
    refetch: refetchPrivileges,
  } = usePrivileges();

  const canView = has(DISCOVERY_SCAN_PRIVILEGES.view);
  const canCreate = has(DISCOVERY_SCAN_PRIVILEGES.create);

  const [filtersOpen, setFiltersOpen] = React.useState(false);
  const [applied, setApplied] = React.useState<ScanFilters>(EMPTY_SCAN_FILTERS);
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [selected, setSelected] = React.useState<number | null>(null);

  const [createScan] = useCreateDiscoveryScanMutation();

  const appliedParams = React.useMemo(() => toScanFilterParams(applied), [applied]);
  const appliedKey = React.useMemo(() => JSON.stringify(appliedParams), [appliedParams]);
  const activeCount = activeScanFilterCount(applied);

  const query = React.useCallback(
    (args: DataTableQueryArgs) =>
      useLiveScanList({
        ...args,
        filters: { ...args.filters, ...appliedParams },
      }),
    [appliedParams],
  );

  async function rescan(row: DiscoveryScanListItem) {
    try {
      const scan = await createScan({ policy_id: row.policy_id }).unwrap();
      toast.success(t("dialog.started", { name: row.policy_name }));
      setSelected(scan.id);
    } catch (error) {
      toast.error(apiErrorMessage(error, t("dialog.startFailed")));
    }
  }

  const columns: ColumnConfig<DiscoveryScanListItem>[] = [
    {
      key: "policy_name",
      label: t("columns.scan"),
      render: (value, row) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate font-medium text-foreground">{String(value) || "—"}</span>
          <span className="truncate text-xs text-muted-foreground">
            {t("scanNumber", { id: row.id })}
          </span>
        </div>
      ),
    },
    {
      key: "status",
      label: t("columns.status"),
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
      key: "targets",
      label: t("columns.targets"),
      accessor: (row) => row.counters,
      render: (_value, row) => (
        <span className="whitespace-nowrap">
          {t("targetsDone", {
            done: row.counters.completed_targets + row.counters.failed_targets,
            total: row.counters.total_targets,
          })}
        </span>
      ),
    },
    {
      key: "files",
      label: t("columns.files"),
      accessor: (row) => row.counters,
      render: (_value, row) => (
        <div className="flex flex-col whitespace-nowrap">
          <span>{row.counters.files_processed.toLocaleString()}</span>
          {row.counters.files_failed > 0 ? (
            <span className="text-xs text-error-text">
              {t("failedCount", { count: row.counters.files_failed })}
            </span>
          ) : null}
        </div>
      ),
    },
    {
      key: "findings",
      label: t("columns.findings"),
      align: "right",
      accessor: (row) => row.counters.findings_total,
      type: "number",
    },
    {
      key: "created_at",
      label: t("columns.requested"),
      type: "datetime",
    },
    {
      key: "duration",
      label: t("columns.duration"),
      accessor: (row) => row.started_at,
      render: (_value, row) => scanElapsed(row) || "—",
    },
    {
      key: "actions",
      label: "",
      align: "right",
      render: (_value, row) => (
        <div onClick={(event) => event.stopPropagation()}>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button type="button" variant="ghost" size="icon">
                  <MoreHorizontal />
                </Button>
              }
            />

            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => setSelected(row.id)}>
                <Eye />
                {t("viewDetails")}
              </DropdownMenuItem>

              <DropdownMenuItem
                disabled={row.counters.files_processed === 0}
                onClick={() => router.push(`/admin/data-discovery/scans/${row.id}`)}
              >
                <FileSearch />
                {t("details.viewFiles")}
              </DropdownMenuItem>

              <DropdownMenuSeparator />

              <DropdownMenuItem
                disabled={!canCreate || isScanActive(row.status)}
                onClick={() => {
                  if (!canCreate || isScanActive(row.status)) return;
                  void rescan(row);
                }}
              >
                <RotateCcw />
                {t("scanAgain")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      ),
    },
  ];

  const actions: TableAction[] = [
    {
      key: "filters",
      label: activeCount > 0 ? `${t("filters.open")} (${activeCount})` : t("filters.open"),
      icon: ListFilter,
      variant: "outline",
      onClick: () => setFiltersOpen(true),
    },
    {
      key: "start",
      label: t("start"),
      icon: Play,
      variant: "default",
      disabled: !canCreate,
      onClick: () => setDrawerOpen(true),
    },
  ];

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

    return (
      <>
        <DataTable
          key={appliedKey}
          query={query}
          columns={columns}
          filters={[]}
          actions={actions}
          getRowId={(row) => row.id}
          onRowClick={(row) => setSelected(row.id)}
          emptyMessage={activeCount > 0 ? t("emptyFiltered") : t("empty")}
        />

        <ScanDrawer
          key={drawerOpen ? "open" : "closed"}
          open={drawerOpen}
          onOpenChange={setDrawerOpen}
          canSave={canCreate}
          onStarted={setSelected}
        />

        <ScanFilterDrawer
          open={filtersOpen}
          onOpenChange={setFiltersOpen}
          value={applied}
          onApply={setApplied}
        />

        <ScanDetailsDrawer
          scanId={selected}
          onOpenChange={(open) => {
            if (!open) setSelected(null);
          }}
        />
      </>
    );
  }

  return <Consolepage heading={t("heading")} subheading={t("subheading")} data={body()} />;
}

"use client";

import {
  CircleAlert,
  ListFilter,
  Loader2,
  MoreHorizontal,
  Pencil,
  PlugZap,
  Plus,
  Trash2,
} from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { ConfigurationDrawer } from "@/components/data-discovery/configuration-drawer";
import {
  ConfigurationFilterDrawer,
  EMPTY_CONFIGURATION_FILTERS,
  activeConfigurationFilterCount,
  toConfigurationFilterParams,
  type ConfigurationFilters,
} from "@/components/data-discovery/configuration-filter-drawer";
import { AccessDenied } from "@/components/dashboard/access-denied";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
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
import { CONFIGURATION_TYPE_META, supportedSourceTypes } from "@/lib/data-discovery";
import { DISCOVERY_CONFIGURATION_PRIVILEGES } from "@/lib/privileges";
import {
  useDeleteDiscoveryConfigurationMutation,
  useListDiscoveryConfigurationsQuery,
  useTestDiscoveryConfigurationMutation,
  type DiscoveryConfigurationListItem,
} from "@/store/api/data-discovery-configurations-api";

export default function DiscoveryConfigurationsPage() {
  const t = useTranslations("data-discovery.configurations");
  const shared = useTranslations("data-discovery");
  const common = useTranslations("common");
  const table = useTranslations("table");

  const {
    has,
    loading: privilegesLoading,
    isError: privilegesError,
    refetch: refetchPrivileges,
  } = usePrivileges();

  const canView = has(DISCOVERY_CONFIGURATION_PRIVILEGES.view);
  const canCreate = has(DISCOVERY_CONFIGURATION_PRIVILEGES.create);
  const canEdit = has(DISCOVERY_CONFIGURATION_PRIVILEGES.edit);
  const canDelete = has(DISCOVERY_CONFIGURATION_PRIVILEGES.delete);
  const canTest = has(DISCOVERY_CONFIGURATION_PRIVILEGES.test);

  const [filtersOpen, setFiltersOpen] = React.useState(false);
  const [applied, setApplied] = React.useState<ConfigurationFilters>(EMPTY_CONFIGURATION_FILTERS);
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<DiscoveryConfigurationListItem | null>(null);
  const [deleting, setDeleting] = React.useState<DiscoveryConfigurationListItem | null>(null);

  const [deleteConfiguration, { isLoading: deletePending }] =
    useDeleteDiscoveryConfigurationMutation();
  const [testConfiguration] = useTestDiscoveryConfigurationMutation();

  const appliedParams = React.useMemo(() => toConfigurationFilterParams(applied), [applied]);
  const appliedKey = React.useMemo(() => JSON.stringify(appliedParams), [appliedParams]);
  const activeCount = activeConfigurationFilterCount(applied);

  const query = React.useCallback(
    (args: DataTableQueryArgs) =>
      useListDiscoveryConfigurationsQuery({
        ...args,
        filters: { ...args.filters, ...appliedParams },
      }),
    [appliedParams],
  );

  const columns: ColumnConfig<DiscoveryConfigurationListItem>[] = [
    {
      key: "name",
      label: t("columns.name"),
      render: (value, row) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate font-medium text-foreground">{String(value)}</span>
          <span className="truncate text-xs text-muted-foreground">
            {shared(`configurationType.${row.configuration_type}`)}
          </span>
        </div>
      ),
    },
    {
      key: "configuration_type",
      label: t("columns.type"),
      render: (_value, row) => (
        <div className="flex min-w-0 items-center gap-2">
          <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-surface-container font-mono text-[0.6875rem]">
            {CONFIGURATION_TYPE_META[row.configuration_type].monogram}
          </span>
          <span
            className="block max-w-48 truncate text-xs text-muted-foreground"
            title={supportedSourceTypes(row.configuration_type)
              .map((source) => shared(`sourceType.${source}`))
              .join(", ")}
          >
            {supportedSourceTypes(row.configuration_type)
              .map((source) => shared(`sourceType.${source}`))
              .join(" · ")}
          </span>
        </div>
      ),
    },
    {
      key: "last_tested_at",
      label: t("columns.lastTested"),
      type: "datetime",
    },
    {
      key: "status",
      label: t("columns.status"),
      type: "status",
    },
    {
      key: "policy_count",
      label: t("columns.policies"),
      type: "number",
      align: "right",
    },
    {
      key: "updated_at",
      label: t("columns.updatedAt"),
      type: "datetime",
      sortable: true,
    },
    {
      key: "actions",
      label: "",
      align: "right",
      render: (_value, row) => {
        if (!canEdit && !canDelete && !canTest) return null;

        return (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button type="button" variant="ghost" size="icon">
                  <MoreHorizontal />
                </Button>
              }
            />

            <DropdownMenuContent align="end">
              <DropdownMenuItem
                disabled={!canEdit}
                onClick={() => {
                  if (!canEdit) return;
                  setEditing(row);
                  setDrawerOpen(true);
                }}
              >
                <Pencil />
                {common("edit")}
              </DropdownMenuItem>

              <DropdownMenuItem disabled={!canTest} onClick={() => runTest(row)}>
                <PlugZap />
                {t("test.start")}
              </DropdownMenuItem>

              <DropdownMenuSeparator />

              <DropdownMenuItem
                variant="destructive"
                disabled={!canDelete}
                onClick={() => {
                  if (!canDelete) return;
                  setDeleting(row);
                }}
              >
                <Trash2 />
                {common("delete")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];

  function runTest(row: DiscoveryConfigurationListItem) {
    if (!canTest) return;

    toast.promise(
      testConfiguration({ configuration_id: row.id, config: {}, secret: {} }).unwrap(),
      {
        loading: t("test.running", { name: row.name }),
        success: t("test.passed", { name: row.name }),
        error: (error) => apiErrorMessage(error, t("test.failed")),
      },
    );
  }

  const actions: TableAction[] = [
    {
      key: "filters",
      label: activeCount > 0 ? `${t("filters.open")} (${activeCount})` : t("filters.open"),
      icon: ListFilter,
      variant: "outline",
      onClick: () => setFiltersOpen(true),
    },
    {
      key: "add",
      label: t("add"),
      icon: Plus,
      variant: "default",
      disabled: !canCreate,
      onClick: () => {
        setEditing(null);
        setDrawerOpen(true);
      },
    },
  ];

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteConfiguration(deleting.id).unwrap();
      toast.success(t("deleted", { name: deleting.name }));
      setDeleting(null);
    } catch (error) {
      toast.error(apiErrorMessage(error, t("deleteFailed")));
    }
  }

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
          emptyMessage={activeCount > 0 ? t("emptyFiltered") : t("empty")}
        />

        <ConfigurationDrawer
          key={editing ? `edit-${editing.id}` : "create"}
          open={drawerOpen}
          onOpenChange={setDrawerOpen}
          configurationId={editing?.id ?? null}
          canSave={editing ? canEdit : canCreate}
        />

        <ConfigurationFilterDrawer
          open={filtersOpen}
          onOpenChange={setFiltersOpen}
          value={applied}
          onApply={setApplied}
        />

        <ConfirmDialog
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
          title={t("deleteTitle")}
          description={
            deleting
              ? deleting.policy_count > 0
                ? t("inUse", { count: deleting.policy_count })
                : t("deleteDescription", { name: deleting.name })
              : undefined
          }
          confirmLabel={common("delete")}
          destructive
          pending={deletePending || (deleting?.policy_count ?? 0) > 0}
          onConfirm={confirmDelete}
        />
      </>
    );
  }

  return <Consolepage heading={t("heading")} subheading={t("subheading")} data={body()} />;
}

"use client";

import {
  CircleAlert,
  ListFilter,
  Loader2,
  MoreHorizontal,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { PolicyDrawer } from "@/components/data-discovery/policy-drawer";
import {
  EMPTY_POLICY_FILTERS,
  PolicyFilterDrawer,
  activePolicyFilterCount,
  toPolicyFilterParams,
  type PolicyFilters,
} from "@/components/data-discovery/policy-filter-drawer";
import { AccessDenied } from "@/components/dashboard/access-denied";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import type {
  ColumnConfig,
  DataTableQueryArgs,
  TableAction,
} from "@/components/data-table/types";
import { Badge } from "@/components/ui/badge";
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
import { DISCOVERY_POLICY_PRIVILEGES } from "@/lib/privileges";
import {
  useDeleteDiscoveryPolicyMutation,
  useListDiscoveryPoliciesQuery,
  type DiscoveryPolicyListItem,
} from "@/store/api/data-discovery-policies-api";

export default function DiscoveryPoliciesPage() {
  const t = useTranslations("data-discovery.policies");
  const shared = useTranslations("data-discovery");
  const common = useTranslations("common");
  const table = useTranslations("table");

  const {
    has,
    loading: privilegesLoading,
    isError: privilegesError,
    refetch: refetchPrivileges,
  } = usePrivileges();

  const canView = has(DISCOVERY_POLICY_PRIVILEGES.view);
  const canCreate = has(DISCOVERY_POLICY_PRIVILEGES.create);
  const canEdit = has(DISCOVERY_POLICY_PRIVILEGES.edit);
  const canDelete = has(DISCOVERY_POLICY_PRIVILEGES.delete);

  const [filtersOpen, setFiltersOpen] = React.useState(false);
  const [applied, setApplied] = React.useState<PolicyFilters>(EMPTY_POLICY_FILTERS);
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<DiscoveryPolicyListItem | null>(null);
  const [deleting, setDeleting] = React.useState<DiscoveryPolicyListItem | null>(null);

  const [deletePolicy, { isLoading: deletePending }] = useDeleteDiscoveryPolicyMutation();

  const appliedParams = React.useMemo(() => toPolicyFilterParams(applied), [applied]);
  const appliedKey = React.useMemo(() => JSON.stringify(appliedParams), [appliedParams]);
  const activeCount = activePolicyFilterCount(applied);

  const query = React.useCallback(
    (args: DataTableQueryArgs) =>
      useListDiscoveryPoliciesQuery({
        ...args,
        filters: { ...args.filters, ...appliedParams },
      }),
    [appliedParams],
  );

  const columns: ColumnConfig<DiscoveryPolicyListItem>[] = [
    {
      key: "name",
      label: t("columns.name"),
      render: (value, row) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate font-medium text-foreground">{String(value)}</span>
          <span className="truncate text-xs text-muted-foreground">
            {shared(`sourceType.${row.source_type}`)}
          </span>
        </div>
      ),
    },
    {
      key: "configuration_name",
      label: t("columns.configuration"),
      render: (value, row) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate">{String(value) || "—"}</span>
          <span className="truncate text-xs text-muted-foreground">
            {shared(`configurationType.${row.configuration_type}`)}
          </span>
        </div>
      ),
    },
    {
      key: "target_count",
      label: t("columns.targets"),
      render: (value) => <Badge variant="secondary">{String(value)}</Badge>,
    },
    {
      key: "rule_count",
      label: t("columns.rules"),
      type: "number",
      align: "right",
    },
    {
      key: "status",
      label: t("columns.status"),
      type: "status",
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
        if (!canEdit && !canDelete) return null;

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
      await deletePolicy(deleting.id).unwrap();
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

        <PolicyDrawer
          key={editing ? `edit-${editing.id}` : "create"}
          open={drawerOpen}
          onOpenChange={setDrawerOpen}
          policyId={editing?.id ?? null}
          canSave={editing ? canEdit : canCreate}
        />

        <PolicyFilterDrawer
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
          description={deleting ? t("deleteDescription", { name: deleting.name }) : undefined}
          confirmLabel={common("delete")}
          destructive
          pending={deletePending}
          onConfirm={confirmDelete}
        />
      </>
    );
  }

  return <Consolepage heading={t("heading")} subheading={t("subheading")} data={body()} />;
}

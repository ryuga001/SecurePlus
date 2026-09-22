"use client";

import { CircleAlert, ListFilter, Loader2, MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { AlertDrawer } from "@/components/admin/alert-dialog";
import {
  AlertFilterDrawer,
  EMPTY_ALERT_FILTERS,
  activeAlertFilterCount,
  toAlertFilterParams,
  type AlertFilters,
} from "@/components/admin/alert-filter-drawer";
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
import { ALERT_PRIVILEGES } from "@/lib/privileges";
import {
  useDeleteAlertMutation,
  useListAlertsQuery,
  type AlertListItem,
} from "@/store/api/alerts-api";

export default function AlertsPage() {
  const t = useTranslations("email-alert");
  const common = useTranslations("common");
  const table = useTranslations("table");

  const {
    has,
    loading: privilegesLoading,
    isError: privilegesError,
    refetch: refetchPrivileges,
  } = usePrivileges();

  const canView = has(ALERT_PRIVILEGES.view);
  const canCreate = has(ALERT_PRIVILEGES.create);
  const canEdit = has(ALERT_PRIVILEGES.edit);
  const canDelete = has(ALERT_PRIVILEGES.delete);

  const [filtersOpen, setFiltersOpen] = React.useState(false);
  const [applied, setApplied] = React.useState<AlertFilters>(EMPTY_ALERT_FILTERS);
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<AlertListItem | null>(null);
  const [deleting, setDeleting] = React.useState<AlertListItem | null>(null);

  const [deleteAlert, { isLoading: deletePending }] = useDeleteAlertMutation();

  const appliedParams = React.useMemo(() => toAlertFilterParams(applied), [applied]);
  const appliedKey = React.useMemo(() => JSON.stringify(appliedParams), [appliedParams]);
  const activeCount = activeAlertFilterCount(applied);

  const query = React.useCallback(
    (args: DataTableQueryArgs) =>
      useListAlertsQuery({ ...args, filters: { ...args.filters, ...appliedParams } }),
    [appliedParams],
  );

  const columns: ColumnConfig<AlertListItem>[] = [
    {
      key: "name",
      label: t("columns.name"),
      render: (value) => (
        <span className="font-medium text-foreground">{String(value)}</span>
      ),
    },
    {
      key: "policies",
      label: t("columns.policies"),
      render: (_value, row) => {
        const names = row.policies ?? [];
        const count = row.policy_count;

        if (count === 0) {
          return <span className="text-sm text-muted-foreground">—</span>;
        }

        const title =
          names.map((policy) => policy.name).join(", ") +
          (count > names.length ? ", …" : "");

        if (count === 1 && names.length > 0) {
          return (
            <span className="block max-w-56 truncate" title={title}>
              {names[0].name}
            </span>
          );
        }

        if (count <= 3 && names.length > 0) {
          return (
            <span className="block max-w-64 truncate" title={title}>
              {t("policySummary.more", { name: names[0].name, count: count - 1 })}
            </span>
          );
        }

        return (
          <span className="text-sm whitespace-nowrap" title={title}>
            {t("policySummary.count", { count })}
          </span>
        );
      },
    },
    {
      key: "notification_type",
      label: t("columns.notificationType"),
      render: (value) => (
        <Badge variant="secondary">
          {value === "SMS" ? t("notificationType.sms") : t("notificationType.email")}
        </Badge>
      ),
    },
    {
      key: "schedule_type",
      label: t("columns.schedule"),
      render: (value) =>
        value === "CUSTOM" ? t("schedule.custom") : t("schedule.realTime"),
    },
    {
      key: "target_count",
      label: t("columns.recipients"),
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
                  setDialogOpen(true);
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
        setDialogOpen(true);
      },
    },
  ];

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteAlert(deleting.id).unwrap();
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
          <p className="text-muted-foreground">{t("privilegesFailed")}</p>
          <Button type="button" variant="outline" onClick={() => refetchPrivileges()}>
            {table("tryAgain")}
          </Button>
        </div>
      );
    }

    if (!canView) {
      return (
        <AccessDenied
          title={t("denied.title")}
          description={t("denied.description")}
        />
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
          emptyMessage={t("empty")}
        />

        <AlertDrawer
          key={editing ? `edit-${editing.id}` : "create"}
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          alertId={editing?.id ?? null}
          canSave={editing ? canEdit : canCreate}
        />

        <AlertFilterDrawer
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
              ? t("deleteDescription", {
                  name: deleting.name,
                  count: deleting.target_count,
                })
              : undefined
          }
          confirmLabel={common("delete")}
          destructive
          pending={deletePending}
          onConfirm={confirmDelete}
        />
      </>
    );
  }

  return (
    <Consolepage heading={t("heading")} subheading={t("subheading")} data={body()} />
  );
}

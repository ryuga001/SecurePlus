"use client";

import { Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import type {
  ColumnConfig,
  FilterConfig,
  RowAction,
  TableAction,
} from "@/components/data-table/types";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { EmailConfigurationDialog } from "@/components/email/configurations/configuration-dialog";
import { providerLabel } from "@/components/email/configurations/provider-meta";
import { apiErrorMessage } from "@/lib/api-error";
import {
  useDeleteEmailConfigurationMutation,
  useListEmailConfigurationsQuery,
  type EmailConfiguration,
} from "@/store/api/email-configurations-api";

export default function EmailConfigurationsPage() {
  const t = useTranslations("configurations");
  const common = useTranslations("common");

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-72",
    },
  ];

  const columns: ColumnConfig<EmailConfiguration>[] = [
    { key: "name", label: t("columns.name") },
    {
      key: "provider",
      label: t("columns.provider"),
      render: (value) => providerLabel(String(value)),
    },
    {
      key: "domain",
      label: t("columns.domain"),
      render: (value) => <span className="font-mono text-[0.8rem]">{String(value)}</span>,
    },
    { key: "created_at", label: t("columns.createdAt"), type: "date" },
    { key: "updated_at", label: t("columns.updatedAt"), type: "date" },
  ];

  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<EmailConfiguration | null>(null);
  const [deleting, setDeleting] = React.useState<EmailConfiguration | null>(null);

  const [deleteConfiguration, { isLoading: deletePending }] =
    useDeleteEmailConfigurationMutation();

  function openCreate() {
    setEditing(null);
    setDialogOpen(true);
  }

  function openEdit(row: EmailConfiguration) {
    setEditing(row);
    setDialogOpen(true);
  }

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

  const actions: TableAction[] = [
    {
      key: "add",
      label: t("add"),
      icon: Plus,
      variant: "default",
      onClick: openCreate,
    },
  ];

  const rowActions: RowAction<EmailConfiguration>[] = [
    { key: "edit", label: common("edit"), icon: Pencil, variant: "ghost", onClick: openEdit },
    {
      key: "delete",
      label: common("delete"),
      icon: Trash2,
      variant: "destructive",
      onClick: (row) => setDeleting(row),
    },
  ];

  return (
    <Consolepage
      heading={t("heading")}
      subheading={t("subheading")}
      data={
        <>
          <DataTable
            query={useListEmailConfigurationsQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage={t("empty")}
          />

          <EmailConfigurationDialog
            key={editing ? `edit-${editing.id}` : "create"}
            open={dialogOpen}
            onOpenChange={setDialogOpen}
            configuration={editing}
          />

          <ConfirmDialog
            open={deleting !== null}
            onOpenChange={(open) => {
              if (!open) setDeleting(null);
            }}
            title={t("deleteTitle")}
            description={
              deleting
                ? t("deleteDescription", { name: deleting.name, domain: deleting.domain })
                : undefined
            }
            confirmLabel={common("delete")}
            destructive
            pending={deletePending}
            onConfirm={confirmDelete}
          />
        </>
      }
    />
  );
}

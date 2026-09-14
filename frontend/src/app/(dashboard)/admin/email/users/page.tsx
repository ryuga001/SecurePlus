"use client";

import { Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { EmailUserDialog } from "@/components/admin/email-user-dialog";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import type {
  ColumnConfig,
  FilterConfig,
  RowAction,
  TableAction,
} from "@/components/data-table/types";
import { apiErrorMessage } from "@/lib/api-error";
import {
  useDeleteEmailUserMutation,
  useListEmailUsersQuery,
  type EmailUser,
} from "@/store/api/email-users-api";

export default function EmailUsersPage() {
  const t = useTranslations("emailUsers");
  const common = useTranslations("common");

  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<EmailUser | null>(null);
  const [deleting, setDeleting] = React.useState<EmailUser | null>(null);

  const [deleteUser, { isLoading: deletePending }] = useDeleteEmailUserMutation();

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-72",
    },
  ];

  const columns: ColumnConfig<EmailUser>[] = [
    {
      key: "name",
      label: t("columns.name"),
      accessor: (row) => `${row.first_name} ${row.last_name}`.trim(),
    },
    {
      key: "email",
      label: t("columns.email"),
      render: (value) => <span className="font-mono text-[0.8rem]">{String(value)}</span>,
    },
    { key: "created_at", label: t("columns.createdAt"), type: "date" },
    { key: "updated_at", label: t("columns.updatedAt"), type: "date" },
  ];

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteUser(deleting.id).unwrap();
      toast.success(t("deleted", { email: deleting.email }));
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
      onClick: () => {
        setEditing(null);
        setDialogOpen(true);
      },
    },
  ];

  const rowActions: RowAction<EmailUser>[] = [
    {
      key: "edit",
      label: common("edit"),
      icon: Pencil,
      variant: "ghost",
      onClick: (row) => {
        setEditing(row);
        setDialogOpen(true);
      },
    },
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
            query={useListEmailUsersQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage={t("empty")}
          />

          <EmailUserDialog
            key={editing ? `edit-${editing.id}` : "create"}
            open={dialogOpen}
            onOpenChange={setDialogOpen}
            user={editing}
          />

          <ConfirmDialog
            open={deleting !== null}
            onOpenChange={(open) => {
              if (!open) setDeleting(null);
            }}
            title={t("deleteTitle")}
            description={
              deleting ? t("deleteDescription", { email: deleting.email }) : undefined
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

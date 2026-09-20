"use client";

import { MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
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

  const [deleteUser, { isLoading: deletePending }] =
    useDeleteEmailUserMutation();

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
      render: (value) => (
        <span className="font-medium text-foreground">{String(value)}</span>
      ),
    },
    {
      key: "email",
      label: t("columns.email"),
      render: (value) => (
        <span className="font-mono text-[0.8rem]">{String(value)}</span>
      ),
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
      render: (_, row) => (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={common("actions")}
                onClick={(event) => event.stopPropagation()}
              >
                <MoreHorizontal />
              </Button>
            }
          />
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={() => {
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
              onClick={() => setDeleting(row)}
            >
              <Trash2 />
              {common("delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
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
              deleting
                ? t("deleteDescription", { email: deleting.email })
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
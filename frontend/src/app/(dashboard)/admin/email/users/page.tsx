"use client";

import { Pencil, Plus, Trash2 } from "lucide-react";
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

const filters: FilterConfig[] = [
  {
    key: "search",
    label: "Search",
    type: "text",
    placeholder: "Email or name",
    width: "w-72",
  },
];

const columns: ColumnConfig<EmailUser>[] = [
  {
    key: "name",
    label: "Name",
    accessor: (row) => `${row.first_name} ${row.last_name}`.trim(),
  },
  {
    key: "email",
    label: "Email",
    render: (value) => <span className="font-mono text-[0.8rem]">{String(value)}</span>,
  },
  { key: "created_at", label: "Created At", type: "date" },
  { key: "updated_at", label: "Updated At", type: "date" },
];

export default function EmailUsersPage() {
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<EmailUser | null>(null);
  const [deleting, setDeleting] = React.useState<EmailUser | null>(null);

  const [deleteUser, { isLoading: deletePending }] = useDeleteEmailUserMutation();

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteUser(deleting.id).unwrap();
      toast.success(`${deleting.email} deleted`);
      setDeleting(null);
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not delete this user"));
    }
  }

  const actions: TableAction[] = [
    {
      key: "add",
      label: "Add User",
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
      label: "Edit",
      icon: Pencil,
      variant: "ghost",
      onClick: (row) => {
        setEditing(row);
        setDialogOpen(true);
      },
    },
    {
      key: "delete",
      label: "Delete",
      icon: Trash2,
      variant: "destructive",
      onClick: (row) => setDeleting(row),
    },
  ];

  return (
    <Consolepage
      heading="Email users"
      subheading="The mailboxes this workspace protects."
      data={
        <>
          <DataTable
            query={useListEmailUsersQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage="No email users yet"
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
            title="Delete user"
            description={
              deleting
                ? `${deleting.email} will be removed from every group they belong to. This cannot be undone.`
                : undefined
            }
            confirmLabel="Delete"
            destructive
            pending={deletePending}
            onConfirm={confirmDelete}
          />
        </>
      }
    />
  );
}

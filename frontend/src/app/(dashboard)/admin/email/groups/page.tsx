"use client";

import { Pencil, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { GroupDialog } from "@/components/admin/group-dialog";
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
  useDeleteEmailGroupMutation,
  useListEmailGroupsQuery,
  type EmailGroup,
} from "@/store/api/email-groups-api";

const filters: FilterConfig[] = [
  { key: "search", label: "Search", type: "text", placeholder: "Group name", width: "w-72" },
];

const columns: ColumnConfig<EmailGroup>[] = [
  { key: "name", label: "Group Name" },
  { key: "member_count", label: "Members", type: "number", align: "right" },
  { key: "created_at", label: "Created At", type: "date" },
  { key: "updated_at", label: "Updated At", type: "date" },
];

export default function EmailGroupsPage() {
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<EmailGroup | null>(null);
  const [deleting, setDeleting] = React.useState<EmailGroup | null>(null);

  const [deleteGroup, { isLoading: deletePending }] = useDeleteEmailGroupMutation();

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteGroup(deleting.id).unwrap();
      toast.success(`${deleting.name} deleted`);
      setDeleting(null);
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not delete this group"));
    }
  }

  const actions: TableAction[] = [
    {
      key: "add",
      label: "Add Group",
      icon: Plus,
      variant: "default",
      onClick: () => {
        setEditing(null);
        setDialogOpen(true);
      },
    },
  ];

  const rowActions: RowAction<EmailGroup>[] = [
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
      heading="Email groups"
      subheading="Collections of users that policies target together."
      data={
        <>
          <DataTable
            query={useListEmailGroupsQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage="No groups yet"
          />

          <GroupDialog
            key={editing ? `edit-${editing.id}` : "create"}
            open={dialogOpen}
            onOpenChange={setDialogOpen}
            group={editing}
          />

          <ConfirmDialog
            open={deleting !== null}
            onOpenChange={(open) => {
              if (!open) setDeleting(null);
            }}
            title="Delete group"
            description={
              deleting
                ? `${deleting.name} will be detached from every policy that targets it, and its ${deleting.member_count} memberships will be removed. This cannot be undone.`
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

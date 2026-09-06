"use client";

import { Pencil, Plus, Trash2 } from "lucide-react";
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

const filters: FilterConfig[] = [
  {
    key: "search",
    label: "Search",
    type: "text",
    placeholder: "Configuration name",
    width: "w-72",
  },
];

const columns: ColumnConfig<EmailConfiguration>[] = [
  { key: "name", label: "Configuration Name" },
  { key: "provider", label: "Provider", render: (value) => providerLabel(String(value)) },
  {
    key: "domain",
    label: "Domain",
    render: (value) => <span className="font-mono text-[0.8rem]">{String(value)}</span>,
  },
  { key: "created_at", label: "Created At", type: "date" },
  { key: "updated_at", label: "Updated At", type: "date" },
];

export default function EmailConfigurationsPage() {
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
      toast.success(`${deleting.name} deleted`);
      setDeleting(null);
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not delete this configuration"));
    }
  }

  const actions: TableAction[] = [
    {
      key: "add",
      label: "Add Configuration",
      icon: Plus,
      variant: "default",
      onClick: openCreate,
    },
  ];

  const rowActions: RowAction<EmailConfiguration>[] = [
    { key: "edit", label: "Edit", icon: Pencil, variant: "ghost", onClick: openEdit },
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
      heading="Email configurations"
      subheading="Connected mailboxes, gateways and delivery settings."
      data={
        <>
          <DataTable
            query={useListEmailConfigurationsQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage="No email configurations yet"
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
            title="Delete configuration"
            description={
              deleting
                ? `${deleting.name} (${deleting.domain}) will stop sending and its credentials will be revoked. This cannot be undone.`
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

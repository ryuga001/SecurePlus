"use client";

import { Pencil, Plus, Power, Trash2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { PolicyDialog } from "@/components/admin/policy-dialog";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import { StatusBadge } from "@/components/data-table/status-badge";
import type {
  ColumnConfig,
  FilterConfig,
  RowAction,
  TableAction,
} from "@/components/data-table/types";
import { apiErrorMessage } from "@/lib/api-error";
import {
  useDeletePolicyMutation,
  useListPoliciesQuery,
  useSetPolicyStatusMutation,
  type PolicyListItem,
} from "@/store/api/policies-api";

const filters: FilterConfig[] = [
  { key: "search", label: "Search", type: "text", placeholder: "Policy name", width: "w-72" },
  {
    key: "active",
    label: "Status",
    type: "select",
    placeholder: "All",
    options: [
      { label: "Enabled", value: "true" },
      { label: "Disabled", value: "false" },
    ],
  },
];

const ACTION_LABELS: Record<string, string> = {
  AUDIT: "Audit",
  BLOCK: "Block",
  QUARANTINE: "Quarantine",
  REDACT: "Redact",
};

const RESTRICTION_LABELS: Record<string, string> = {
  NONE: "None",
  BLOCK: "Block list",
  ALLOW: "Allow list",
};

const columns: ColumnConfig<PolicyListItem>[] = [
  { key: "policy_name", label: "Policy Name" },
  { key: "action", label: "Action", render: (value) => ACTION_LABELS[String(value)] ?? value },
  {
    key: "active",
    label: "Status",
    render: (value) => <StatusBadge status={value ? "enabled" : "disabled"} />,
  },
  {
    key: "domain_restriction_mode",
    label: "Domains",
    render: (value) => RESTRICTION_LABELS[String(value)] ?? value,
  },
  {
    key: "attachment_restriction_mode",
    label: "Attachments",
    render: (value) => RESTRICTION_LABELS[String(value)] ?? value,
  },
  { key: "rule_count", label: "Rules", type: "number", align: "right" },
  { key: "group_count", label: "Groups", type: "number", align: "right" },
  { key: "updated_at", label: "Updated At", type: "date" },
];

export default function PoliciesPage() {
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editingId, setEditingId] = React.useState<number | null>(null);
  const [deleting, setDeleting] = React.useState<PolicyListItem | null>(null);

  const [setStatus] = useSetPolicyStatusMutation();
  const [deletePolicy, { isLoading: deletePending }] = useDeletePolicyMutation();

  async function toggleStatus(row: PolicyListItem) {
    try {
      await setStatus({ id: row.id, active: !row.active }).unwrap();
      toast.success(`${row.policy_name} ${row.active ? "disabled" : "enabled"}`);
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not change the policy status"));
    }
  }

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deletePolicy(deleting.id).unwrap();
      toast.success(`${deleting.policy_name} deleted`);
      setDeleting(null);
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not delete this policy"));
    }
  }

  const actions: TableAction[] = [
    {
      key: "add",
      label: "Add Policy",
      icon: Plus,
      variant: "default",
      onClick: () => {
        setEditingId(null);
        setDialogOpen(true);
      },
    },
  ];

  const rowActions: RowAction<PolicyListItem>[] = [
    {
      key: "status",
      label: "Toggle",
      icon: Power,
      variant: "ghost",
      onClick: toggleStatus,
    },
    {
      key: "edit",
      label: "Edit",
      icon: Pencil,
      variant: "ghost",
      onClick: (row) => {
        setEditingId(row.id);
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
      heading="Email policies"
      subheading="Which rules apply to which groups of users."
      data={
        <>
          <DataTable
            query={useListPoliciesQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage="No policies yet"
          />

          <PolicyDialog
            key={editingId ? `edit-${editingId}` : "create"}
            open={dialogOpen}
            onOpenChange={setDialogOpen}
            policyId={editingId}
          />

          <ConfirmDialog
            open={deleting !== null}
            onOpenChange={(open) => {
              if (!open) setDeleting(null);
            }}
            title="Delete policy"
            description={
              deleting
                ? `${deleting.policy_name} and its rule and group assignments will be removed. This cannot be undone.`
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

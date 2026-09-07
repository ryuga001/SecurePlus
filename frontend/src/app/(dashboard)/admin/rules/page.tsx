"use client";

import { Pencil, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { RuleDialog } from "@/components/admin/rule-dialog";
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
import { useDeleteRuleMutation, useListRulesQuery, type Rule } from "@/store/api/rules-api";

const filters: FilterConfig[] = [
  { key: "search", label: "Search", type: "text", placeholder: "Rule name", width: "w-72" },
];

const columns: ColumnConfig<Rule>[] = [
  { key: "rule_name", label: "Rule Name" },
  {
    key: "type",
    label: "Type",
    render: (value) => (value === "REGEX" ? "Regular expression" : "Keyword"),
  },
  {
    key: "value",
    label: "Matches",
    render: (value) => (
      <span className="block max-w-md truncate font-mono text-[0.8rem]">{String(value)}</span>
    ),
  },
  { key: "created_at", label: "Created At", type: "date" },
  { key: "updated_at", label: "Updated At", type: "date" },
];

export default function RulesPage() {
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<Rule | null>(null);
  const [deleting, setDeleting] = React.useState<Rule | null>(null);

  const [deleteRule, { isLoading: deletePending }] = useDeleteRuleMutation();

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteRule(deleting.id).unwrap();
      toast.success(`${deleting.rule_name} deleted`);
      setDeleting(null);
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not delete this rule"));
    }
  }

  const actions: TableAction[] = [
    {
      key: "add",
      label: "Add Rule",
      icon: Plus,
      variant: "default",
      onClick: () => {
        setEditing(null);
        setDialogOpen(true);
      },
    },
  ];

  const rowActions: RowAction<Rule>[] = [
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
      heading="Rules"
      subheading="Keyword and regular-expression matchers that policies apply to mail."
      data={
        <>
          <DataTable
            query={useListRulesQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage="No rules yet"
          />

          <RuleDialog
            key={editing ? `edit-${editing.id}` : "create"}
            open={dialogOpen}
            onOpenChange={setDialogOpen}
            rule={editing}
          />

          <ConfirmDialog
            open={deleting !== null}
            onOpenChange={(open) => {
              if (!open) setDeleting(null);
            }}
            title="Delete rule"
            description={
              deleting
                ? `${deleting.rule_name} will be removed from every policy that uses it. This cannot be undone.`
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

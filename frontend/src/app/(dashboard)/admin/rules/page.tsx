"use client";

import { Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
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

export default function RulesPage() {
  const t = useTranslations("rules");
  const common = useTranslations("common");

  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<Rule | null>(null);
  const [deleting, setDeleting] = React.useState<Rule | null>(null);

  const [deleteRule, { isLoading: deletePending }] = useDeleteRuleMutation();

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-72",
    },
  ];

  const columns: ColumnConfig<Rule>[] = [
    { key: "rule_name", label: t("columns.name") },
    {
      key: "type",
      label: t("columns.type"),
      render: (value) => (value === "REGEX" ? t("type.regex") : t("type.keyword")),
    },
    {
      key: "value",
      label: t("columns.value"),
      render: (value) => (
        <span className="block max-w-md truncate font-mono text-[0.8rem]">{String(value)}</span>
      ),
    },
    { key: "created_at", label: t("columns.createdAt"), type: "date" },
    { key: "updated_at", label: t("columns.updatedAt"), type: "date" },
  ];

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteRule(deleting.id).unwrap();
      toast.success(t("deleted", { name: deleting.rule_name }));
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

  const rowActions: RowAction<Rule>[] = [
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
            query={useListRulesQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage={t("empty")}
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
            title={t("deleteTitle")}
            description={
              deleting ? t("deleteDescription", { name: deleting.rule_name }) : undefined
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

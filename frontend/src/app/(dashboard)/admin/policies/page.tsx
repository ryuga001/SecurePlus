"use client";

import { Pencil, Plus, Power, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
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

export default function PoliciesPage() {
  const t = useTranslations("policies");
  const status = useTranslations("status");
  const common = useTranslations("common");

  const restrictionLabel = (value: unknown) => {
    const key = String(value).toLowerCase();

    return t.has(`restriction.${key}`) ? t(`restriction.${key}`) : String(value);
  };

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-72",
    },
    {
      key: "active",
      label: t("statusFilter"),
      type: "select",
      placeholder: common("none"),
      options: [
        { label: status("enabled"), value: "true" },
        { label: status("disabled"), value: "false" },
      ],
    },
  ];

  const columns: ColumnConfig<PolicyListItem>[] = [
    { key: "policy_name", label: t("columns.name") },
    {
      key: "action",
      label: t("columns.action"),
      render: (value) => {
        const key = String(value).toLowerCase();

        return status.has(key) ? status(key) : String(value);
      },
    },
    {
      key: "active",
      label: t("columns.status"),
      render: (value) => <StatusBadge status={value ? "enabled" : "disabled"} />,
    },
    {
      key: "domain_restriction_mode",
      label: t("columns.domains"),
      render: restrictionLabel,
    },
    {
      key: "attachment_restriction_mode",
      label: t("columns.attachments"),
      render: restrictionLabel,
    },
    { key: "rule_count", label: t("columns.rules"), type: "number", align: "right" },
    { key: "group_count", label: t("columns.groups"), type: "number", align: "right" },
    { key: "updated_at", label: t("columns.updatedAt"), type: "date" },
  ];

  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editingId, setEditingId] = React.useState<number | null>(null);
  const [deleting, setDeleting] = React.useState<PolicyListItem | null>(null);

  const [setStatus] = useSetPolicyStatusMutation();
  const [deletePolicy, { isLoading: deletePending }] = useDeletePolicyMutation();

  async function toggleStatus(row: PolicyListItem) {
    try {
      await setStatus({ id: row.id, active: !row.active }).unwrap();
      toast.success(
        row.active
          ? t("disabled", { name: row.policy_name })
          : t("enabled", { name: row.policy_name })
      );
    } catch (error) {
      toast.error(apiErrorMessage(error, t("statusFailed")));
    }
  }

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deletePolicy(deleting.id).unwrap();
      toast.success(t("deleted", { name: deleting.policy_name }));
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
        setEditingId(null);
        setDialogOpen(true);
      },
    },
  ];

  const rowActions: RowAction<PolicyListItem>[] = [
    {
      key: "status",
      label: common("toggle"),
      icon: Power,
      variant: "ghost",
      onClick: toggleStatus,
    },
    {
      key: "edit",
      label: common("edit"),
      icon: Pencil,
      variant: "ghost",
      onClick: (row) => {
        setEditingId(row.id);
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
            query={useListPoliciesQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            rowActions={rowActions}
            getRowId={(row) => row.id}
            emptyMessage={t("empty")}
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
            title={t("deleteTitle")}
            description={
              deleting ? t("deleteDescription", { name: deleting.policy_name }) : undefined
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

"use client";

import { MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { PolicyDialog } from "@/components/admin/policy-dialog";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import type {
  ColumnConfig,
  FilterConfig,
  TableAction,
} from "@/components/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Switch } from "@/components/ui/switch";
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

  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editingId, setEditingId] = React.useState<number | null>(null);
  const [deleting, setDeleting] = React.useState<PolicyListItem | null>(null);

  const [setStatus] = useSetPolicyStatusMutation();
  const [deletePolicy, { isLoading: deletePending }] =
    useDeletePolicyMutation();

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

  async function toggleStatus(row: PolicyListItem, active: boolean) {
    try {
      await setStatus({ id: row.id, active }).unwrap();

      toast.success(
        active
          ? t("enabled", { name: row.policy_name })
          : t("disabled", { name: row.policy_name }),
      );
    } catch (error) {
      toast.error(apiErrorMessage(error, t("statusFailed")));
    }
  }

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deletePolicy(deleting.id).unwrap();

      toast.success(
        t("deleted", {
          name: deleting.policy_name,
        }),
      );

      setDeleting(null);
    } catch (error) {
      toast.error(apiErrorMessage(error, t("deleteFailed")));
    }
  }

  const columns: ColumnConfig<PolicyListItem>[] = [
    {
      key: "policy_name",
      label: t("columns.name"),
      render: (value) => (
        <span className="font-medium text-foreground">
          {String(value)}
        </span>
      ),
    },
    {
      key: "action",
      label: t("columns.action"),
      render: (value) => {
        const key = String(value).toLowerCase();

        return status.has(key) ? (
          <Badge variant="secondary">{status(key)}</Badge>
        ) : (
          <Badge variant="secondary">{String(value)}</Badge>
        );
      },
    },
    {
      key: "active",
      label: t("columns.status"),
      render: (_value, row) => (
        <Switch
          checked={row.active}
          onCheckedChange={(checked) => toggleStatus(row, checked)}
          aria-label={
            row.active
              ? t("disabled", { name: row.policy_name })
              : t("enabled", { name: row.policy_name })
          }
        />
      ),
    },
    {
      key: "groups",
      label: t("columns.groups"),
      render: (_value, row) => {
        const groups = row.groups ?? [];
        const visible = groups.slice(0, 3);
        const remaining = Math.max(
          row.group_count - visible.length,
          0,
        );

        if (visible.length === 0 && row.group_count === 0) {
          return (
            <span className="text-sm text-muted-foreground">
              —
            </span>
          );
        }

        return (
          <div className="flex min-w-0 flex-wrap items-center gap-1.5">
            {visible.map((group) => (
              <Badge
                key={group.id}
                variant="secondary"
                className="max-w-32 truncate font-normal"
                title={group.name}
              >
                {group.name}
              </Badge>
            ))}

            {remaining > 0 ? (
              <Badge variant="outline" className="font-medium">
                +{remaining}
              </Badge>
            ) : null}
          </div>
        );
      },
    },
    {
      key: "actions",
      label: "",
      align: "right",
      render: (_value, row) => (
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
                setEditingId(row.id);
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
              deleting
                ? t("deleteDescription", {
                    name: deleting.policy_name,
                  })
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
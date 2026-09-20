"use client";

import { MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import type {
  ColumnConfig,
  FilterConfig,
  TableAction,
} from "@/components/data-table/types";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { EmailConfigurationDrawer } from "@/components/email/configurations/configuration-drawer";
import { providerLabel } from "@/components/email/configurations/provider-meta";
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
  useDeleteEmailConfigurationMutation,
  useListEmailConfigurationsQuery,
  type EmailConfiguration,
} from "@/store/api/email-configurations-api";

export default function EmailConfigurationsPage() {
  const t = useTranslations("configurations");
  const common = useTranslations("common");

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-72",
    },
  ];

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
      toast.success(t("deleted", { name: deleting.name }));
      setDeleting(null);
    } catch (error) {
      toast.error(apiErrorMessage(error, t("deleteFailed")));
    }
  }

  const columns: ColumnConfig<EmailConfiguration>[] = [
    { key: "name", label: t("columns.name") },
    {
      key: "provider",
      label: t("columns.provider"),
      render: (value) => providerLabel(String(value)),
    },
    {
      key: "domain",
      label: t("columns.domain"),
      render: (value) => (
        <span className="font-mono text-[0.8rem]">{String(value)}</span>
      ),
    },
    { key: "updated_at", label: t("columns.updatedAt"), type: "date" },
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
            <DropdownMenuItem onClick={() => openEdit(row)}>
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
      onClick: openCreate,
    },
  ];

  return (
    <Consolepage
      heading={t("heading")}
      subheading={t("subheading")}
      data={
        <>
          <DataTable
            query={useListEmailConfigurationsQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            getRowId={(row) => row.id}
            emptyMessage={t("empty")}
          />

          <EmailConfigurationDrawer
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
            title={t("deleteTitle")}
            description={
              deleting
                ? t("deleteDescription", { name: deleting.name, domain: deleting.domain })
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
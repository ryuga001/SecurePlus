"use client";

import { MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { GroupDrawer } from "@/components/admin/group-dialog";
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
  useDeleteEmailGroupMutation,
  useListEmailGroupsQuery,
  type EmailGroup,
} from "@/store/api/email-groups-api";

export default function EmailGroupsPage() {
  const t = useTranslations("groups");
  const common = useTranslations("common");

  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<EmailGroup | null>(null);
  const [deleting, setDeleting] = React.useState<EmailGroup | null>(null);

  const [deleteGroup, { isLoading: deletePending }] =
    useDeleteEmailGroupMutation();

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-72",
    },
  ];

  const columns: ColumnConfig<EmailGroup>[] = [
    {
      key: "name",
      label: t("columns.name"),
      render: (value) => (
        <span className="font-medium text-foreground">
          {String(value)}
        </span>
      ),
    },
    {
      key: "member_count",
      label: t("columns.members"),
      render: (_value, row) => {
        const members = row.members ?? [];
        const visible = members.slice(0, 3);
        const remaining = Math.max(row.member_count - visible.length, 0);

        if (visible.length === 0 && row.member_count === 0) {
          return (
            <span className="text-sm text-muted-foreground">—</span>
          );
        }

        return (
          <div className="flex min-w-0 flex-wrap items-center gap-1.5">
            {visible.map((member) => (
              <span
                key={member.id}
                title={member.email}
                className="max-w-32 truncate rounded-md bg-muted px-2 py-1 text-xs font-medium"
              >
                {member.name || member.email}
              </span>
            ))}

            {remaining > 0 ? (
              <span className="rounded-md border px-2 py-1 text-xs font-medium">
                +{remaining}
              </span>
            ) : null}
          </div>
        );
      },
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
      render: (_value, row) => (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon"
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
      await deleteGroup(deleting.id).unwrap();
      toast.success(t("deleted", { name: deleting.name }));
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
            query={useListEmailGroupsQuery}
            columns={columns}
            filters={filters}
            actions={actions}
            getRowId={(row) => row.id}
            emptyMessage={t("empty")}
          />

          <GroupDrawer
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
            title={t("deleteTitle")}
            description={
              deleting
                ? t("deleteDescription", {
                    name: deleting.name,
                    count: deleting.member_count,
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
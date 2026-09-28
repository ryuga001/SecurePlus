"use client";

import { Eye, MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { RoleDrawer } from "@/components/console/role-drawer";
import { AccessDenied } from "@/components/dashboard/access-denied";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import { NameChips } from "@/components/data-table/name-chips";
import type { ColumnConfig, FilterConfig, TableAction } from "@/components/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useIsAdmin } from "@/hooks/use-is-admin";
import { usePrivilegeLabels } from "@/hooks/use-privilege-labels";
import { apiErrorCode, apiErrorMessage } from "@/lib/api-error";
import {
  useDeleteRoleMutation,
  useListRolesQuery,
  type RoleListItem,
} from "@/store/api/roles-api";

export default function UserRolesPage() {
  const t = useTranslations("console.roles");
  const shared = useTranslations("console");
  const common = useTranslations("common");
  const labels = usePrivilegeLabels();
  const isAdmin = useIsAdmin();

  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [selected, setSelected] = React.useState<RoleListItem | null>(null);
  const [deleting, setDeleting] = React.useState<RoleListItem | null>(null);

  const [deleteRole, { isLoading: deletePending }] = useDeleteRoleMutation();

  function open(role: RoleListItem | null) {
    setSelected(role);
    setDrawerOpen(true);
  }

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-64",
    },
  ];

  const columns: ColumnConfig<RoleListItem>[] = [
    {
      key: "name",
      label: t("columns.role"),
      render: (value, row) => (
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="flex items-center gap-2">
            <span className="truncate font-medium text-foreground">{String(value)}</span>
            {row.system ? (
              <Badge variant="info" size="sm">
                {t("system")}
              </Badge>
            ) : null}
          </span>
          {row.description ? (
            <span className="max-w-72 truncate text-xs text-muted-foreground">{row.description}</span>
          ) : null}
        </div>
      ),
    },
    {
      key: "privileges",
      label: t("columns.privileges"),
      render: (_value, row) => (
        <NameChips
          items={row.privileges.map((name) => ({ key: name, label: labels.privilege(name), title: name }))}
          total={row.privilege_count}
        />
      ),
    },
    {
      key: "users",
      label: t("columns.users"),
      render: (_value, row) => (
        <NameChips
          items={row.users.map((user) => ({ key: user.id, label: user.name || user.email, title: user.email }))}
          total={row.user_count}
        />
      ),
    },
    {
      key: "updated_at",
      label: t("columns.updated"),
      type: "date",
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
            {row.system ? (
              <DropdownMenuItem onClick={() => open(row)}>
                <Eye />
                {t("view")}
              </DropdownMenuItem>
            ) : (
              <>
                <DropdownMenuItem onClick={() => open(row)}>
                  <Pencil />
                  {common("edit")}
                </DropdownMenuItem>

                <DropdownMenuSeparator />

                <DropdownMenuItem
                  variant="destructive"
                  disabled={row.user_count > 0}
                  title={row.user_count > 0 ? t("deleteBlocked") : undefined}
                  onClick={() => setDeleting(row)}
                >
                  <Trash2 />
                  {common("delete")}
                </DropdownMenuItem>
              </>
            )}
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
      onClick: () => open(null),
    },
  ];

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteRole(deleting.id).unwrap();
      toast.success(t("deleted", { name: deleting.name }));
      setDeleting(null);
    } catch (error) {
      toast.error(
        apiErrorCode(error) === "role_in_use" ? t("deleteBlocked") : apiErrorMessage(error, t("deleteFailed")),
      );
    }
  }

  function body() {
    if (!isAdmin) {
      return <AccessDenied title={shared("denied.title")} description={shared("denied.description")} />;
    }

    return (
      <>
        <DataTable
          query={useListRolesQuery}
          columns={columns}
          filters={filters}
          actions={actions}
          getRowId={(row) => row.id}
          onRowClick={(row) => open(row)}
          emptyMessage={t("empty")}
        />

        <RoleDrawer
          key={selected ? `role-${selected.id}` : "create"}
          open={drawerOpen}
          onOpenChange={setDrawerOpen}
          roleId={selected?.id ?? null}
          system={selected?.system ?? false}
        />

        <ConfirmDialog
          open={deleting !== null}
          onOpenChange={(next) => {
            if (!next) setDeleting(null);
          }}
          title={t("deleteTitle")}
          description={deleting ? t("deleteDescription", { name: deleting.name }) : undefined}
          confirmLabel={common("delete")}
          destructive
          pending={deletePending}
          onConfirm={confirmDelete}
        />
      </>
    );
  }

  return <Consolepage heading={t("heading")} subheading={t("subheading")} data={body()} />;
}

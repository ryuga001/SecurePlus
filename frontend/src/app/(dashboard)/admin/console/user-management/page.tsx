"use client";

import { MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { useAuth } from "@/components/auth-provider";
import { UserDrawer } from "@/components/console/user-drawer";
import { AccessDenied } from "@/components/dashboard/access-denied";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
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
import { apiErrorCode, apiErrorMessage } from "@/lib/api-error";
import { useListRoleOptionsQuery } from "@/store/api/roles-api";
import { useDeleteUserMutation, useListUsersQuery, type User } from "@/store/api/users-api";

function fullName(user: User) {
  return `${user.first_name} ${user.last_name}`.trim();
}

export default function UserManagementPage() {
  const t = useTranslations("console.users");
  const shared = useTranslations("console");
  const common = useTranslations("common");
  const { identity } = useAuth();
  const isAdmin = useIsAdmin();

  const { data: roleOptions } = useListRoleOptionsQuery(undefined, { skip: !isAdmin });

  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<User | null>(null);
  const [deleting, setDeleting] = React.useState<User | null>(null);

  const [deleteUser, { isLoading: deletePending }] = useDeleteUserMutation();

  const currentUserId = identity?.user.id;

  function open(user: User | null) {
    setEditing(user);
    setDrawerOpen(true);
  }

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-72",
    },
    {
      key: "role_id",
      label: t("roleFilter"),
      type: "select",
      placeholder: t("allRoles"),
      options: (roleOptions ?? []).map((role) => ({ value: String(role.id), label: role.name })),
      width: "w-48",
    },
  ];

  const columns: ColumnConfig<User>[] = [
    {
      key: "name",
      label: t("columns.name"),
      accessor: fullName,
      render: (value, row) => (
        <span className="flex items-center gap-2">
          <span className="truncate font-medium text-foreground">{String(value)}</span>
          {row.id === currentUserId ? (
            <Badge variant="neutral" size="sm">
              {t("you")}
            </Badge>
          ) : null}
        </span>
      ),
    },
    { key: "email", label: t("columns.email") },
    {
      key: "role",
      label: t("columns.role"),
      render: (_value, row) =>
        row.role ? (
          <Badge variant={row.role.system ? "info" : "secondary"}>{row.role.name}</Badge>
        ) : (
          <span className="text-sm text-muted-foreground">—</span>
        ),
    },
    { key: "created_at", label: t("columns.created"), type: "date" },
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
            <DropdownMenuItem onClick={() => open(row)}>
              <Pencil />
              {common("edit")}
            </DropdownMenuItem>

            <DropdownMenuSeparator />

            <DropdownMenuItem
              variant="destructive"
              disabled={row.id === currentUserId}
              title={row.id === currentUserId ? t("deleteSelf") : undefined}
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
      onClick: () => open(null),
    },
  ];

  async function confirmDelete() {
    if (!deleting) return;

    try {
      await deleteUser(deleting.id).unwrap();
      toast.success(t("deleted", { email: deleting.email }));
      setDeleting(null);
    } catch (error) {
      const code = apiErrorCode(error);

      toast.error(
        code === "last_administrator"
          ? t("lastAdministrator")
          : code === "self_modification"
            ? t("deleteSelf")
            : apiErrorMessage(error, t("deleteFailed")),
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
          query={useListUsersQuery}
          columns={columns}
          filters={filters}
          actions={actions}
          getRowId={(row) => row.id}
          onRowClick={(row) => open(row)}
          emptyMessage={t("empty")}
        />

        <UserDrawer
          key={editing ? `user-${editing.id}` : "create"}
          open={drawerOpen}
          onOpenChange={setDrawerOpen}
          user={editing}
          self={editing !== null && editing.id === currentUserId}
        />

        <ConfirmDialog
          open={deleting !== null}
          onOpenChange={(next) => {
            if (!next) setDeleting(null);
          }}
          title={t("deleteTitle")}
          description={deleting ? t("deleteDescription", { email: deleting.email }) : undefined}
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

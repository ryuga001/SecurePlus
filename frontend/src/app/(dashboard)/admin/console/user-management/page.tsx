"use client";

import { Download, Plus, RefreshCw, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import type { ColumnConfig, FilterConfig, RowAction, TableAction } from "@/components/data-table/types";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { useListUsersQuery, type User } from "@/store/api/users-api";

export default function UserManagementPage() {
  const t = useTranslations("console.users");
  const status = useTranslations("status");
  const common = useTranslations("common");

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-64",
    },
    {
      key: "status",
      label: t("columns.status"),
      type: "select",
      options: [
        { label: status("active"), value: "active" },
        { label: status("invited"), value: "invited" },
        { label: status("disabled"), value: "disabled" },
      ],
    },
    {
      key: "role",
      label: t("columns.role"),
      type: "select",
      options: [
        { label: t("roles.admin"), value: "admin" },
        { label: t("roles.analyst"), value: "analyst" },
        { label: t("roles.auditor"), value: "auditor" },
      ],
    },
    { key: "createdAt", label: t("createdFilter"), type: "dateRange", width: "w-80" },
    { key: "onlyMfa", label: t("mfaFilter"), type: "checkbox", placeholder: t("mfaOnly") },
  ];

  const columns: ColumnConfig<User>[] = [
    {
      key: "name",
      label: t("columns.name"),
      sortable: true,
      accessor: (row) => `${row.first_name} ${row.last_name}`.trim(),
    },
    { key: "email", label: t("columns.email"), sortable: true },
    { key: "role", label: t("columns.role") },
    { key: "status", label: t("columns.status"), type: "status" },
    { key: "last_login_at", label: t("columns.lastLogin"), type: "datetime", sortable: true },
    { key: "created_at", label: t("columns.created"), type: "date", sortable: true, align: "right" },
  ];

  const actions: TableAction[] = [
    {
      key: "add",
      label: t("add"),
      icon: Plus,
      variant: "default",
      onClick: () => toast.info(t("add")),
    },
    {
      key: "export",
      label: common("export"),
      icon: Download,
      onClick: () => toast.info(t("exportStarted")),
    },
    {
      key: "refresh",
      label: common("refresh"),
      icon: RefreshCw,
      onClick: () => toast.success(common("refreshed")),
    },
  ];

  const rowActions: RowAction<User>[] = [
    {
      key: "delete",
      label: common("delete"),
      icon: Trash2,
      variant: "destructive",
      onClick: (row) => toast.warning(t("deletePrompt", { email: row.email })),
    },
  ];

  return (
    <Consolepage
      heading={t("heading")}
      subheading={t("subheading")}
      data={
        <DataTable
          query={useListUsersQuery}
          columns={columns}
          filters={filters}
          actions={actions}
          rowActions={rowActions}
          getRowId={(row) => row.id}
          emptyMessage={t("empty")}
        />
      }
    />
  );
}

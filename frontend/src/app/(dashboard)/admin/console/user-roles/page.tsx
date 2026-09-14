"use client";

import { Plus, RefreshCw } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import type { ColumnConfig, FilterConfig, TableAction } from "@/components/data-table/types";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { useListUserRolesQuery, type UserRole } from "@/store/api/users-api";

export default function UserRolesPage() {
  const t = useTranslations("console.roles");
  const common = useTranslations("common");

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchPlaceholder"),
      width: "w-64",
    },
  ];

  const columns: ColumnConfig<UserRole>[] = [
    { key: "name", label: t("columns.role"), sortable: true },
    { key: "description", label: t("columns.description") },
    { key: "users_count", label: t("columns.users"), type: "number", align: "right" },
    { key: "privileges_count", label: t("columns.privileges"), type: "number", align: "right" },
    { key: "updated_at", label: t("columns.updated"), type: "date", sortable: true, align: "right" },
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
      key: "refresh",
      label: common("refresh"),
      icon: RefreshCw,
      onClick: () => toast.success(common("refreshed")),
    },
  ];

  return (
    <Consolepage
      heading={t("heading")}
      subheading={t("subheading")}
      data={
        <DataTable
          query={useListUserRolesQuery}
          columns={columns}
          filters={filters}
          actions={actions}
          getRowId={(row) => row.id}
          emptyMessage={t("empty")}
        />
      }
    />
  );
}

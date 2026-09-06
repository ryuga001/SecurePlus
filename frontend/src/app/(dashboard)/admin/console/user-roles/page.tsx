"use client";

import { Plus, RefreshCw } from "lucide-react";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import type { ColumnConfig, FilterConfig, TableAction } from "@/components/data-table/types";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { useListUserRolesQuery, type UserRole } from "@/store/api/users-api";

const filters: FilterConfig[] = [
  { key: "search", label: "Search", type: "text", placeholder: "Role name", width: "w-64" },
];

const columns: ColumnConfig<UserRole>[] = [
  { key: "name", label: "Role", sortable: true },
  { key: "description", label: "Description" },
  { key: "users_count", label: "Users", type: "number", align: "right" },
  { key: "privileges_count", label: "Privileges", type: "number", align: "right" },
  { key: "updated_at", label: "Updated", type: "date", sortable: true, align: "right" },
];

export default function UserRolesPage() {
  const actions: TableAction[] = [
    {
      key: "add",
      label: "Add role",
      icon: Plus,
      variant: "default",
      onClick: () => toast.info("Add role"),
    },
    { key: "refresh", label: "Refresh", icon: RefreshCw, onClick: () => toast.success("Refreshed") },
  ];

  return (
    <Consolepage
      heading="User Roles"
      subheading="Roles and the privileges attached to them."
      data={
        <DataTable
          query={useListUserRolesQuery}
          columns={columns}
          filters={filters}
          actions={actions}
          getRowId={(row) => row.id}
          emptyMessage="No roles defined yet"
        />
      }
    />
  );
}

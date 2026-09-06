"use client";

import { Download, Plus, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import type { ColumnConfig, FilterConfig, RowAction, TableAction } from "@/components/data-table/types";
import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { useListUsersQuery, type User } from "@/store/api/users-api";

const filters: FilterConfig[] = [
  { key: "search", label: "Search", type: "text", placeholder: "Name or email", width: "w-64" },
  {
    key: "status",
    label: "Status",
    type: "select",
    options: [
      { label: "Active", value: "active" },
      { label: "Invited", value: "invited" },
      { label: "Disabled", value: "disabled" },
    ],
  },
  {
    key: "role",
    label: "Role",
    type: "select",
    options: [
      { label: "Administrator", value: "admin" },
      { label: "Analyst", value: "analyst" },
      { label: "Auditor", value: "auditor" },
    ],
  },
  { key: "createdAt", label: "Created", type: "dateRange", width: "w-80" },
  { key: "onlyMfa", label: "MFA", type: "checkbox", placeholder: "MFA enabled only" },
];

const columns: ColumnConfig<User>[] = [
  {
    key: "name",
    label: "Name",
    sortable: true,
    accessor: (row) => `${row.first_name} ${row.last_name}`.trim(),
  },
  { key: "email", label: "Email", sortable: true },
  { key: "role", label: "Role" },
  { key: "status", label: "Status", type: "status" },
  { key: "last_login_at", label: "Last login", type: "datetime", sortable: true },
  { key: "created_at", label: "Created", type: "date", sortable: true, align: "right" },
];

export default function UserManagementPage() {
  const actions: TableAction[] = [
    {
      key: "add",
      label: "Add user",
      icon: Plus,
      variant: "default",
      onClick: () => toast.info("Add user"),
    },
    { key: "export", label: "Export", icon: Download, onClick: () => toast.info("Export started") },
    { key: "refresh", label: "Refresh", icon: RefreshCw, onClick: () => toast.success("Refreshed") },
  ];

  const rowActions: RowAction<User>[] = [
    {
      key: "delete",
      label: "Delete",
      icon: Trash2,
      variant: "destructive",
      onClick: (row) => toast.warning(`Delete ${row.email}`),
    },
  ];

  return (
    <Consolepage
      heading="User Management"
      subheading="People with access to this console."
      data={
        <DataTable
          query={useListUsersQuery}
          columns={columns}
          filters={filters}
          actions={actions}
          rowActions={rowActions}
          getRowId={(row) => row.id}
          emptyMessage="No users match these filters"
        />
      }
    />
  );
}

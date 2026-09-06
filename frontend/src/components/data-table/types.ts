import type { LucideIcon } from "lucide-react";
import type * as React from "react";

export type FilterType = "text" | "select" | "date" | "dateRange" | "checkbox";

export type FilterOption = { label: string; value: string };

export type FilterConfig = {
  key: string;
  label: string;
  type: FilterType;
  placeholder?: string;
  options?: FilterOption[];
  defaultValue?: FilterValue;
  width?: string;
};

export type DateRangeValue = { from?: string; to?: string };

export type FilterValue = string | boolean | DateRangeValue | undefined;

export type FilterState = Record<string, FilterValue>;

export type ColumnConfig<Row> = {
  key: string;
  label: string;
  type?: "text" | "number" | "date" | "datetime" | "status" | "boolean";
  align?: "left" | "right" | "center";
  width?: string;
  sortable?: boolean;
  accessor?: (row: Row) => unknown;
  render?: (value: unknown, row: Row, index: number) => React.ReactNode;
};

export type RowAction<Row> = {
  key: string;
  label: string;
  icon?: LucideIcon;
  variant?: "default" | "outline" | "secondary" | "ghost" | "destructive";
  hidden?: (row: Row) => boolean;
  disabled?: (row: Row) => boolean;
  onClick: (row: Row) => void;
};

export type TableAction = {
  key: string;
  label: string;
  icon?: LucideIcon;
  variant?: "default" | "outline" | "secondary" | "ghost" | "destructive";
  disabled?: boolean;
  onClick: () => void;
};

export type SortState = { key: string; direction: "asc" | "desc" } | undefined;

export type DataTableQueryArgs = {
  page: number;
  pageSize: number;
  sortBy?: string;
  sortDir?: "asc" | "desc";
  filters: Record<string, string>;
};

export type PaginatedResponse<Row> = {
  items: Row[];
  total?: number;
  page?: number;
  pageSize?: number;
};

export type DataTableQueryResult<Row> = {
  data?: PaginatedResponse<Row> | Row[];
  isLoading: boolean;
  isFetching: boolean;
  isError: boolean;
  error?: unknown;
  refetch: () => void;
};

export type DataTableQueryHook<Row> = (args: DataTableQueryArgs) => DataTableQueryResult<Row>;

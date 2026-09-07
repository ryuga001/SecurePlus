"use client";

import { ArrowDown, ArrowUp, ArrowUpDown, CircleAlert, Inbox, Loader2 } from "lucide-react";
import * as React from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { StatusBadge } from "./status-badge";
import { TableFilters } from "./table-filters";
import { PAGE_SIZES, TablePagination } from "./table-pagination";
import type {
  ColumnConfig,
  DataTableQueryArgs,
  DataTableQueryHook,
  FilterConfig,
  FilterState,
  FilterValue,
  PaginatedResponse,
  RowAction,
  SortState,
  TableAction,
} from "./types";

function initialFilterState(filters: FilterConfig[]): FilterState {
  const state: FilterState = {};
  for (const filter of filters) {
    if (filter.defaultValue !== undefined) state[filter.key] = filter.defaultValue;
  }
  return state;
}

function toQueryParams(values: FilterState): Record<string, string> {
  const params: Record<string, string> = {};

  for (const [key, value] of Object.entries(values)) {
    if (value === undefined || value === "" || value === false) continue;

    if (typeof value === "object") {
      if (value.from) params[`${key}From`] = value.from;
      if (value.to) params[`${key}To`] = value.to;
      continue;
    }

    params[key] = String(value);
  }

  return params;
}

function useDebounced<T>(value: T, delay = 350) {
  const [debounced, setDebounced] = React.useState(value);

  React.useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debounced;
}

function formatValue(value: unknown, type: ColumnConfig<unknown>["type"]) {
  if (value === null || value === undefined || value === "") return "—";

  switch (type) {
    case "number":
      return typeof value === "number" ? value.toLocaleString() : String(value);
    case "date":
      return new Date(String(value)).toLocaleDateString();
    case "datetime":
      return new Date(String(value)).toLocaleString();
    case "boolean":
      return value ? "Yes" : "No";
    case "status":
      return <StatusBadge status={String(value)} />;
    default:
      return String(value);
  }
}

function errorMessage(error: unknown) {
  if (!error) return "Something went wrong while loading this table.";

  if (typeof error === "object" && error !== null) {
    const shape = error as { data?: { message?: string }; error?: string; status?: number };
    if (shape.data?.message) return shape.data.message;
    if (shape.error) return shape.error;
    if (shape.status) return `Request failed with status ${shape.status}`;
  }

  return "Something went wrong while loading this table.";
}

function normalize<Row>(data: PaginatedResponse<Row> | Row[] | undefined) {
  if (!data) return { items: [] as Row[], total: undefined as number | undefined };
  if (Array.isArray(data)) return { items: data, total: data.length };
  return { items: data.items ?? [], total: data.total };
}

export type DataTableProps<Row> = {
  query: DataTableQueryHook<Row>;
  columns: ColumnConfig<Row>[];
  filters?: FilterConfig[];
  actions?: TableAction[];
  rowActions?: RowAction<Row>[];
  getRowId?: (row: Row, index: number) => string | number;
  onRowClick?: (row: Row) => void;
  defaultPageSize?: (typeof PAGE_SIZES)[number];
  defaultSort?: SortState;
  emptyMessage?: string;
  className?: string;
};

export function DataTable<Row>({
  query: useTableQuery,
  columns,
  filters = [],
  actions = [],
  rowActions = [],
  getRowId,
  onRowClick,
  defaultPageSize = 25,
  defaultSort,
  emptyMessage = "No records found",
  className,
}: DataTableProps<Row>) {
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState<number>(defaultPageSize);
  const [sort, setSort] = React.useState<SortState>(defaultSort);
  const [filterValues, setFilterValues] = React.useState<FilterState>(() =>
    initialFilterState(filters)
  );

  const debouncedFilters = useDebounced(filterValues);

  const args: DataTableQueryArgs = React.useMemo(
    () => ({
      page,
      pageSize,
      sortBy: sort?.key,
      sortDir: sort?.direction,
      filters: toQueryParams(debouncedFilters),
    }),
    [page, pageSize, sort, debouncedFilters]
  );

  const { data, isLoading, isFetching, isError, error, refetch } = useTableQuery(args);
  const { items, total } = normalize<Row>(data);

  function updateFilter(key: string, value: FilterValue) {
    setFilterValues((current) => ({ ...current, [key]: value }));
    setPage(1);
  }

  function clearFilters() {
    setFilterValues(initialFilterState(filters));
    setPage(1);
  }

  function changePageSize(size: number) {
    setPageSize(size);
    setPage(1);
  }

  function toggleSort(column: ColumnConfig<Row>) {
    if (!column.sortable) return;

    setSort((current) => {
      if (current?.key !== column.key) return { key: column.key, direction: "asc" };
      if (current.direction === "asc") return { key: column.key, direction: "desc" };
      return undefined;
    });
  }

  const visibleActions = actions;
  const visibleRowActions = rowActions;

  const columnCount = columns.length + (visibleRowActions.length > 0 ? 1 : 0);

  return (
    <div className={cn("flex flex-col border bg-card", className)}>
      {(filters.length > 0 || visibleActions.length > 0) && (
        <div className="flex flex-wrap items-end justify-between gap-4 border-b p-4">
          <TableFilters
            filters={filters}
            values={filterValues}
            onChange={updateFilter}
            onClear={clearFilters}
          />

          <div className="ml-auto flex flex-wrap items-center gap-2">
            {visibleActions.map((action) => {
              const Icon = action.icon;

              return (
                <Button
                  key={action.key}
                  type="button"
                  variant={action.variant ?? "outline"}
                  disabled={action.disabled}
                  onClick={action.onClick}
                >
                  {Icon ? <Icon /> : null}
                  {action.label}
                </Button>
              );
            })}
          </div>
        </div>
      )}

      <div className="relative overflow-x-auto">
        {isFetching && !isLoading ? (
          <div className="absolute inset-x-0 top-0 h-0.5 animate-pulse bg-primary" />
        ) : null}

        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b bg-muted/60">
              {columns.map((column) => {
                const sorted = sort?.key === column.key;
                const SortIcon = !sorted ? ArrowUpDown : sort?.direction === "asc" ? ArrowUp : ArrowDown;

                return (
                  <th
                    key={column.key}
                    scope="col"
                    style={column.width ? { width: column.width } : undefined}
                    className={cn(
                      "px-4 py-3 text-xs font-semibold tracking-wide text-muted-foreground uppercase",
                      column.align === "right" && "text-right",
                      column.align === "center" && "text-center",
                      !column.align && "text-left"
                    )}
                  >
                    {column.sortable ? (
                      <button
                        type="button"
                        onClick={() => toggleSort(column)}
                        className={cn(
                          "inline-flex items-center gap-1.5 transition-colors hover:text-foreground",
                          sorted && "text-primary"
                        )}
                      >
                        {column.label}
                        <SortIcon className="size-3.5" />
                      </button>
                    ) : (
                      column.label
                    )}
                  </th>
                );
              })}

              {visibleRowActions.length > 0 ? (
                <th
                  scope="col"
                  className="px-4 py-3 text-right text-xs font-semibold tracking-wide text-muted-foreground uppercase"
                >
                  Actions
                </th>
              ) : null}
            </tr>
          </thead>

          <tbody>
            {isLoading ? (
              <tr>
                <td colSpan={columnCount} className="px-4 py-16">
                  <div className="flex flex-col items-center gap-3 text-muted-foreground">
                    <Loader2 className="size-6 animate-spin text-primary" />
                    <p className="text-sm">Loading records...</p>
                  </div>
                </td>
              </tr>
            ) : null}

            {!isLoading && isError ? (
              <tr>
                <td colSpan={columnCount} className="px-4 py-16">
                  <div className="flex flex-col items-center gap-3 text-center">
                    <CircleAlert className="size-6 text-destructive" />
                    <p className="text-sm font-medium">Could not load data</p>
                    <p className="max-w-md text-sm text-muted-foreground">{errorMessage(error)}</p>
                    <Button type="button" variant="outline" size="sm" onClick={() => refetch()}>
                      Try again
                    </Button>
                  </div>
                </td>
              </tr>
            ) : null}

            {!isLoading && !isError && items.length === 0 ? (
              <tr>
                <td colSpan={columnCount} className="px-4 py-16">
                  <div className="flex flex-col items-center gap-3 text-muted-foreground">
                    <Inbox className="size-6" />
                    <p className="text-sm">{emptyMessage}</p>
                  </div>
                </td>
              </tr>
            ) : null}

            {!isLoading && !isError
              ? items.map((row, index) => (
                  <tr
                    key={getRowId ? getRowId(row, index) : index}
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                    className={cn(
                      "border-b last:border-b-0",
                      onRowClick && "cursor-pointer",
                      "hover:bg-muted/50"
                    )}
                  >
                    {columns.map((column) => {
                      const value = column.accessor
                        ? column.accessor(row)
                        : (row as Record<string, unknown>)[column.key];

                      return (
                        <td
                          key={column.key}
                          className={cn(
                            "px-4 py-3 align-middle",
                            column.align === "right" && "text-right",
                            column.align === "center" && "text-center"
                          )}
                        >
                          {column.render
                            ? column.render(value, row, index)
                            : formatValue(value, column.type)}
                        </td>
                      );
                    })}

                    {visibleRowActions.length > 0 ? (
                      <td className="px-4 py-3 text-right">
                        <div className="flex justify-end gap-1.5">
                          {visibleRowActions
                            .filter((action) => !action.hidden?.(row))
                            .map((action) => {
                              const Icon = action.icon;

                              return (
                                <Button
                                  key={action.key}
                                  type="button"
                                  variant={action.variant ?? "ghost"}
                                  size="sm"
                                  disabled={action.disabled?.(row)}
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    action.onClick(row);
                                  }}
                                >
                                  {Icon ? <Icon /> : null}
                                  {action.label}
                                </Button>
                              );
                            })}
                        </div>
                      </td>
                    ) : null}
                  </tr>
                ))
              : null}
          </tbody>
        </table>
      </div>

      <TablePagination
        page={page}
        pageSize={pageSize}
        rowCount={items.length}
        total={total}
        onPageChange={setPage}
        onPageSizeChange={changePageSize}
      />
    </div>
  );
}

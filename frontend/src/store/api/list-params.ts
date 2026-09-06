import type { DataTableQueryArgs } from "@/components/data-table/types";

export function listParams({ page, pageSize, sortBy, sortDir, filters }: DataTableQueryArgs) {
  return {
    page,
    page_size: pageSize,
    ...(sortBy ? { sort_by: sortBy, sort_dir: sortDir } : {}),
    ...filters,
  };
}

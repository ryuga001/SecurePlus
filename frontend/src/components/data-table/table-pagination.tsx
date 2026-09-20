"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";

export const PAGE_SIZES = [10, 25, 50] as const;

export function TablePagination({
  page,
  pageSize,
  rowCount,
  total,
  onPageChange,
  onPageSizeChange,
}: {
  page: number;
  pageSize: number;
  rowCount: number;
  total?: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}) {
  const t = useTranslations("table");
  const knownTotal = typeof total === "number";
  const lastPage = knownTotal ? Math.max(1, Math.ceil(total / pageSize)) : undefined;
  const first = rowCount === 0 ? 0 : (page - 1) * pageSize + 1;
  const last = rowCount === 0 ? 0 : first + rowCount - 1;

  const canPrev = page > 1;
  const canNext = knownTotal ? page < (lastPage as number) : rowCount === pageSize;

  return (
    <div className="flex flex-wrap items-center justify-between gap-4 border-t border-border-subtle px-4 py-3">
      <div className="flex items-center gap-2.5">
        <label htmlFor="page-size" className="text-sm text-muted-foreground">
          {t("rowsPerPage")}
        </label>
        <select
          id="page-size"
          className="h-8 w-20 rounded-md border border-input bg-surface px-2 text-[0.8125rem] text-foreground outline-none focus-visible:border-primary focus-visible:shadow-[0_0_0_2px_var(--primary-container)]"
          value={pageSize}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
        >
          {PAGE_SIZES.map((size) => (
            <option key={size} value={size}>
              {size}
            </option>
          ))}
        </select>
      </div>

      <p className="text-sm text-muted-foreground">
        {knownTotal
          ? t("showing", { first, last, total: total as number })
          : t("page", { page })}
      </p>

      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!canPrev}
          onClick={() => onPageChange(page - 1)}
        >
          <ChevronLeft />
          {t("previous")}
        </Button>

        <span className="px-1 text-sm text-muted-foreground">
          {lastPage ? `${page} / ${lastPage}` : page}
        </span>

        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!canNext}
          onClick={() => onPageChange(page + 1)}
        >
          {t("next")}
          <ChevronRight />
        </Button>
      </div>
    </div>
  );
}

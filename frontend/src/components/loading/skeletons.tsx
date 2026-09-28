"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

const CELL_WIDTHS = ["w-3/5", "w-2/5", "w-4/5", "w-1/2", "w-2/3", "w-1/3"] as const;

function cellWidth(row: number, column: number) {
  return CELL_WIDTHS[(row * 3 + column * 5) % CELL_WIDTHS.length];
}

export function LoadingRegion({
  label,
  className,
  children,
}: {
  label?: string;
  className?: string;
  children: React.ReactNode;
}) {
  const common = useTranslations("common");

  return (
    <div role="status" aria-live="polite" aria-busy="true" className={className}>
      <span className="sr-only">{label ?? common("loading")}</span>
      {children}
    </div>
  );
}

export function TableRowsSkeleton({ columns, rows }: { columns: number; rows: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, row) => (
        <tr key={row} aria-hidden className="h-12 border-b border-border-subtle last:border-b-0">
          {Array.from({ length: columns }, (_, column) => (
            <td key={column} className="px-3 py-3 align-middle">
              <Skeleton className={cn("h-3.5", column === 0 ? "w-3/4" : cellWidth(row, column))} />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function TableFrameSkeleton({ columns, rows }: { columns: number; rows: number }) {
  return (
    <div className="flex flex-col border border-border bg-surface">
      <div className="flex flex-wrap items-end justify-between gap-4 border-b border-border-subtle p-4">
        <Skeleton className="h-10 w-72 max-w-full" />
        <div className="ml-auto flex gap-2">
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-32" />
        </div>
      </div>

      <table className="w-full border-collapse text-sm">
        <thead>
          <tr className="h-10 border-b border-border-subtle bg-surface-container-low">
            {Array.from({ length: columns }, (_, column) => (
              <th key={column} className="px-3">
                <Skeleton className="h-3 w-16" />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          <TableRowsSkeleton columns={columns} rows={rows} />
        </tbody>
      </table>

      <div className="flex items-center justify-between gap-4 border-t border-border-subtle px-4 py-3">
        <Skeleton className="h-4 w-40" />
        <div className="flex gap-2">
          <Skeleton className="size-8" />
          <Skeleton className="size-8" />
        </div>
      </div>
    </div>
  );
}

export function DataTableSkeleton({
  columns = 5,
  rows = 8,
  label,
}: {
  columns?: number;
  rows?: number;
  label?: string;
}) {
  return (
    <LoadingRegion label={label}>
      <TableFrameSkeleton columns={columns} rows={rows} />
    </LoadingRegion>
  );
}

export function FormSkeleton({ fields = 4, label }: { fields?: number; label?: string }) {
  return (
    <LoadingRegion label={label} className="flex flex-col gap-6">
      {Array.from({ length: fields }, (_, index) => (
        <div key={index} className="flex flex-col gap-2">
          <Skeleton className={cn("h-3.5", index % 2 === 0 ? "w-28" : "w-36")} />
          <Skeleton className={index === fields - 1 ? "h-24 w-full" : "h-10 w-full"} />
        </div>
      ))}

      <div className="flex justify-end gap-2 border-t pt-4">
        <Skeleton className="h-10 w-24" />
        <Skeleton className="h-10 w-32" />
      </div>
    </LoadingRegion>
  );
}

export function MetricsSkeleton({ count = 4, className }: { count?: number; className?: string }) {
  return (
    <div className={cn("grid grid-cols-2 gap-3 sm:grid-cols-4", className)}>
      {Array.from({ length: count }, (_, index) => (
        <div key={index} className="flex flex-col gap-2 border bg-surface-container-low p-3">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-5 w-14" />
        </div>
      ))}
    </div>
  );
}

export function DetailSkeleton({ sections = 2, label }: { sections?: number; label?: string }) {
  return (
    <LoadingRegion label={label} className="flex flex-col gap-6">
      <div className="flex items-center gap-3">
        <Skeleton className="h-6 w-20" />
        <Skeleton className="h-4 w-48" />
      </div>

      <MetricsSkeleton />

      {Array.from({ length: sections }, (_, section) => (
        <div key={section} className="flex flex-col gap-3">
          <Skeleton className="h-4 w-32" />
          <div className="flex flex-col gap-2.5 border p-4">
            {Array.from({ length: 4 }, (_, line) => (
              <div key={line} className="flex items-center justify-between gap-6">
                <Skeleton className="h-3.5 w-28" />
                <Skeleton className={cn("h-3.5", cellWidth(section, line), "max-w-60")} />
              </div>
            ))}
          </div>
        </div>
      ))}
    </LoadingRegion>
  );
}

export function ListSkeleton({
  rows = 6,
  selectable = true,
  label,
  className,
}: {
  rows?: number;
  selectable?: boolean;
  label?: string;
  className?: string;
}) {
  return (
    <LoadingRegion label={label} className={cn("flex flex-col", className)}>
      {Array.from({ length: rows }, (_, row) => (
        <div key={row} className="flex items-center gap-3 border-b px-4 py-2.5 last:border-b-0">
          {selectable ? <Skeleton className="size-4 shrink-0 rounded-sm" /> : null}
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Skeleton className={cn("h-3.5", cellWidth(row, 0))} />
            <Skeleton className={cn("h-3", cellWidth(row, 1), "max-w-40")} />
          </div>
        </div>
      ))}
    </LoadingRegion>
  );
}

export function OptionSkeleton({ rows = 3, label }: { rows?: number; label?: string }) {
  return (
    <LoadingRegion label={label} className="flex flex-col gap-1 p-1">
      {Array.from({ length: rows }, (_, row) => (
        <div key={row} className="flex flex-col gap-1.5 px-2 py-2">
          <Skeleton className={cn("h-3.5", cellWidth(row, 0))} />
          <Skeleton className="h-3 w-20" />
        </div>
      ))}
    </LoadingRegion>
  );
}

export function SettingsCardSkeleton({ rows = 2 }: { rows?: number }) {
  return (
    <div className="flex flex-col gap-5 rounded-lg border bg-surface p-6">
      <div className="flex flex-col gap-2">
        <Skeleton className="h-4.5 w-40" />
        <Skeleton className="h-3.5 w-72 max-w-full" />
      </div>

      {Array.from({ length: rows }, (_, row) => (
        <div key={row} className="grid gap-2 md:grid-cols-[minmax(0,220px)_1fr] md:gap-8">
          <div className="flex flex-col gap-2">
            <Skeleton className="h-3.5 w-28" />
            <Skeleton className="h-3 w-40" />
          </div>
          <Skeleton className="h-10 w-full md:max-w-xl" />
        </div>
      ))}
    </div>
  );
}

export function SettingsSkeleton({ cards = [1, 3], label }: { cards?: number[]; label?: string }) {
  return (
    <LoadingRegion label={label} className="flex flex-col gap-6">
      {cards.map((rows, index) => (
        <SettingsCardSkeleton key={index} rows={rows} />
      ))}
    </LoadingRegion>
  );
}

export function AppShellSkeleton() {
  return (
    <LoadingRegion className="flex h-svh w-full overflow-hidden">
      <aside className="flex h-svh w-60 shrink-0 flex-col border-r bg-sidebar">
        <div className="flex h-16 items-center gap-3 border-b border-sidebar-border px-4">
          <Skeleton className="size-8" />
          <Skeleton className="h-4 w-28" />
        </div>
        <div className="flex flex-1 flex-col gap-2 px-3 py-4">
          {Array.from({ length: 7 }, (_, index) => (
            <div key={index} className="flex h-10 items-center gap-3 px-3">
              <Skeleton className="size-4.5 shrink-0" />
              <Skeleton className={cn("h-3.5", cellWidth(index, 2))} />
            </div>
          ))}
        </div>
        <div className="flex flex-col gap-2 border-t border-sidebar-border p-5">
          <Skeleton className="h-3.5 w-32" />
          <Skeleton className="h-3 w-40" />
          <Skeleton className="mt-1 h-9 w-full" />
        </div>
      </aside>

      <div className="min-w-0 flex-1 overflow-hidden">
        <div className="mx-auto flex w-full max-w-[1440px] flex-col gap-6 p-4 sm:p-6 xl:p-8">
          <div className="flex flex-col gap-2">
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-4 w-96 max-w-full" />
          </div>
          <div className="flex gap-8 border-b pb-3">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-24" />
          </div>
          <TableFrameSkeleton columns={5} rows={8} />
        </div>
      </div>
    </LoadingRegion>
  );
}

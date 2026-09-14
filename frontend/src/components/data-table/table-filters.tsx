"use client";

import { Search, X } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { cn } from "@/lib/utils";

import type { DateRangeValue, FilterConfig, FilterState, FilterValue } from "./types";

function asText(value: FilterValue) {
  return typeof value === "string" ? value : "";
}

function asRange(value: FilterValue): DateRangeValue {
  return value && typeof value === "object" ? value : {};
}

export function TableFilters({
  filters,
  values,
  onChange,
  onClear,
}: {
  filters: FilterConfig[];
  values: FilterState;
  onChange: (key: string, value: FilterValue) => void;
  onClear: () => void;
}) {
  const t = useTranslations("table");

  if (filters.length === 0) return null;

  const active = filters.some((filter) => {
    const value = values[filter.key];
    if (value === undefined || value === "" || value === false) return false;
    if (typeof value === "object") return Boolean(value.from || value.to);
    return true;
  });

  return (
    <div className="flex flex-wrap items-end gap-3">
      {filters.map((filter) => {
        const value = values[filter.key];

        return (
          <div key={filter.key} className={cn("flex flex-col gap-1.5", filter.width ?? "w-52")}>
            <label
              htmlFor={`filter-${filter.key}`}
              className="text-xs font-medium tracking-wide text-muted-foreground uppercase"
            >
              {filter.label}
            </label>

            {filter.type === "text" ? (
              <div className="relative">
                <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id={`filter-${filter.key}`}
                  className="pl-9"
                  placeholder={
                    filter.placeholder ?? t("searchFilter", { label: filter.label.toLowerCase() })
                  }
                  value={asText(value)}
                  onChange={(event) => onChange(filter.key, event.target.value)}
                />
              </div>
            ) : null}

            {filter.type === "select" ? (
              <Select
                id={`filter-${filter.key}`}
                value={asText(value)}
                onChange={(event) => onChange(filter.key, event.target.value)}
                placeholder={filter.placeholder ?? t("all")}
                options={filter.options}
              />
            ) : null}

            {filter.type === "date" ? (
              <Input
                id={`filter-${filter.key}`}
                type="date"
                value={asText(value)}
                onChange={(event) => onChange(filter.key, event.target.value)}
              />
            ) : null}

            {filter.type === "dateRange" ? (
              <div className="flex items-center gap-2">
                <Input
                  id={`filter-${filter.key}`}
                  type="date"
                  value={asRange(value).from ?? ""}
                  onChange={(event) =>
                    onChange(filter.key, { ...asRange(value), from: event.target.value })
                  }
                />
                <span className="text-sm text-muted-foreground">to</span>
                <Input
                  type="date"
                  value={asRange(value).to ?? ""}
                  onChange={(event) =>
                    onChange(filter.key, { ...asRange(value), to: event.target.value })
                  }
                />
              </div>
            ) : null}

            {filter.type === "checkbox" ? (
              <label className="flex h-11 items-center gap-2.5 border border-input px-3 text-sm">
                <input
                  id={`filter-${filter.key}`}
                  type="checkbox"
                  className="size-4 accent-[var(--primary)]"
                  checked={value === true}
                  onChange={(event) => onChange(filter.key, event.target.checked)}
                />
                {filter.placeholder ?? filter.label}
              </label>
            ) : null}
          </div>
        );
      })}

      {active ? (
        <Button type="button" variant="ghost" size="sm" className="h-11" onClick={onClear}>
          <X />
          {t("clear")}
        </Button>
      ) : null}
    </div>
  );
}

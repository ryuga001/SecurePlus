"use client";

import { CheckCheck, CircleAlert, Inbox, Loader2, Search, X } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { useListPoliciesQuery, type PolicyListItem } from "@/store/api/policies-api";

export type SelectedPolicy = { id: number; label: string };

const PAGE_SIZE = 10;
const SEARCH_DEBOUNCE = 350;

export function PolicyMultiSelect({
  value,
  onChange,
  disabled,
  invalid,
  max = 200,
}: {
  value: SelectedPolicy[];
  onChange: (next: SelectedPolicy[]) => void;
  disabled?: boolean;
  invalid?: boolean;
  max?: number;
}) {
  const t = useTranslations("email-alert");
  const common = useTranslations("common");
  const table = useTranslations("table");

  const [search, setSearch] = React.useState("");
  const [debounced, setDebounced] = React.useState("");
  const [page, setPage] = React.useState(1);

  React.useEffect(() => {
    const timer = setTimeout(() => setDebounced(search.trim()), SEARCH_DEBOUNCE);

    return () => clearTimeout(timer);
  }, [search]);

  React.useEffect(() => {
    setPage(1);
  }, [debounced]);

  const { data, isLoading, isFetching, isError, refetch } = useListPoliciesQuery({
    page,
    pageSize: PAGE_SIZE,
    filters: debounced ? { search: debounced } : {},
  });

  const rows = React.useMemo<PolicyListItem[]>(() => data?.items ?? [], [data]);
  const total = data?.total ?? 0;

  const selected = React.useMemo(
    () => new Set(value.map((item) => item.id)),
    [value],
  );

  const pageAllSelected =
    rows.length > 0 && rows.every((row) => selected.has(row.id));

  const atMax = value.length >= max;

  function toggle(row: PolicyListItem, checked: boolean) {
    if (!checked) {
      onChange(value.filter((item) => item.id !== row.id));
      return;
    }

    if (selected.has(row.id) || atMax) return;

    onChange([...value, { id: row.id, label: row.policy_name }]);
  }

  function selectPage() {
    const additions: SelectedPolicy[] = [];

    for (const row of rows) {
      if (selected.has(row.id)) continue;
      if (value.length + additions.length >= max) break;

      additions.push({ id: row.id, label: row.policy_name });
    }

    if (additions.length > 0) onChange([...value, ...additions]);
  }

  const lastPage = total > 0 ? Math.ceil(total / PAGE_SIZE) : 1;

  return (
    <div
      className={cn(
        "overflow-hidden rounded-lg border",
        invalid && "border-error",
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3">
        <span className="text-sm tabular-nums">
          <span className="font-semibold">{value.length}</span>{" "}
          <span className="text-muted-foreground">{t("dialog.selectedCount")}</span>
        </span>

        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={disabled || rows.length === 0 || pageAllSelected || atMax}
            onClick={selectPage}
          >
            <CheckCheck />
            {t("dialog.selectAllVisible")}
          </Button>

          <Button
            type="button"
            size="sm"
            variant="ghost"
            disabled={disabled || value.length === 0}
            onClick={() => onChange([])}
          >
            <X />
            {t("dialog.clearSelection")}
          </Button>
        </div>
      </div>

      <div className="max-h-32 overflow-y-auto border-b px-4 py-3">
        {value.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            {t("dialog.noSelectedPolicies")}
          </p>
        ) : (
          <div className="flex flex-wrap gap-1.5">
            {value.map((item) => (
              <Badge key={item.id} variant="secondary" className="gap-1 pr-1">
                <span className="max-w-40 truncate">{item.label}</span>
                <button
                  type="button"
                  aria-label={t("dialog.removePolicy", { name: item.label })}
                  disabled={disabled}
                  onClick={() => onChange(value.filter((row) => row.id !== item.id))}
                  className="rounded-full p-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-40"
                >
                  <X className="size-3" />
                </button>
              </Badge>
            ))}
          </div>
        )}
      </div>

      <div className="border-b p-4">
        <div className="relative">
          <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t("dialog.searchPolicies")}
            aria-label={common("search")}
            value={search}
            disabled={disabled}
            onChange={(event) => setSearch(event.target.value)}
          />
        </div>
      </div>

      <div className="relative min-h-48">
        {isFetching && !isLoading ? (
          <div className="absolute inset-x-0 top-0 h-0.5 animate-pulse bg-primary" />
        ) : null}

        {isLoading ? (
          <div className="flex min-h-48 items-center justify-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            {common("loading")}
          </div>
        ) : isError ? (
          <div className="flex min-h-48 flex-col items-center justify-center gap-2 text-sm">
            <CircleAlert className="size-5 text-error" />
            <p className="text-muted-foreground">{table("couldNotLoad")}</p>
            <Button type="button" size="sm" variant="outline" onClick={() => refetch()}>
              {table("tryAgain")}
            </Button>
          </div>
        ) : rows.length === 0 ? (
          <div className="flex min-h-48 flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
            <Inbox className="size-5" />
            {t("dialog.noPolicies")}
          </div>
        ) : (
          <ul className="divide-y">
            {rows.map((row) => {
              const checked = selected.has(row.id);

              return (
                <li key={row.id}>
                  <label
                    className={cn(
                      "flex cursor-pointer items-center gap-3 px-4 py-2.5 transition-colors",
                      checked ? "bg-primary-container/40" : "hover:bg-surface-container-low",
                    )}
                  >
                    <Checkbox
                      checked={checked}
                      disabled={disabled || (!checked && atMax)}
                      aria-label={t("dialog.selectPolicy", { name: row.policy_name })}
                      onCheckedChange={(next) => toggle(row, next === true)}
                    />

                    <span className="min-w-0 flex-1 truncate text-sm font-medium">
                      {row.policy_name}
                    </span>

                    <Badge variant="secondary" size="sm">
                      {row.action}
                    </Badge>
                  </label>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t px-4 py-3 text-xs text-muted-foreground">
        <span className="tabular-nums">
          {t("dialog.policyPageStatus", { page, pages: lastPage, total })}
        </span>

        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={disabled || page <= 1 || isFetching}
            onClick={() => setPage((current) => Math.max(current - 1, 1))}
          >
            {table("previous")}
          </Button>

          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={disabled || page >= lastPage || isFetching}
            onClick={() => setPage((current) => current + 1)}
          >
            {table("next")}
          </Button>
        </div>
      </div>
    </div>
  );
}

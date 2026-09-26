"use client";

import { ChevronLeft, ChevronRight, CircleAlert, Inbox, Loader2, Search, X } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";

import {
  TARGET_LIMIT,
  newTargetRow,
  type TargetRow,
} from "@/components/data-discovery/target-list-editor";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { apiErrorMessage } from "@/lib/api-error";
import { emptyTargetValues, targetFields, type SourceType } from "@/lib/data-discovery";
import { cn } from "@/lib/utils";
import {
  useBrowseDiscoveryTargetsInfiniteQuery,
  type TargetOption,
} from "@/store/api/data-discovery-configurations-api";

const SEARCH_DEBOUNCE = 350;
const SCROLL_THRESHOLD = 96;

type Filter = "all" | "selected";

function isBlank(row: TargetRow) {
  return Object.values(row.values).every((value) => !value.trim());
}

function primaryKey(sourceType: SourceType) {
  return targetFields(sourceType)[0].key;
}

function rowMatches(sourceType: SourceType, row: TargetRow, option: TargetOption, parent: TargetOption | null) {
  const library = (row.values.documentLibrary ?? "").trim();

  if (parent) return row.values.site === parent.value && library === option.value;
  if (sourceType === "SHARE_POINT") return row.values.site === option.value && library === "";

  return (row.values[primaryKey(sourceType)] ?? "") === option.value;
}

function rowFor(sourceType: SourceType, option: TargetOption, parent: TargetOption | null): TargetRow {
  const values = emptyTargetValues(sourceType);

  if (parent) {
    values.site = parent.value;
    values.documentLibrary = option.value;
  } else {
    values[primaryKey(sourceType)] = option.value;
  }

  return {
    key: crypto.randomUUID(),
    values,
    label: parent ? `${parent.label} › ${option.label}` : option.label,
  };
}

function describeRow(sourceType: SourceType, row: TargetRow) {
  const fields = targetFields(sourceType);
  const primary = row.values[fields[0].key] ?? "";

  return {
    label: row.label || primary,
    detail: fields
      .slice(row.label ? 0 : 1)
      .map((field) => (row.values[field.key] ?? "").trim())
      .filter(Boolean)
      .join(" / "),
  };
}

export function TargetBrowser({
  id,
  configurationId,
  sourceType,
  value,
  onChange,
  disabled,
  max = TARGET_LIMIT,
}: {
  id: string;
  configurationId: number | null;
  sourceType: SourceType;
  value: TargetRow[];
  onChange: (next: TargetRow[]) => void;
  disabled?: boolean;
  max?: number;
}) {
  const t = useTranslations("data-discovery.policies.dialog.browser");
  const common = useTranslations("common");
  const table = useTranslations("table");

  const [search, setSearch] = React.useState("");
  const [debounced, setDebounced] = React.useState("");
  const [filter, setFilter] = React.useState<Filter>("all");
  const [parent, setParent] = React.useState<TargetOption | null>(null);

  const listRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const timer = setTimeout(() => setDebounced(search.trim()), SEARCH_DEBOUNCE);

    return () => clearTimeout(timer);
  }, [search]);

  const {
    data,
    error,
    isLoading,
    isFetching,
    isError,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    refetch,
  } = useBrowseDiscoveryTargetsInfiniteQuery(
    {
      configurationId: configurationId ?? 0,
      sourceType,
      search: debounced,
      parent: parent?.value ?? "",
    },
    { skip: configurationId === null || filter === "selected" },
  );

  const options = React.useMemo(() => data?.pages.flatMap((page) => page.items) ?? [], [data]);
  const chosen = React.useMemo(() => value.filter((row) => !isBlank(row)), [value]);
  const atMax = chosen.length >= max;

  const visibleChosen = React.useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) return chosen;

    return chosen.filter((row) => {
      const { label, detail } = describeRow(sourceType, row);

      return `${label} ${detail}`.toLowerCase().includes(term);
    });
  }, [chosen, search, sourceType]);

  const loadMore = React.useCallback(() => {
    const list = listRef.current;
    if (!list || filter !== "all" || !hasNextPage || isFetchingNextPage || isError) return;

    if (list.scrollHeight - list.scrollTop - list.clientHeight <= SCROLL_THRESHOLD) {
      void fetchNextPage();
    }
  }, [fetchNextPage, filter, hasNextPage, isError, isFetchingNextPage]);

  React.useEffect(() => {
    loadMore();
  }, [options.length, loadMore]);

  function select(option: TargetOption, checked: boolean) {
    const matching = (row: TargetRow) => rowMatches(sourceType, row, option, parent);

    if (!checked) {
      const remaining = value.filter((row) => !matching(row));
      onChange(remaining.length > 0 ? remaining : [newTargetRow(sourceType)]);

      return;
    }

    if (atMax || value.some(matching)) return;

    const next = rowFor(sourceType, option, parent);
    onChange(value.length === 1 && isBlank(value[0]) ? [next] : [...value, next]);
  }

  function remove(row: TargetRow) {
    const remaining = value.filter((item) => item.key !== row.key);
    onChange(remaining.length > 0 ? remaining : [newTargetRow(sourceType)]);
  }

  function openParent(option: TargetOption) {
    setParent(option);
    setSearch("");
    setDebounced("");
    listRef.current?.scrollTo({ top: 0 });
  }

  if (configurationId === null) {
    return (
      <div className="flex min-h-24 items-center justify-center rounded-lg border border-dashed px-4 text-center text-sm text-muted-foreground">
        {t("selectConfiguration")}
      </div>
    );
  }

  function allView() {
    if (isLoading || (options.length === 0 && !isError && (hasNextPage || isFetchingNextPage))) {
      return (
        <div className="flex min-h-48 items-center justify-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          {common("loading")}
        </div>
      );
    }

    if (isError && options.length === 0) {
      return (
        <div className="flex min-h-48 flex-col items-center justify-center gap-2 px-4 text-center text-sm">
          <CircleAlert className="size-5 text-error" />
          <p className="max-w-sm text-muted-foreground">{apiErrorMessage(error, t("loadFailed"))}</p>
          <Button type="button" size="sm" variant="outline" onClick={() => refetch()}>
            {table("tryAgain")}
          </Button>
        </div>
      );
    }

    if (options.length === 0) {
      return (
        <div className="flex min-h-48 flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
          <Inbox className="size-5" />
          {t("empty")}
        </div>
      );
    }

    return (
      <>
        <ul className="divide-y">
          {options.map((option) => {
            const checked = value.some((row) => rowMatches(sourceType, row, option, parent));

            return (
              <li key={option.value} className="flex items-center">
                <label
                  className={cn(
                    "flex min-w-0 flex-1 cursor-pointer items-center gap-3 px-4 py-2.5 transition-colors",
                    checked ? "bg-primary-container/40" : "hover:bg-surface-container-low",
                  )}
                >
                  <Checkbox
                    checked={checked}
                    disabled={disabled || (!checked && atMax)}
                    aria-label={t("selectTarget", { name: option.label })}
                    onCheckedChange={(next) => select(option, next === true)}
                  />

                  <span className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate text-sm font-medium">{option.label}</span>
                    {option.description && option.description !== option.label ? (
                      <span className="truncate font-mono text-xs text-muted-foreground">
                        {option.description}
                      </span>
                    ) : null}
                  </span>

                  {option.kind && t.has(`kinds.${option.kind}`) ? (
                    <Badge variant="secondary" size="sm" className="shrink-0">
                      {t(`kinds.${option.kind}`)}
                    </Badge>
                  ) : null}
                </label>

                {option.expandable && !parent ? (
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    className="mr-2"
                    aria-label={t("browseLibraries", { name: option.label })}
                    disabled={disabled}
                    onClick={() => openParent(option)}
                  >
                    <ChevronRight />
                  </Button>
                ) : null}
              </li>
            );
          })}
        </ul>

        <div className="flex min-h-10 items-center justify-center gap-2 border-t px-4 py-2 text-xs text-muted-foreground">
          {isFetchingNextPage ? (
            <>
              <Loader2 className="size-3 animate-spin" />
              {t("loadingMore")}
            </>
          ) : isError ? (
            <span className="flex flex-wrap items-center justify-center gap-2 text-center">
              <span className="text-error-text">{apiErrorMessage(error, t("loadFailed"))}</span>
              <Button type="button" size="sm" variant="ghost" onClick={() => void fetchNextPage()}>
                {table("tryAgain")}
              </Button>
            </span>
          ) : hasNextPage ? (
            t("scrollForMore")
          ) : (
            t("end")
          )}
        </div>
      </>
    );
  }

  function selectedView() {
    if (visibleChosen.length === 0) {
      return (
        <div className="flex min-h-48 flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
          <Inbox className="size-5" />
          {t("emptySelected")}
        </div>
      );
    }

    return (
      <ul className="divide-y">
        {visibleChosen.map((row) => {
          const { label, detail } = describeRow(sourceType, row);

          return (
            <li key={row.key} className="flex items-center gap-3 px-4 py-2.5">
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="truncate text-sm font-medium">{label}</span>
                {detail ? <span className="truncate font-mono text-xs text-muted-foreground">{detail}</span> : null}
              </span>

              <Button
                type="button"
                size="icon"
                variant="ghost"
                aria-label={t("removeTarget", { name: label })}
                disabled={disabled}
                onClick={() => remove(row)}
              >
                <X />
              </Button>
            </li>
          );
        })}
      </ul>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3">
        <span className="text-sm tabular-nums">
          <span className="font-semibold">{chosen.length}</span>{" "}
          <span className="text-muted-foreground">{t("selectedOf", { max })}</span>
        </span>

        <div className="flex items-center gap-1 rounded-md border p-0.5" role="tablist">
          {(["all", "selected"] as const).map((option) => (
            <button
              key={option}
              type="button"
              role="tab"
              aria-selected={filter === option}
              onClick={() => setFilter(option)}
              className={cn(
                "rounded px-3 py-1 text-xs font-medium transition-colors",
                filter === option
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-surface-container-low",
              )}
            >
              {option === "all" ? t("all") : t("selected", { count: chosen.length })}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2 border-b p-4">
        {parent && filter === "all" ? (
          <div className="flex min-w-0 items-center gap-2 text-sm">
            <Button type="button" size="sm" variant="ghost" onClick={() => setParent(null)}>
              <ChevronLeft />
              {t("allSites")}
            </Button>
            <Badge variant="secondary" className="max-w-full truncate">
              {parent.label}
            </Badge>
          </div>
        ) : null}

        <div className="relative">
          <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id={id}
            className="pl-9"
            placeholder={parent ? t("searchLibraries") : t("searchPlaceholder")}
            aria-label={common("search")}
            value={search}
            disabled={disabled}
            onChange={(event) => setSearch(event.target.value)}
          />
        </div>

        {atMax ? <p className="text-xs text-muted-foreground">{t("limitReached", { max })}</p> : null}
      </div>

      <div ref={listRef} onScroll={loadMore} className="relative max-h-72 min-h-48 overflow-y-auto">
        {isFetching && !isLoading && !isFetchingNextPage && filter === "all" ? (
          <div className="sticky top-0 h-0.5 animate-pulse bg-primary" />
        ) : null}

        {filter === "all" ? allView() : selectedView()}
      </div>
    </div>
  );
}

"use client";

import { CircleAlert, Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
} from "@/components/ui/combobox";
import type { SourceType } from "@/lib/data-discovery";
import { cn } from "@/lib/utils";
import { useListDiscoveryPoliciesQuery } from "@/store/api/data-discovery-policies-api";

const SEARCH_DEBOUNCE = 350;
const PAGE_SIZE = 20;

export type PolicyRef = {
  id: number;
  name: string;
  source_type: SourceType;
  target_count?: number;
  rule_count?: number;
};

export function PolicyCombobox({
  id,
  value,
  onChange,
  sourceTypes,
  activeOnly,
  disabled,
  invalid,
  placeholder,
  className,
}: {
  id: string;
  value: PolicyRef | null;
  onChange: (next: PolicyRef | null) => void;
  sourceTypes?: readonly SourceType[];
  activeOnly?: boolean;
  disabled?: boolean;
  invalid?: boolean;
  placeholder?: string;
  className?: string;
}) {
  const t = useTranslations("data-discovery");
  const table = useTranslations("table");

  const [query, setQuery] = React.useState("");
  const [debounced, setDebounced] = React.useState("");

  React.useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), SEARCH_DEBOUNCE);

    return () => clearTimeout(timer);
  }, [query]);

  const { data, isFetching, isError, refetch } = useListDiscoveryPoliciesQuery({
    page: 1,
    pageSize: PAGE_SIZE,
    filters: {
      ...(debounced ? { search: debounced } : {}),
      ...(sourceTypes && sourceTypes.length > 0 ? { source_type: sourceTypes.join(",") } : {}),
      ...(activeOnly ? { status: "ACTIVE" } : {}),
    },
  });

  const rows = React.useMemo<PolicyRef[]>(
    () =>
      (data?.items ?? []).map((item) => ({
        id: item.id,
        name: item.name,
        source_type: item.source_type,
        target_count: item.target_count,
        rule_count: item.rule_count,
      })),
    [data],
  );

  const items = React.useMemo(() => {
    if (value && !rows.some((row) => row.id === value.id)) return [value, ...rows];

    return rows;
  }, [rows, value]);

  const total = data?.total ?? 0;

  return (
    <div className={cn("relative", className)}>
      <Combobox<PolicyRef>
        items={items}
        filter={null}
        value={value}
        onValueChange={(next) => onChange(next ?? null)}
        inputValue={query}
        onInputValueChange={setQuery}
        itemToStringLabel={(item) => item.name}
        isItemEqualToValue={(a, b) => a.id === b.id}
        disabled={disabled}
      >
        <div className="relative">
          <ComboboxInput
            id={id}
            placeholder={placeholder}
            aria-invalid={invalid || undefined}
            className={invalid ? "border-error" : undefined}
          />
          <ComboboxTrigger disabled={disabled} />
        </div>

        <ComboboxContent>
          {isFetching ? (
            <div className="flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground">
              <Loader2 className="size-3 animate-spin" />
              {t("scans.picker.searching")}
            </div>
          ) : null}

          {isError ? (
            <div className="flex flex-col items-center gap-2 px-3 py-4 text-sm">
              <CircleAlert className="size-4 text-error" />
              <span className="text-muted-foreground">{table("couldNotLoad")}</span>
              <Button type="button" size="sm" variant="outline" onClick={() => refetch()}>
                {table("tryAgain")}
              </Button>
            </div>
          ) : null}

          <ComboboxEmpty>{t("scans.picker.noPolicies")}</ComboboxEmpty>

          <ComboboxList>
            {items.map((item) => (
              <ComboboxItem key={item.id} value={item}>
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate font-medium">{item.name}</span>
                  <span className="truncate text-xs text-muted-foreground">
                    {t(`sourceType.${item.source_type}`)}
                  </span>
                </span>
              </ComboboxItem>
            ))}
          </ComboboxList>

          {total > items.length ? (
            <p className="border-t px-3 py-2 text-xs text-muted-foreground">
              {t("scans.picker.refineSearch", { shown: items.length, total })}
            </p>
          ) : null}
        </ComboboxContent>
      </Combobox>

      {value ? (
        <Badge variant="secondary" size="sm" className="mt-2">
          {t(`sourceType.${value.source_type}`)}
        </Badge>
      ) : null}
    </div>
  );
}

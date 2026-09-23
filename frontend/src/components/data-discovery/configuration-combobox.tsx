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
import type { ConfigurationType } from "@/lib/data-discovery";
import { cn } from "@/lib/utils";
import { useListDiscoveryConfigurationsQuery } from "@/store/api/data-discovery-configurations-api";

const SEARCH_DEBOUNCE = 350;
const PAGE_SIZE = 20;

export type ConfigurationRef = {
  id: number;
  name: string;
  configuration_type: ConfigurationType;
};

export function ConfigurationCombobox({
  id,
  value,
  onChange,
  configurationType,
  disabled,
  invalid,
  placeholder,
  className,
}: {
  id: string;
  value: ConfigurationRef | null;
  onChange: (next: ConfigurationRef | null) => void;
  configurationType?: ConfigurationType;
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

  const { data, isFetching, isError, refetch } = useListDiscoveryConfigurationsQuery({
    page: 1,
    pageSize: PAGE_SIZE,
    filters: {
      ...(debounced ? { search: debounced } : {}),
      ...(configurationType ? { configuration_type: configurationType } : {}),
      status: "ACTIVE",
    },
  });

  const rows = React.useMemo<ConfigurationRef[]>(
    () =>
      (data?.items ?? []).map((item) => ({
        id: item.id,
        name: item.name,
        configuration_type: item.configuration_type,
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
      <Combobox<ConfigurationRef>
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
              {t("configurations.searching")}
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

          <ComboboxEmpty>{t("policies.dialog.noConfigurations")}</ComboboxEmpty>

          <ComboboxList>
            {items.map((item) => (
              <ComboboxItem key={item.id} value={item}>
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate font-medium">{item.name}</span>
                  <span className="truncate text-xs text-muted-foreground">
                    {t(`configurationType.${item.configuration_type}`)}
                  </span>
                </span>
              </ComboboxItem>
            ))}
          </ComboboxList>

          {total > items.length ? (
            <p className="border-t px-3 py-2 text-xs text-muted-foreground">
              {t("configurations.refineSearch", { shown: items.length, total })}
            </p>
          ) : null}
        </ComboboxContent>
      </Combobox>

      {value ? (
        <Badge variant="secondary" size="sm" className="mt-2">
          {t(`configurationType.${value.configuration_type}`)}
        </Badge>
      ) : null}
    </div>
  );
}

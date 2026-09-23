"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { Field } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import {
  CONFIGURATION_TYPES,
  DISCOVERY_STATUSES,
  type ConfigurationType,
  type DiscoveryStatus,
} from "@/lib/data-discovery";

export type ConfigurationFilters = {
  search: string;
  configurationType: "" | ConfigurationType;
  status: "" | DiscoveryStatus;
};

export const EMPTY_CONFIGURATION_FILTERS: ConfigurationFilters = {
  search: "",
  configurationType: "",
  status: "",
};

export function toConfigurationFilterParams(
  filters: ConfigurationFilters,
): Record<string, string> {
  const params: Record<string, string> = {};

  if (filters.search.trim()) params.search = filters.search.trim();
  if (filters.configurationType) params.configuration_type = filters.configurationType;
  if (filters.status) params.status = filters.status;

  return params;
}

export function activeConfigurationFilterCount(filters: ConfigurationFilters) {
  return Object.keys(toConfigurationFilterParams(filters)).length;
}

export function ConfigurationFilterDrawer({
  open,
  onOpenChange,
  value,
  onApply,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: ConfigurationFilters;
  onApply: (next: ConfigurationFilters) => void;
}) {
  const t = useTranslations("data-discovery.configurations");
  const shared = useTranslations("data-discovery");
  const common = useTranslations("common");

  const [draft, setDraft] = React.useState(value);
  const wasOpen = React.useRef(open);

  React.useEffect(() => {
    if (open && !wasOpen.current) setDraft(value);
    wasOpen.current = open;
  }, [open, value]);

  const patch = React.useCallback((partial: Partial<ConfigurationFilters>) => {
    setDraft((current) => ({ ...current, ...partial }));
  }, []);

  const isEmpty = activeConfigurationFilterCount(draft) === 0;

  const typeOptions = CONFIGURATION_TYPES.map((type) => ({
    value: type,
    label: shared(`configurationType.${type}`),
  }));

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={t("filters.title")}
      description={t("filters.description")}
      width="lg"
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          onApply(draft);
          onOpenChange(false);
        }}
        className="flex flex-col gap-5"
      >
        <Field id="configuration-filter-name" label={t("filters.name")}>
          <Input
            id="configuration-filter-name"
            autoFocus
            placeholder={t("filters.namePlaceholder")}
            value={draft.search}
            onChange={(event) => patch({ search: event.target.value })}
          />
        </Field>

        <Field id="configuration-filter-type" label={t("filters.type")}>
          <Select
            id="configuration-filter-type"
            value={draft.configurationType}
            options={typeOptions}
            placeholder={t("filters.anyType")}
            onChange={(event) =>
              patch({ configurationType: event.target.value as "" | ConfigurationType })
            }
          />
        </Field>

        <Field id="configuration-filter-status" label={t("filters.status")}>
          <Select
            id="configuration-filter-status"
            value={draft.status}
            options={DISCOVERY_STATUSES.map((status) => ({
              value: status,
              label: status === "ACTIVE" ? t("filters.active") : t("filters.inactive"),
            }))}
            placeholder={t("filters.anyStatus")}
            onChange={(event) => patch({ status: event.target.value as "" | DiscoveryStatus })}
          />
        </Field>

        <div className="flex items-center justify-between gap-2 border-t pt-4">
          <Button
            type="button"
            variant="ghost"
            disabled={isEmpty}
            onClick={() => setDraft(EMPTY_CONFIGURATION_FILTERS)}
          >
            {t("filters.clearAll")}
          </Button>

          <div className="flex gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {common("cancel")}
            </Button>

            <Button type="submit">{t("filters.apply")}</Button>
          </div>
        </div>
      </form>
    </DrawerWrapper>
  );
}

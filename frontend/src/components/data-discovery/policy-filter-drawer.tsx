"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { Field } from "@/components/auth/auth-form";
import {
  ConfigurationCombobox,
  type ConfigurationRef,
} from "@/components/data-discovery/configuration-combobox";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import {
  DISCOVERY_STATUSES,
  SOURCE_TYPES,
  type DiscoveryStatus,
  type SourceType,
} from "@/lib/data-discovery";

export type PolicyFilters = {
  search: string;
  sourceType: "" | SourceType;
  configuration: ConfigurationRef | null;
  status: "" | DiscoveryStatus;
};

export const EMPTY_POLICY_FILTERS: PolicyFilters = {
  search: "",
  sourceType: "",
  configuration: null,
  status: "",
};

export function toPolicyFilterParams(filters: PolicyFilters): Record<string, string> {
  const params: Record<string, string> = {};

  if (filters.search.trim()) params.search = filters.search.trim();
  if (filters.sourceType) params.source_type = filters.sourceType;
  if (filters.configuration) params.configuration_id = String(filters.configuration.id);
  if (filters.status) params.status = filters.status;

  return params;
}

export function activePolicyFilterCount(filters: PolicyFilters) {
  return Object.keys(toPolicyFilterParams(filters)).length;
}

export function PolicyFilterDrawer({
  open,
  onOpenChange,
  value,
  onApply,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: PolicyFilters;
  onApply: (next: PolicyFilters) => void;
}) {
  const t = useTranslations("data-discovery.policies");
  const shared = useTranslations("data-discovery");
  const common = useTranslations("common");

  const [draft, setDraft] = React.useState(value);
  const wasOpen = React.useRef(open);

  React.useEffect(() => {
    if (open && !wasOpen.current) setDraft(value);
    wasOpen.current = open;
  }, [open, value]);

  const patch = React.useCallback((partial: Partial<PolicyFilters>) => {
    setDraft((current) => ({ ...current, ...partial }));
  }, []);

  const isEmpty = activePolicyFilterCount(draft) === 0;

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={t("filters.title")}
      description={t("filters.description")}
      width="xl"
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          onApply(draft);
          onOpenChange(false);
        }}
        className="flex flex-col gap-5"
      >
        <Field id="policy-filter-name" label={t("filters.name")}>
          <Input
            id="policy-filter-name"
            autoFocus
            placeholder={t("filters.namePlaceholder")}
            value={draft.search}
            onChange={(event) => patch({ search: event.target.value })}
          />
        </Field>

        <Field id="policy-filter-source" label={t("filters.sourceType")}>
          <Select
            id="policy-filter-source"
            value={draft.sourceType}
            options={SOURCE_TYPES.map((source) => ({
              value: source,
              label: shared(`sourceType.${source}`),
            }))}
            placeholder={t("filters.anySourceType")}
            onChange={(event) => patch({ sourceType: event.target.value as "" | SourceType })}
          />
        </Field>

        <Field id="policy-filter-configuration" label={t("filters.configuration")}>
          <ConfigurationCombobox
            id="policy-filter-configuration"
            value={draft.configuration}
            placeholder={t("filters.anyConfiguration")}
            onChange={(next) => patch({ configuration: next })}
          />
        </Field>

        <Field id="policy-filter-status" label={t("filters.status")}>
          <Select
            id="policy-filter-status"
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
            onClick={() => setDraft(EMPTY_POLICY_FILTERS)}
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

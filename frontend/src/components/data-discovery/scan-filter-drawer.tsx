"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { Field } from "@/components/auth/auth-form";
import { PolicyCombobox, type PolicyRef } from "@/components/data-discovery/policy-combobox";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { SCAN_STATUSES, type ScanStatus } from "@/lib/data-discovery";

export type ScanFilters = {
  policy: PolicyRef | null;
  status: "" | ScanStatus;
};

export const EMPTY_SCAN_FILTERS: ScanFilters = {
  policy: null,
  status: "",
};

export function toScanFilterParams(filters: ScanFilters): Record<string, string> {
  const params: Record<string, string> = {};

  if (filters.policy) params.policy_id = String(filters.policy.id);
  if (filters.status) params.status = filters.status;

  return params;
}

export function activeScanFilterCount(filters: ScanFilters) {
  return Object.keys(toScanFilterParams(filters)).length;
}

export function ScanFilterDrawer({
  open,
  onOpenChange,
  value,
  onApply,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: ScanFilters;
  onApply: (next: ScanFilters) => void;
}) {
  const t = useTranslations("data-discovery.scans");
  const status = useTranslations("status");
  const common = useTranslations("common");

  const [draft, setDraft] = React.useState(value);
  const wasOpen = React.useRef(open);

  React.useEffect(() => {
    if (open && !wasOpen.current) setDraft(value);
    wasOpen.current = open;
  }, [open, value]);

  const patch = React.useCallback((partial: Partial<ScanFilters>) => {
    setDraft((current) => ({ ...current, ...partial }));
  }, []);

  const isEmpty = activeScanFilterCount(draft) === 0;

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
        <Field id="scan-filter-policy" label={t("filters.policy")}>
          <PolicyCombobox
            id="scan-filter-policy"
            value={draft.policy}
            placeholder={t("filters.anyPolicy")}
            onChange={(next) => patch({ policy: next })}
          />
        </Field>

        <Field id="scan-filter-status" label={t("filters.status")}>
          <Select
            id="scan-filter-status"
            value={draft.status}
            options={SCAN_STATUSES.map((value) => ({
              value,
              label: status(value.toLowerCase()),
            }))}
            placeholder={t("filters.anyStatus")}
            onChange={(event) => patch({ status: event.target.value as "" | ScanStatus })}
          />
        </Field>

        <div className="flex items-center justify-between gap-2 border-t pt-4">
          <Button
            type="button"
            variant="ghost"
            disabled={isEmpty}
            onClick={() => setDraft(EMPTY_SCAN_FILTERS)}
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

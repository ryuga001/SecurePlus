"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { Field } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { FILE_RESULT_STATUSES, type FileResultStatus } from "@/lib/data-discovery";

export type FileResultFilters = {
  status: "" | FileResultStatus;
  withFindings: boolean;
};

export const EMPTY_FILE_RESULT_FILTERS: FileResultFilters = {
  status: "",
  withFindings: false,
};

export function toFileResultFilterParams(filters: FileResultFilters): Record<string, string> {
  const params: Record<string, string> = {};

  if (filters.status) params.status = filters.status;
  if (filters.withFindings) params.with_findings = "true";

  return params;
}

export function activeFileResultFilterCount(filters: FileResultFilters) {
  return Object.keys(toFileResultFilterParams(filters)).length;
}

export function FileResultFilterDrawer({
  open,
  onOpenChange,
  value,
  onApply,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: FileResultFilters;
  onApply: (next: FileResultFilters) => void;
}) {
  const t = useTranslations("data-discovery.scans.files");
  const status = useTranslations("status");
  const common = useTranslations("common");

  const [draft, setDraft] = React.useState(value);
  const wasOpen = React.useRef(open);

  React.useEffect(() => {
    if (open && !wasOpen.current) setDraft(value);
    wasOpen.current = open;
  }, [open, value]);

  const patch = React.useCallback((partial: Partial<FileResultFilters>) => {
    setDraft((current) => ({ ...current, ...partial }));
  }, []);

  const isEmpty = activeFileResultFilterCount(draft) === 0;

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
        <Field id="file-filter-status" label={t("filters.status")}>
          <Select
            id="file-filter-status"
            value={draft.status}
            options={FILE_RESULT_STATUSES.map((value) => ({
              value,
              label: status(value.toLowerCase()),
            }))}
            placeholder={t("filters.anyStatus")}
            onChange={(event) => patch({ status: event.target.value as "" | FileResultStatus })}
          />
        </Field>

        <div className="rounded-lg border bg-surface-container-low p-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="text-sm font-medium">{t("filters.withFindings")}</div>
              <div className="mt-1 text-xs text-muted-foreground">{t("filters.withFindingsHint")}</div>
            </div>

            <Switch
              checked={draft.withFindings}
              onCheckedChange={(checked) => patch({ withFindings: checked })}
            />
          </div>
        </div>

        <div className="flex items-center justify-between gap-2 border-t pt-4">
          <Button
            type="button"
            variant="ghost"
            disabled={isEmpty}
            onClick={() => setDraft(EMPTY_FILE_RESULT_FILTERS)}
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

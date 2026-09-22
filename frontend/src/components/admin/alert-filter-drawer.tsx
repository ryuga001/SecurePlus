"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { Field } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import {
  PolicyMultiSelect,
  type SelectedPolicy,
} from "@/components/admin/policy-multi-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import type { NotificationType, ScheduleType } from "@/store/api/alerts-api";

export type AlertFilters = {
  search: string;
  policies: SelectedPolicy[];
  notificationType: "" | NotificationType;
  scheduleType: "" | ScheduleType;
};

export const EMPTY_ALERT_FILTERS: AlertFilters = {
  search: "",
  policies: [],
  notificationType: "",
  scheduleType: "",
};

export function toAlertFilterParams(filters: AlertFilters): Record<string, string> {
  const params: Record<string, string> = {};

  if (filters.search.trim()) params.search = filters.search.trim();
  if (filters.policies.length > 0) {
    params.policy_ids = filters.policies.map((policy) => policy.id).join(",");
  }
  if (filters.notificationType) params.notification_type = filters.notificationType;
  if (filters.scheduleType) params.schedule_type = filters.scheduleType;

  return params;
}

export function activeAlertFilterCount(filters: AlertFilters) {
  return Object.keys(toAlertFilterParams(filters)).length;
}

export function AlertFilterDrawer({
  open,
  onOpenChange,
  value,
  onApply,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: AlertFilters;
  onApply: (next: AlertFilters) => void;
}) {
  const t = useTranslations("email-alert");
  const common = useTranslations("common");

  const [draft, setDraft] = React.useState(value);
  const wasOpen = React.useRef(open);

  React.useEffect(() => {
    if (open && !wasOpen.current) setDraft(value);
    wasOpen.current = open;
  }, [open, value]);

  const patch = React.useCallback((partial: Partial<AlertFilters>) => {
    setDraft((current) => ({ ...current, ...partial }));
  }, []);

  const notificationOptions = [
    { label: t("notificationType.email"), value: "EMAIL" },
    { label: t("notificationType.sms"), value: "SMS" },
  ];

  const scheduleOptions = [
    { label: t("schedule.realTime"), value: "REAL_TIME" },
    { label: t("schedule.custom"), value: "CUSTOM" },
  ];

  const isEmpty = activeAlertFilterCount(draft) === 0;

  function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onApply(draft);
    onOpenChange(false);
  }

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={t("filters.title")}
      description={t("filters.description")}
      width="2xl"
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-5">
        <Field id="alert-filter-name" label={t("filters.name")}>
          <Input
            id="alert-filter-name"
            autoFocus
            placeholder={t("filters.namePlaceholder")}
            value={draft.search}
            onChange={(event) => patch({ search: event.target.value })}
          />
        </Field>

        <Field id="alert-filter-notification" label={t("filters.notificationType")}>
          <Select
            id="alert-filter-notification"
            value={draft.notificationType}
            options={notificationOptions}
            placeholder={t("filters.anyNotificationType")}
            onChange={(event) =>
              patch({ notificationType: event.target.value as "" | NotificationType })
            }
          />
        </Field>

        <Field id="alert-filter-schedule" label={t("filters.schedule")}>
          <Select
            id="alert-filter-schedule"
            value={draft.scheduleType}
            options={scheduleOptions}
            placeholder={t("filters.anySchedule")}
            onChange={(event) =>
              patch({ scheduleType: event.target.value as "" | ScheduleType })
            }
          />
        </Field>

        <Field id="alert-filter-policies" label={t("filters.policies")}>
          <PolicyMultiSelect
            value={draft.policies}
            onChange={(policies) => patch({ policies })}
          />
        </Field>

        <div className="flex items-center justify-between gap-2 border-t pt-4">
          <Button
            type="button"
            variant="ghost"
            disabled={isEmpty}
            onClick={() => setDraft(EMPTY_ALERT_FILTERS)}
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

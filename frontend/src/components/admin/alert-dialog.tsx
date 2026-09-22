"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import {
  PolicyMultiSelect,
  type SelectedPolicy,
} from "@/components/admin/policy-multi-select";
import { Field, FormError } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmailTagInput } from "@/components/ui/email-tag-input";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { apiErrorCode, apiErrorMessage, apiErrorStatus } from "@/lib/api-error";
import { cn } from "@/lib/utils";
import {
  useCreateAlertMutation,
  useGetAlertQuery,
  useUpdateAlertMutation,
  type Alert,
  type AlertInput,
  type NotificationType,
  type ScheduleType,
} from "@/store/api/alerts-api";

function AlertForm({
  alert,
  canSave,
  onClose,
}: {
  alert: Alert | null;
  canSave: boolean;
  onClose: () => void;
}) {
  const t = useTranslations("email-alert");
  const common = useTranslations("common");

  const [name, setName] = React.useState(alert?.name ?? "");
  const [notificationType, setNotificationType] = React.useState<NotificationType>(
    alert?.notification_type ?? "EMAIL",
  );
  const [scheduleType, setScheduleType] = React.useState<ScheduleType>(
    alert?.schedule_type ?? "REAL_TIME",
  );
  const [policies, setPolicies] = React.useState<SelectedPolicy[]>(
    (alert?.policies ?? []).map((policy) => ({ id: policy.id, label: policy.name })),
  );
  const [targets, setTargets] = React.useState<string[]>(alert?.target ?? []);

  const [error, setError] = React.useState("");
  const [nameInvalid, setNameInvalid] = React.useState(false);
  const [policiesInvalid, setPoliciesInvalid] = React.useState(false);
  const [targetsInvalid, setTargetsInvalid] = React.useState(false);

  const nameRef = React.useRef<HTMLInputElement>(null);

  const [createAlert, { isLoading: creating }] = useCreateAlertMutation();
  const [updateAlert, { isLoading: updating }] = useUpdateAlertMutation();

  const pending = creating || updating;
  const smsBlocked = notificationType === "SMS";

  const notificationOptions: { value: NotificationType; label: string }[] = [
    { value: "EMAIL", label: t("notificationType.email") },
    { value: "SMS", label: t("notificationType.sms") },
  ];

  const scheduleOptions: { value: ScheduleType; label: string }[] = [
    { value: "REAL_TIME", label: t("schedule.realTime") },
    { value: "CUSTOM", label: t("schedule.custom") },
  ];

  const targetLabels = {
    add: t("dialog.addTarget"),
    empty: t("dialog.noTargets"),
    count: (count: number) => t("dialog.targetCount", { count }),
    remove: (value: string) => t("dialog.removeTarget", { value }),
    clear: t("dialog.clearTargets"),
    showAll: (count: number) => t("dialog.showAllTargets", { count }),
    showFewer: t("dialog.showFewerTargets"),
    invalid: (value: string) => t("dialog.invalidTarget", { value }),
    skippedInvalid: (count: number) => t("dialog.invalidTargetsSkipped", { count }),
    skippedDuplicate: (count: number) => t("dialog.duplicateTargetsSkipped", { count }),
  };

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmedName = name.trim();

    setError("");
    setNameInvalid(false);
    setPoliciesInvalid(false);
    setTargetsInvalid(false);

    if (!trimmedName) {
      setNameInvalid(true);
      setError(t("dialog.nameRequired"));
      nameRef.current?.focus();
      return;
    }

    if (policies.length === 0) {
      setPoliciesInvalid(true);
      setError(t("dialog.policiesRequired"));
      return;
    }

    if (targets.length === 0) {
      setTargetsInvalid(true);
      setError(t("dialog.targetsRequired"));
      return;
    }

    if (smsBlocked) {
      setError(t("dialog.smsUnsupported"));
      return;
    }

    const input: AlertInput = {
      name: trimmedName,
      schedule_type: scheduleType,
      notification_type: notificationType,
      target: targets,
      policy_ids: policies.map((policy) => policy.id),
    };

    try {
      if (alert) {
        await updateAlert({ id: alert.id, ...input }).unwrap();
        toast.success(t("dialog.updated"));
      } else {
        await createAlert(input).unwrap();
        toast.success(t("dialog.created"));
      }

      onClose();
    } catch (caught) {
      const status = apiErrorStatus(caught);
      const code = apiErrorCode(caught);

      if (status === 409) {
        setNameInvalid(true);
        setError(apiErrorMessage(caught, t("dialog.duplicateName")));
        nameRef.current?.focus();
        return;
      }

      if (code === "policy_not_found") {
        setPoliciesInvalid(true);
        setError(apiErrorMessage(caught, t("dialog.stalePolicySelection")));
        return;
      }

      if (code === "unsupported_notification_type") {
        setError(apiErrorMessage(caught, t("dialog.smsUnsupported")));
        return;
      }

      if (code === "invalid_target") {
        setTargetsInvalid(true);
        setError(apiErrorMessage(caught, t("dialog.targetsRequired")));
        return;
      }

      setError(apiErrorMessage(caught, t("dialog.saveFailed")));
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-5">
      <FormError message={error} />

      <Field id="alert-name" label={t("dialog.name")}>
        <Input
          id="alert-name"
          ref={nameRef}
          required
          autoFocus
          placeholder={t("dialog.namePlaceholder")}
          value={name}
          disabled={pending}
          aria-invalid={nameInvalid || undefined}
          onChange={(event) => {
            setName(event.target.value);
            setNameInvalid(false);
            setError("");
          }}
        />
      </Field>

      <Field
        id="alert-notification-type"
        label={t("dialog.notificationType")}
        hint={t("dialog.notificationTypeHint")}
      >
        <RadioGroup
          value={notificationType}
          disabled={pending}
          onValueChange={(next) => {
            setNotificationType(next as NotificationType);
            setError("");
          }}
          className="grid gap-2 sm:grid-cols-2"
        >
          {notificationOptions.map((option) => (
            <label
              key={option.value}
              className={cn(
                "flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition",
                notificationType === option.value
                  ? "border-primary bg-primary/5"
                  : "hover:bg-muted/50",
              )}
            >
              <RadioGroupItem value={option.value} />
              <span className="flex-1 text-sm font-medium">{option.label}</span>
              {option.value === "SMS" ? (
                <Badge variant="warning" size="sm">
                  {t("dialog.smsComingSoon")}
                </Badge>
              ) : null}
            </label>
          ))}
        </RadioGroup>
      </Field>

      {smsBlocked ? <FormError message={t("dialog.smsUnsupported")} /> : null}

      <Field id="alert-schedule" label={t("dialog.schedule")} hint={t("dialog.scheduleHint")}>
        <RadioGroup
          value={scheduleType}
          disabled={pending}
          onValueChange={(next) => {
            setScheduleType(next as ScheduleType);
            setError("");
          }}
          className="grid gap-2 sm:grid-cols-2"
        >
          {scheduleOptions.map((option) => (
            <label
              key={option.value}
              className={cn(
                "flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition",
                scheduleType === option.value
                  ? "border-primary bg-primary/5"
                  : "hover:bg-muted/50",
              )}
            >
              <RadioGroupItem value={option.value} />
              <span className="text-sm font-medium">{option.label}</span>
            </label>
          ))}
        </RadioGroup>
      </Field>

      {scheduleType === "CUSTOM" ? (
        <p className="-mt-3 text-xs text-muted-foreground">
          {t("dialog.customScheduleNotice")}
        </p>
      ) : null}

      <Field id="alert-policies" label={t("dialog.policies")} hint={t("dialog.policiesHint")}>
        <PolicyMultiSelect
          value={policies}
          disabled={pending}
          invalid={policiesInvalid}
          onChange={(next) => {
            setPolicies(next);
            setPoliciesInvalid(false);
            setError("");
          }}
        />
      </Field>

      <Field id="alert-targets" label={t("dialog.targets")} hint={t("dialog.targetsHint")}>
        <EmailTagInput
          id="alert-targets"
          values={targets}
          labels={targetLabels}
          placeholder={t("dialog.targetsPlaceholder")}
          disabled={pending}
          invalid={targetsInvalid}
          onChange={(next) => {
            setTargets(next);
            setTargetsInvalid(false);
            setError("");
          }}
        />
      </Field>

      <div className="flex shrink-0 justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
          {common("cancel")}
        </Button>

        <Button type="submit" disabled={pending || smsBlocked || !canSave}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          {alert ? t("dialog.saveChanges") : t("dialog.create")}
        </Button>
      </div>
    </form>
  );
}

function AlertLoader({
  alertId,
  canSave,
  onClose,
}: {
  alertId: string | null;
  canSave: boolean;
  onClose: () => void;
}) {
  const t = useTranslations("email-alert");
  const common = useTranslations("common");

  const { data, isLoading, isError } = useGetAlertQuery(alertId ?? "", {
    skip: alertId === null,
  });

  if (isLoading) {
    return (
      <div className="flex min-h-48 items-center justify-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        {t("dialog.oneMoment")}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex flex-col gap-4">
        <FormError message={t("dialog.loadFailed")} />

        <div className="flex justify-end">
          <Button type="button" variant="outline" onClick={onClose}>
            {common("close")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <AlertForm
      key={alertId ?? "create"}
      alert={data ?? null}
      canSave={canSave}
      onClose={onClose}
    />
  );
}

export function AlertDrawer({
  open,
  onOpenChange,
  alertId,
  canSave,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  alertId?: string | null;
  canSave: boolean;
}) {
  const t = useTranslations("email-alert");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={alertId == null ? t("dialog.addTitle") : t("dialog.editTitle")}
      description={t("dialog.description")}
      width="3xl"
    >
      <AlertLoader
        alertId={alertId ?? null}
        canSave={canSave}
        onClose={() => onOpenChange(false)}
      />
    </DrawerWrapper>
  );
}

"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import {
  ConfigurationCombobox,
  type ConfigurationRef,
} from "@/components/data-discovery/configuration-combobox";
import { FileTypePicker } from "@/components/data-discovery/file-type-picker";
import {
  POLICY_TABS,
  firstInvalidField,
  firstInvalidTab,
  parseBackendField,
  tabErrorCount,
  TAB_BY_FIELD,
  type PolicyFieldKey,
  type PolicyTab,
} from "@/components/data-discovery/policy-tabs";
import { RulePicker, type SelectedRule } from "@/components/data-discovery/rule-picker";
import { TargetBrowser } from "@/components/data-discovery/target-browser";
import {
  TargetListEditor,
  newTargetRow,
  type TargetRow,
} from "@/components/data-discovery/target-list-editor";
import { ConfirmDialog } from "@/components/dashboard/confirm-dialog";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiErrorCode, apiErrorMessage, apiErrorStatus } from "@/lib/api-error";
import {
  SOURCE_TYPES,
  configurationTypesFor,
  decodeTarget,
  encodeTarget,
  targetFields,
  type DiscoveryStatus,
  type SourceType,
} from "@/lib/data-discovery";
import {
  useCreateDiscoveryPolicyMutation,
  useGetDiscoveryPolicyQuery,
  useUpdateDiscoveryPolicyMutation,
  type DiscoveryPolicy,
} from "@/store/api/data-discovery-policies-api";

type FieldErrors = Partial<Record<PolicyFieldKey, string>>;
type TargetErrors = Record<number, Record<string, string>>;

function PolicyForm({
  policy,
  canSave,
  onClose,
}: {
  policy: DiscoveryPolicy | null;
  canSave: boolean;
  onClose: () => void;
}) {
  const t = useTranslations("data-discovery");
  const common = useTranslations("common");

  const [tab, setTab] = React.useState<PolicyTab>("general");
  const [mountedTabs, setMountedTabs] = React.useState<Set<PolicyTab>>(
    () => new Set<PolicyTab>(["general", "source"]),
  );

  const [name, setName] = React.useState(policy?.name ?? "");
  const [description, setDescription] = React.useState(policy?.description ?? "");
  const [status, setStatus] = React.useState<DiscoveryStatus>(policy?.status ?? "ACTIVE");
  const [sourceType, setSourceType] = React.useState<SourceType>(policy?.source_type ?? "AWS_S3");
  const [configuration, setConfiguration] = React.useState<ConfigurationRef | null>(
    policy
      ? {
          id: policy.configuration_id,
          name: policy.configuration_name,
          configuration_type: policy.configuration_type,
        }
      : null,
  );
  const [targets, setTargets] = React.useState<TargetRow[]>(() => {
    if (!policy || policy.target_list.length === 0) {
      return [newTargetRow(policy?.source_type ?? "AWS_S3")];
    }

    return policy.target_list.map((encoded) => ({
      key: crypto.randomUUID(),
      values: decodeTarget(policy.source_type, encoded),
    }));
  });
  const [fileTypes, setFileTypes] = React.useState<string[]>(policy?.file_types ?? []);
  const [rules, setRules] = React.useState<SelectedRule[]>(
    (policy?.rules ?? []).map((rule) => ({ id: rule.id, label: rule.name })),
  );

  const [fieldErrors, setFieldErrors] = React.useState<FieldErrors>({});
  const [targetErrors, setTargetErrors] = React.useState<TargetErrors>({});
  const [error, setError] = React.useState("");
  const [pendingSourceType, setPendingSourceType] = React.useState<SourceType | null>(null);
  const [focusField, setFocusField] = React.useState<PolicyFieldKey | null>(null);

  const focusRefs = React.useRef<Partial<Record<PolicyFieldKey, HTMLElement | null>>>({});

  const [createPolicy, { isLoading: creating }] = useCreateDiscoveryPolicyMutation();
  const [updatePolicy, { isLoading: updating }] = useUpdateDiscoveryPolicyMutation();

  const pending = creating || updating;

  React.useEffect(() => {
    setMountedTabs((current) => {
      if (current.has(tab)) return current;

      const next = new Set(current);
      next.add(tab);

      return next;
    });
  }, [tab]);

  React.useEffect(() => {
    if (!focusField) return;

    const node = focusRefs.current[focusField];
    const frame = requestAnimationFrame(() => {
      node?.focus?.();
      node?.scrollIntoView?.({ block: "center", behavior: "smooth" });
      setFocusField(null);
    });

    return () => cancelAnimationFrame(frame);
  }, [focusField, tab]);

  function jumpTo(field: PolicyFieldKey) {
    setTab(TAB_BY_FIELD[field]);
    setFocusField(field);
  }

  function applySourceType(next: SourceType) {
    setSourceType(next);
    setConfiguration(null);
    setTargets([newTargetRow(next)]);
    setTargetErrors({});
    setFieldErrors(({ configuration_id: _c, targets: _t, source_type: _s, ...rest }) => rest);
    setError("");
  }

  function requestSourceType(next: SourceType) {
    if (next === sourceType) return;

    const wouldLose =
      configuration !== null ||
      targets.some((row) => Object.values(row.values).some((value) => value.trim() !== ""));

    if (!wouldLose) {
      applySourceType(next);
      return;
    }

    setPendingSourceType(next);
  }

  function collect() {
    const errors: FieldErrors = {};
    const rowErrors: TargetErrors = {};

    if (!name.trim()) errors.name = t("policies.dialog.nameRequired");
    if (!configuration) errors.configuration_id = t("policies.dialog.configurationRequired");

    const fields = targetFields(sourceType);
    const seen = new Set<string>();

    targets.forEach((row, index) => {
      for (const field of fields) {
        const value = (row.values[field.key] ?? "").trim();

        if (field.required && !value) {
          rowErrors[index] = {
            ...rowErrors[index],
            [field.key]: t("policies.dialog.targetFieldRequired"),
          };
          continue;
        }

        if (value && field.pattern && !field.pattern.test(value)) {
          rowErrors[index] = {
            ...rowErrors[index],
            [field.key]: t("policies.dialog.invalidTargetField"),
          };
        }
      }

      const encoded = encodeTarget(sourceType, row.values);
      if (encoded && seen.has(encoded)) {
        rowErrors[index] = {
          ...rowErrors[index],
          [fields[0].key]: t("policies.dialog.duplicateTarget"),
        };
      }

      if (encoded) seen.add(encoded);
    });

    if (Object.keys(rowErrors).length > 0) {
      errors.targets = t("policies.dialog.targetsInvalid");
    }

    if (rules.length === 0) errors.rule_ids = t("policies.dialog.rulesRequired");

    return { errors, rowErrors };
  }

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setError("");

    const { errors, rowErrors } = collect();

    setFieldErrors(errors);
    setTargetErrors(rowErrors);

    if (Object.keys(errors).length > 0) {
      const target = firstInvalidField(errors);
      setError(t("policies.dialog.fixErrors", { count: Object.keys(errors).length }));
      if (target) jumpTo(target);

      return;
    }

    const input = {
      name: name.trim(),
      description: description.trim(),
      configuration_id: configuration!.id,
      source_type: sourceType,
      status,
      target_list: targets
        .map((row) => encodeTarget(sourceType, row.values))
        .filter(Boolean),
      file_types: fileTypes,
      rule_ids: rules.map((rule) => rule.id),
    };

    try {
      if (policy) {
        await updatePolicy({ id: policy.id, ...input }).unwrap();
        toast.success(t("policies.dialog.updated"));
      } else {
        await createPolicy(input).unwrap();
        toast.success(t("policies.dialog.created"));
      }

      onClose();
    } catch (caught) {
      const status = apiErrorStatus(caught);
      const code = apiErrorCode(caught);

      if (status === 409) {
        setFieldErrors({ name: t("policies.dialog.duplicateName") });
        setError(apiErrorMessage(caught, t("policies.dialog.duplicateName")));
        jumpTo("name");
        return;
      }

      if (code === "incompatible_source_type" || code === "discovery_configuration_not_found") {
        setFieldErrors({
          configuration_id: apiErrorMessage(caught, t("policies.dialog.incompatibleConfiguration")),
        });
        setError(apiErrorMessage(caught, t("policies.dialog.incompatibleConfiguration")));
        jumpTo("configuration_id");
        return;
      }

      if (code === "invalid_discovery_target") {
        setFieldErrors({ targets: apiErrorMessage(caught, t("policies.dialog.targetsInvalid")) });
        setError(apiErrorMessage(caught, t("policies.dialog.targetsInvalid")));
        jumpTo("targets");
        return;
      }

      if (code === "rule_not_found") {
        setFieldErrors({ rule_ids: apiErrorMessage(caught, t("policies.dialog.rulesRequired")) });
        setError(apiErrorMessage(caught, t("policies.dialog.rulesRequired")));
        jumpTo("rule_ids");
        return;
      }

      setError(apiErrorMessage(caught, t("policies.dialog.saveFailed")));
    }
  }

  const requiredConfigurationType = configurationTypesFor(sourceType)[0];

  return (
    <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
      <FormError message={error} />

      <Tabs
        value={tab}
        onValueChange={(next) => setTab(next as PolicyTab)}
        className="mt-4 flex min-h-0 flex-1 flex-col"
      >
        <TabsList className="grid w-full grid-cols-2 sm:grid-cols-4">
          {POLICY_TABS.map((item) => {
            const count = tabErrorCount(fieldErrors, item);

            return (
              <TabsTrigger key={item} value={item}>
                {t(`policies.dialog.tabs.${item}`)}
                {count > 0 ? (
                  <Badge variant="error" size="sm" className="ml-1.5 tabular-nums">
                    {count}
                  </Badge>
                ) : null}
              </TabsTrigger>
            );
          })}
        </TabsList>

        <div className="min-h-0 flex-1 overflow-y-auto py-6">
          <TabsContent value="general" keepMounted className="mt-0 flex flex-col gap-5">
            <Field id="policy-name" label={t("policies.dialog.name")} error={fieldErrors.name}>
              <Input
                id="policy-name"
                required
                autoFocus
                ref={(node) => {
                  focusRefs.current.name = node;
                }}
                placeholder={t("policies.dialog.namePlaceholder")}
                value={name}
                disabled={pending}
                aria-invalid={fieldErrors.name ? true : undefined}
                onChange={(event) => {
                  setName(event.target.value);
                  setFieldErrors(({ name: _removed, ...rest }) => rest);
                  setError("");
                }}
              />
            </Field>

            <Field id="policy-description" label={t("policies.dialog.description")}>
              <Input
                id="policy-description"
                value={description}
                disabled={pending}
                onChange={(event) => setDescription(event.target.value)}
              />
            </Field>

            <div className="rounded-lg border bg-surface-container-low p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <div className="text-sm font-medium">{t("policies.dialog.status")}</div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {t("policies.dialog.statusHint")}
                  </div>
                </div>

                <Switch
                  checked={status === "ACTIVE"}
                  disabled={pending}
                  onCheckedChange={(checked) => setStatus(checked ? "ACTIVE" : "INACTIVE")}
                />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="source" keepMounted className="mt-0 flex flex-col gap-5">
            <Field
              id="policy-source-type"
              label={t("policies.dialog.sourceType")}
              hint={t("policies.dialog.requiresConfigurationType", {
                type: t(`configurationType.${requiredConfigurationType}`),
              })}
            >
              <Select
                id="policy-source-type"
                value={sourceType}
                disabled={pending}
                options={SOURCE_TYPES.map((source) => ({
                  value: source,
                  label: t(`sourceType.${source}`),
                }))}
                onChange={(event) => requestSourceType(event.target.value as SourceType)}
              />
            </Field>

            <Field
              id="policy-configuration"
              label={t("policies.dialog.configuration")}
              error={fieldErrors.configuration_id}
            >
              <ConfigurationCombobox
                id="policy-configuration"
                value={configuration}
                configurationType={requiredConfigurationType}
                disabled={pending}
                invalid={Boolean(fieldErrors.configuration_id)}
                placeholder={t("policies.dialog.searchConfigurations")}
                onChange={(next) => {
                  setConfiguration(next);
                  setFieldErrors(({ configuration_id: _removed, ...rest }) => rest);
                  setError("");
                }}
              />
            </Field>

            <Field
              id="policy-target-browser"
              label={t("policies.dialog.browser.label")}
              hint={t("policies.dialog.browser.hint")}
            >
              <TargetBrowser
                key={`${sourceType}-${configuration?.id ?? "none"}`}
                id="policy-target-browser"
                configurationId={configuration?.id ?? null}
                sourceType={sourceType}
                value={targets}
                disabled={pending}
                onChange={(next) => {
                  setTargets(next);
                  setTargetErrors({});
                  setFieldErrors(({ targets: _removed, ...rest }) => rest);
                  setError("");
                }}
              />
            </Field>

            <Field
              id="policy-targets"
              label={t("policies.dialog.targets")}
              hint={t("policies.dialog.targetsHint")}
              error={fieldErrors.targets}
            >
              <TargetListEditor
                sourceType={sourceType}
                value={targets}
                onChange={(next) => {
                  setTargets(next);
                  setTargetErrors({});
                  setFieldErrors(({ targets: _removed, ...rest }) => rest);
                  setError("");
                }}
                errors={targetErrors}
                disabled={pending}
              />
            </Field>
          </TabsContent>

          <TabsContent
            value="fileTypes"
            keepMounted={mountedTabs.has("fileTypes")}
            className="mt-0 flex flex-col gap-4"
          >
            <p className="text-sm text-muted-foreground">{t("policies.dialog.fileTypesHint")}</p>
            <FileTypePicker value={fileTypes} onChange={setFileTypes} disabled={pending} />
          </TabsContent>

          <TabsContent
            value="rules"
            keepMounted={mountedTabs.has("rules")}
            className="mt-0 flex flex-col gap-4"
          >
            <p className="text-sm text-muted-foreground">{t("policies.dialog.rulesHint")}</p>
            <RulePicker
              value={rules}
              disabled={pending}
              invalid={Boolean(fieldErrors.rule_ids)}
              onChange={(next) => {
                setRules(next);
                setFieldErrors(({ rule_ids: _removed, ...rest }) => rest);
                setError("");
              }}
            />
          </TabsContent>
        </div>
      </Tabs>

      <div className="flex shrink-0 justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
          {common("cancel")}
        </Button>

        <Button type="submit" disabled={pending || !canSave}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          {policy ? t("policies.dialog.saveChanges") : t("policies.dialog.create")}
        </Button>
      </div>

      <ConfirmDialog
        open={pendingSourceType !== null}
        onOpenChange={(open) => {
          if (!open) setPendingSourceType(null);
        }}
        title={t("policies.dialog.changeSourceTitle")}
        description={t("policies.dialog.changeSourceDescription", { count: targets.length })}
        confirmLabel={t("policies.dialog.changeSourceConfirm")}
        onConfirm={() => {
          if (pendingSourceType) applySourceType(pendingSourceType);
          setPendingSourceType(null);
        }}
      />
    </form>
  );
}

function PolicyLoader({
  policyId,
  canSave,
  onClose,
}: {
  policyId: number | null;
  canSave: boolean;
  onClose: () => void;
}) {
  const t = useTranslations("data-discovery.policies.dialog");
  const common = useTranslations("common");

  const { data, isLoading, isError } = useGetDiscoveryPolicyQuery(policyId ?? 0, {
    skip: policyId === null,
  });

  if (isLoading) {
    return (
      <div className="flex min-h-48 items-center justify-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        {t("oneMoment")}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex flex-col gap-4">
        <FormError message={t("loadFailed")} />

        <div className="flex justify-end">
          <Button type="button" variant="outline" onClick={onClose}>
            {common("close")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <PolicyForm key={policyId ?? "create"} policy={data ?? null} canSave={canSave} onClose={onClose} />
  );
}

export function PolicyDrawer({
  open,
  onOpenChange,
  policyId,
  canSave,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  policyId?: number | null;
  canSave: boolean;
}) {
  const t = useTranslations("data-discovery.policies.dialog");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={policyId == null ? t("addTitle") : t("editTitle")}
      description={t("description")}
      width="3xl"
    >
      <PolicyLoader policyId={policyId ?? null} canSave={canSave} onClose={() => onOpenChange(false)} />
    </DrawerWrapper>
  );
}

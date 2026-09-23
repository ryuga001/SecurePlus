"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { ProviderFields } from "@/components/data-discovery/provider-fields";
import {
  TestConnectionPanel,
  type SaveStage,
  type TestState,
} from "@/components/data-discovery/test-connection-panel";
import { Field, FormError } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Switch } from "@/components/ui/switch";
import { apiErrorCode, apiErrorMessage, apiErrorStatus } from "@/lib/api-error";
import {
  CONFIGURATION_TYPES,
  CONFIGURATION_TYPE_META,
  credentialFields,
  secretFieldKeys,
  supportedSourceTypes,
  type ConfigurationType,
  type DiscoveryStatus,
} from "@/lib/data-discovery";
import type { ServiceAccountKey } from "@/lib/gcp-service-account";
import { cn } from "@/lib/utils";
import {
  useCreateDiscoveryConfigurationMutation,
  useGetDiscoveryConfigurationQuery,
  useTestDiscoveryConfigurationMutation,
  useUpdateDiscoveryConfigurationMutation,
  type DiscoveryConfiguration,
} from "@/store/api/data-discovery-configurations-api";

function initialValues(configuration: DiscoveryConfiguration | null) {
  if (!configuration) return {};

  const values: Record<string, string> = {};
  for (const [key, value] of Object.entries(configuration.config)) {
    values[key] = value;
  }

  return values;
}

function ConfigurationForm({
  configuration,
  canSave,
  onClose,
}: {
  configuration: DiscoveryConfiguration | null;
  canSave: boolean;
  onClose: () => void;
}) {
  const t = useTranslations("data-discovery");
  const common = useTranslations("common");

  const [type, setType] = React.useState<ConfigurationType | null>(
    configuration?.configuration_type ?? null,
  );
  const [name, setName] = React.useState(configuration?.name ?? "");
  const [description, setDescription] = React.useState(configuration?.description ?? "");
  const [status, setStatus] = React.useState<DiscoveryStatus>(configuration?.status ?? "ACTIVE");
  const [values, setValues] = React.useState<Record<string, string>>(initialValues(configuration));
  const [keyFile, setKeyFile] = React.useState<ServiceAccountKey | null>(null);
  const [replacing, setReplacing] = React.useState<Set<string>>(new Set());
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});
  const [error, setError] = React.useState("");
  const [stage, setStage] = React.useState<SaveStage>("idle");
  const [test, setTest] = React.useState<TestState>({ status: "idle" });
  const [credentialsDirty, setCredentialsDirty] = React.useState(!configuration);

  const nameRef = React.useRef<HTMLInputElement>(null);

  const [createConfiguration] = useCreateDiscoveryConfigurationMutation();
  const [updateConfiguration] = useUpdateDiscoveryConfigurationMutation();
  const [testConfiguration] = useTestDiscoveryConfigurationMutation();

  const pending = stage !== "idle";
  const hasStoredSecret = Boolean(configuration?.has_credential);

  const storedDerived =
    configuration && configuration.config.projectId
      ? {
          projectId: configuration.config.projectId,
          clientEmail: configuration.config.clientEmail ?? "",
        }
      : undefined;

  function changeValue(key: string, value: string) {
    setValues((current) => ({ ...current, [key]: value }));
    setFieldErrors(({ [key]: _removed, ...rest }) => rest);
    setError("");
    setCredentialsDirty(true);
    setTest({ status: "idle" });
  }

  function toggleReplace(key: string, next: boolean) {
    setReplacing((current) => {
      const updated = new Set(current);
      if (next) updated.add(key);
      else updated.delete(key);

      return updated;
    });

    if (!next) {
      setValues((current) => ({ ...current, [key]: "" }));
      setKeyFile(null);
    }

    setCredentialsDirty(true);
    setTest({ status: "idle" });
    setError("");
  }

  function buildPayload(selected: ConfigurationType) {
    const config: Record<string, string> = {};
    const secret: Record<string, string> = {};

    for (const field of credentialFields(selected)) {
      if (field.kind === "text") {
        const value = (values[field.key] ?? "").trim();
        if (value) config[field.key] = value;
        continue;
      }

      if (field.kind === "secret") {
        if (hasStoredSecret && !replacing.has(field.key)) continue;

        const value = (values[field.key] ?? "").trim();
        if (value) secret[field.key] = value;
        continue;
      }

      if (hasStoredSecret && !replacing.has(field.key)) continue;
      if (keyFile) secret[field.key] = keyFile.raw;
    }

    return { config, secret };
  }

  function validate(selected: ConfigurationType) {
    const errors: Record<string, string> = {};

    if (!name.trim()) {
      errors.name = t("configurations.dialog.nameRequired");
    }

    for (const field of credentialFields(selected)) {
      const secretField = field.kind !== "text";
      const stored = hasStoredSecret && !replacing.has(field.key);

      if (secretField && stored) continue;

      if (field.kind === "gcpServiceAccountKey") {
        if (field.required && !keyFile) {
          errors[field.key] = t("configurations.dialog.requiredField", {
            field: t(`fields.${field.labelKey}`),
          });
        }
        continue;
      }

      const value = (values[field.key] ?? "").trim();

      if (field.required && !value) {
        errors[field.key] = t("configurations.dialog.requiredField", {
          field: t(`fields.${field.labelKey}`),
        });
        continue;
      }

      if (value && field.pattern && !field.pattern.test(value)) {
        errors[field.key] = t("configurations.dialog.invalidFormat", {
          field: t(`fields.${field.labelKey}`),
        });
      }
    }

    return errors;
  }

  async function runTest() {
    if (!type) return;

    const errors = validate(type);
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      setError(Object.values(errors)[0]);
      return;
    }

    setTest({ status: "running" });

    const { config, secret } = buildPayload(type);

    try {
      await testConfiguration({
        configuration_id: configuration?.id,
        configuration_type: type,
        config,
        secret,
      }).unwrap();

      setTest({ status: "passed" });
      setCredentialsDirty(false);
    } catch (caught) {
      setTest({
        status: "failed",
        message: apiErrorMessage(caught, t("configurations.dialog.connectionFailed")),
        code: apiErrorCode(caught),
      });
    }
  }

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setError("");
    setFieldErrors({});
    setStage("validating");

    if (!type) {
      setStage("idle");
      setError(t("configurations.dialog.typeRequired"));
      return;
    }

    const errors = validate(type);
    if (Object.keys(errors).length > 0) {
      setStage("idle");
      setFieldErrors(errors);
      setError(Object.values(errors)[0]);
      if (errors.name) nameRef.current?.focus();
      return;
    }

    const { config, secret } = buildPayload(type);

    const needsTest = credentialsDirty || test.status !== "passed";

    if (needsTest) {
      setStage("testing");

      try {
        await testConfiguration({
          configuration_id: configuration?.id,
          configuration_type: type,
          config,
          secret,
        }).unwrap();

        setTest({ status: "passed" });
        setCredentialsDirty(false);
      } catch (caught) {
        setStage("idle");
        setTest({
          status: "failed",
          message: apiErrorMessage(caught, t("configurations.dialog.connectionFailed")),
          code: apiErrorCode(caught),
        });
        setError(apiErrorMessage(caught, t("configurations.dialog.connectionFailed")));
        return;
      }
    }

    setStage("saving");

    try {
      if (configuration) {
        await updateConfiguration({
          id: configuration.id,
          name: name.trim(),
          description: description.trim(),
          status,
          config,
          secret,
        }).unwrap();

        toast.success(t("configurations.dialog.updated"));
      } else {
        await createConfiguration({
          name: name.trim(),
          description: description.trim(),
          configuration_type: type,
          status,
          config,
          secret,
        }).unwrap();

        toast.success(t("configurations.dialog.created"));
      }

      onClose();
    } catch (caught) {
      if (apiErrorStatus(caught) === 409) {
        setFieldErrors({ name: t("configurations.dialog.duplicateName") });
        setError(apiErrorMessage(caught, t("configurations.dialog.duplicateName")));
        nameRef.current?.focus();
      } else {
        setError(apiErrorMessage(caught, t("configurations.dialog.saveFailed")));
      }
    } finally {
      setStage("idle");
    }
  }

  const canTest = Boolean(type) && !pending;

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-5">
      <FormError message={error} />

      {configuration ? (
        <Field id="configuration-type" label={t("configurations.dialog.type")} hint={t("configurations.dialog.typeLockedHint")}>
          <div className="flex items-center gap-3 rounded-lg border bg-surface-container-low p-3">
            <span className="flex size-8 items-center justify-center rounded-md bg-surface-container font-mono text-xs">
              {CONFIGURATION_TYPE_META[configuration.configuration_type].monogram}
            </span>
            <span className="text-sm font-medium">
              {t(`configurationType.${configuration.configuration_type}`)}
            </span>
          </div>
        </Field>
      ) : (
        <Field id="configuration-type" label={t("configurations.dialog.type")}>
          <RadioGroup
            value={type ?? ""}
            disabled={pending}
            onValueChange={(next) => {
              setType(next as ConfigurationType);
              setValues({});
              setKeyFile(null);
              setFieldErrors({});
              setError("");
              setTest({ status: "idle" });
              setCredentialsDirty(true);
            }}
            className="grid gap-2 sm:grid-cols-2"
          >
            {CONFIGURATION_TYPES.map((option) => (
              <label
                key={option}
                className={cn(
                  "flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition",
                  type === option ? "border-primary bg-primary/5" : "hover:bg-muted/50",
                )}
              >
                <RadioGroupItem value={option} />
                <span className="min-w-0">
                  <span className="block text-sm font-medium">{t(`configurationType.${option}`)}</span>
                  <span className="mt-0.5 block text-xs text-muted-foreground">
                    {supportedSourceTypes(option)
                      .map((source) => t(`sourceType.${source}`))
                      .join(" · ")}
                  </span>
                </span>
              </label>
            ))}
          </RadioGroup>
        </Field>
      )}

      <Field id="configuration-name" label={t("configurations.dialog.name")} error={fieldErrors.name}>
        <Input
          id="configuration-name"
          ref={nameRef}
          required
          autoFocus
          placeholder={t("configurations.dialog.namePlaceholder")}
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

      <Field id="configuration-description" label={t("configurations.dialog.description")}>
        <Input
          id="configuration-description"
          value={description}
          disabled={pending}
          onChange={(event) => setDescription(event.target.value)}
        />
      </Field>

      {type ? (
        <ProviderFields
          configurationType={type}
          values={values}
          onValueChange={changeValue}
          keyFile={keyFile}
          onKeyFileChange={(next) => {
            setKeyFile(next);
            setFieldErrors(({ serviceAccountKey: _removed, ...rest }) => rest);
            setError("");
            setCredentialsDirty(true);
            setTest({ status: "idle" });
          }}
          onKeyFileError={(message) => {
            setFieldErrors((current) => ({ ...current, serviceAccountKey: message }));
            setError(message);
          }}
          replacing={replacing}
          onReplaceToggle={toggleReplace}
          hasStoredSecret={hasStoredSecret}
          storedDerived={storedDerived}
          errors={fieldErrors}
          disabled={pending}
        />
      ) : null}

      <div className="rounded-lg border bg-surface-container-low p-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-medium">{t("configurations.dialog.status")}</div>
            <div className="mt-1 text-xs text-muted-foreground">
              {t("configurations.dialog.statusHint")}
            </div>
          </div>

          <Switch
            checked={status === "ACTIVE"}
            disabled={pending}
            onCheckedChange={(checked) => setStatus(checked ? "ACTIVE" : "INACTIVE")}
          />
        </div>
      </div>

      {type ? (
        <TestConnectionPanel stage={stage} test={test} canTest={canTest} onTest={runTest} />
      ) : null}

      <div className="flex shrink-0 justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
          {common("cancel")}
        </Button>

        <Button type="submit" disabled={pending || !canSave}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          {configuration
            ? t("configurations.dialog.saveChanges")
            : t("configurations.dialog.create")}
        </Button>
      </div>
    </form>
  );
}

function ConfigurationLoader({
  configurationId,
  canSave,
  onClose,
}: {
  configurationId: number | null;
  canSave: boolean;
  onClose: () => void;
}) {
  const t = useTranslations("data-discovery.configurations.dialog");
  const common = useTranslations("common");

  const { data, isLoading, isError } = useGetDiscoveryConfigurationQuery(configurationId ?? 0, {
    skip: configurationId === null,
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
    <ConfigurationForm
      key={configurationId ?? "create"}
      configuration={data ?? null}
      canSave={canSave}
      onClose={onClose}
    />
  );
}

export function ConfigurationDrawer({
  open,
  onOpenChange,
  configurationId,
  canSave,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  configurationId?: number | null;
  canSave: boolean;
}) {
  const t = useTranslations("data-discovery.configurations.dialog");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={configurationId == null ? t("addTitle") : t("editTitle")}
      description={t("description")}
      width="2xl"
    >
      <ConfigurationLoader
        configurationId={configurationId ?? null}
        canSave={canSave}
        onClose={() => onOpenChange(false)}
      />
    </DrawerWrapper>
  );
}

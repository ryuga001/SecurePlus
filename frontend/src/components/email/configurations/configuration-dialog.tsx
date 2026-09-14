"use client";

import { ArrowLeft, Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { FormError, StepIndicator } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { apiErrorMessage, apiErrorStatus } from "@/lib/api-error";
import {
  useCreateEmailConfigurationMutation,
  useUpdateEmailConfigurationMutation,
  type EmailConfiguration,
  type EmailProvider,
} from "@/store/api/email-configurations-api";

import { ConfigurationFields, type ConfigurationFormValues } from "./configuration-fields";
import { CredentialsPanel } from "./credentials-panel";
import { providerMeta } from "./provider-meta";
import { ProviderStep } from "./provider-step";

const DOMAIN_PATTERN = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*\.[a-z]{2,}$/i;

type Step = "provider" | "details";

function ConfigurationDialogBody({
  configuration,
  onClose,
}: {
  configuration: EmailConfiguration | null;
  onClose: () => void;
}) {
  const t = useTranslations("configurations");
  const common = useTranslations("common");

  const editing = configuration !== null;

  const [step, setStep] = React.useState<Step>(editing ? "details" : "provider");
  const [provider, setProvider] = React.useState<EmailProvider | null>(
    configuration?.provider ?? null
  );
  const [configurationId, setConfigurationId] = React.useState<number | null>(
    configuration?.id ?? null
  );
  const [values, setValues] = React.useState<ConfigurationFormValues>({
    name: configuration?.name ?? "",
    domain: configuration?.domain ?? "",
  });
  const [error, setError] = React.useState("");
  const [domainInvalid, setDomainInvalid] = React.useState(false);

  const [createConfiguration, { isLoading: creating }] = useCreateEmailConfigurationMutation();
  const [updateConfiguration, { isLoading: updating }] = useUpdateEmailConfigurationMutation();

  const saved = configurationId !== null;
  const pending = creating || updating;

  function update<K extends keyof ConfigurationFormValues>(
    field: K,
    value: ConfigurationFormValues[K]
  ) {
    setValues((current) => ({ ...current, [field]: value }));
    setError("");

    if (field === "domain") setDomainInvalid(false);
  }

  function selectProvider(next: EmailProvider) {
    setProvider(next);
    setStep("details");
  }

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!provider) return;

    const name = values.name.trim();
    const domain = values.domain.trim().toLowerCase();

    if (!name) {
      setError(t("dialog.nameRequired"));
      return;
    }

    if (!DOMAIN_PATTERN.test(domain)) {
      setDomainInvalid(true);
      setError(t("dialog.domainRequired"));
      return;
    }

    setError("");
    setDomainInvalid(false);

    const input = { name, domain, provider };

    try {
      if (configurationId === null) {
        const created = await createConfiguration(input).unwrap();
        setConfigurationId(created.id);
        setValues({ name: created.name, domain: created.domain });
        toast.success(t("dialog.created"));
      } else {
        const updated = await updateConfiguration({ id: configurationId, ...input }).unwrap();
        setValues({ name: updated.name, domain: updated.domain });
        toast.success(t("dialog.updated"));
      }
    } catch (caught) {
      setDomainInvalid(apiErrorStatus(caught) === 409);
      setError(apiErrorMessage(caught, t("dialog.saveFailed")));
    }
  }

  const meta = provider ? providerMeta[provider] : null;

  return (
    <>
      <DialogTitle>{editing ? t("dialog.editTitle") : t("dialog.addTitle")}</DialogTitle>
      <DialogDescription>
        {step === "provider"
          ? t("dialog.providerStep")
          : t("dialog.detailsStep")}
      </DialogDescription>

      {editing ? null : (
        <div className="mt-5">
          <StepIndicator current={step === "provider" ? 1 : 2} total={2} />
        </div>
      )}

      {step === "provider" ? (
        <>
          <ProviderStep value={provider} onSelect={selectProvider} />

          <div className="mt-6 flex justify-end border-t pt-4">
            <Button type="button" variant="outline" onClick={onClose}>
              {common("cancel")}
            </Button>
          </div>
        </>
      ) : (
        <form onSubmit={onSubmit} className="mt-2 flex flex-col gap-5">
          {meta ? (
            <div className="flex items-center gap-2.5 border p-3">
              <span className="flex size-8 items-center justify-center bg-primary/10 font-heading text-sm font-semibold text-primary">
                {meta.monogram}
              </span>
              <span className="text-sm font-medium">{meta.label}</span>
            </div>
          ) : null}

          <FormError message={error} />

          <ConfigurationFields
            values={values}
            onChange={update}
            domainInvalid={domainInvalid}
            disabled={pending}
          />

          <CredentialsPanel
            configurationId={configurationId}
            initialDkimPublicKey={configuration?.dkim_public_key}
          />

          <div className="flex justify-end gap-2 border-t pt-4">
            {!editing && !saved ? (
              <Button type="button" variant="outline" onClick={() => setStep("provider")}>
                <ArrowLeft />
                {t("dialog.back")}
              </Button>
            ) : (
              <Button type="button" variant="outline" onClick={onClose}>
                {t("dialog.done")}
              </Button>
            )}

            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="animate-spin" /> : null}
              {saved ? t("dialog.saveChanges") : t("dialog.create")}
            </Button>
          </div>
        </form>
      )}
    </>
  );
}

export function EmailConfigurationDialog({
  open,
  onOpenChange,
  configuration,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  configuration?: EmailConfiguration | null;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100svh-4rem)] max-w-xl overflow-y-auto">
        <ConfigurationDialogBody
          configuration={configuration ?? null}
          onClose={() => onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  );
}

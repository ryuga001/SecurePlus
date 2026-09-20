"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { FormError } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { apiErrorMessage, apiErrorStatus } from "@/lib/api-error";
import {
  useCreateEmailConfigurationMutation,
  useGeneratePendingAccessTokenMutation,
  useGeneratePendingDkimMutation,
  useUpdateEmailConfigurationMutation,
  type EmailConfiguration,
  type EmailProvider,
} from "@/store/api/email-configurations-api";

import {
  ConfigurationFields,
  type ConfigurationFormValues,
} from "./configuration-fields";
import { CredentialsPanel } from "./credentials-panel";
import { ProviderStep } from "./provider-step";

const DOMAIN_PATTERN = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*\.[a-z]{2,}$/i;

function ConfigurationDrawerBody({
  configuration,
  onClose,
}: {
  configuration: EmailConfiguration | null;
  onClose: () => void;
}) {
  const t = useTranslations("configurations");
  const common = useTranslations("common");

  const editing = configuration !== null;

  const [provider, setProvider] = React.useState<EmailProvider | null>(
    configuration?.provider ?? null,
  );
  const [values, setValues] = React.useState<ConfigurationFormValues>({
    name: configuration?.name ?? "",
    domain: configuration?.domain ?? "",
  });

  const [dkimPublicKey, setDkimPublicKey] = React.useState(
    configuration?.dkim_public_key ?? "",
  );
  const [dkimPrivateKey, setDkimPrivateKey] = React.useState("");
  const [recordName, setRecordName] = React.useState("");
  const [accessToken, setAccessToken] = React.useState("");
  const [expiresAt, setExpiresAt] = React.useState("");

  const [domainInvalid, setDomainInvalid] = React.useState(false);
  const [error, setError] = React.useState("");

  const [createConfiguration, { isLoading: creating }] =
    useCreateEmailConfigurationMutation();
  const [updateConfiguration, { isLoading: updating }] =
    useUpdateEmailConfigurationMutation();
  const [generatePendingDkim, { isLoading: dkimPending }] =
    useGeneratePendingDkimMutation();
  const [generatePendingAccessToken, { isLoading: tokenPending }] =
    useGeneratePendingAccessTokenMutation();

  const pending = creating || updating;

  function update<K extends keyof ConfigurationFormValues>(
    field: K,
    value: ConfigurationFormValues[K],
  ) {
    setValues((current) => ({ ...current, [field]: value }));
    setError("");

    if (field === "domain") setDomainInvalid(false);
  }

  async function handleGenerateDkim() {
    try {
      setError("");
      const result = await generatePendingDkim({
        domain: values.domain.trim().toLowerCase() || undefined,
      }).unwrap();
      setDkimPublicKey(result.dkim_public_key);
      setDkimPrivateKey(result.dkim_private_key);
      setRecordName(result.record_name);
      toast.success(t("dialog.dkimGenerated"));
    } catch (caught) {
      setError(apiErrorMessage(caught, t("dialog.saveFailed")));
    }
  }

  async function handleGenerateAccessToken() {
    const domain = values.domain.trim().toLowerCase();

    if (!DOMAIN_PATTERN.test(domain)) {
      setDomainInvalid(true);
      setError(t("dialog.domainRequired"));
      return;
    }

    try {
      setError("");
      const result = await generatePendingAccessToken({ domain }).unwrap();
      setAccessToken(result.access_token);
      setExpiresAt(result.expires_at);
      toast.success(t("dialog.tokenGenerated"));
    } catch (caught) {
      setError(apiErrorMessage(caught, t("dialog.saveFailed")));
    }
  }

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!provider) {
      setError(t("dialog.providerRequired"));
      return;
    }

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

    const input = {
      name,
      domain,
      provider,
      dkim_public_key: dkimPublicKey.trim() || undefined,
      dkim_private_key: dkimPrivateKey.trim() || undefined,
      access_token: accessToken.trim() || undefined,
    };

    try {
      if (editing) {
        await updateConfiguration({
          id: configuration!.id,
          ...input,
        }).unwrap();
        toast.success(t("dialog.updated"));
      } else {
        await createConfiguration(input).unwrap();
        toast.success(t("dialog.created"));
      }

      onClose();
    } catch (caught) {
      setDomainInvalid(apiErrorStatus(caught) === 409);
      setError(apiErrorMessage(caught, t("dialog.saveFailed")));
    }
  }

  return (
    <form
      onSubmit={onSubmit}
      className="flex min-h-0 flex-1 flex-col gap-5"
    >
      <FormError message={error} />

      <div>
        <div className="mb-2 text-sm font-medium">
          {t("dialog.provider")}
        </div>

        <ProviderStep
          value={provider}
          onSelect={(next) => {
            setProvider(next);
            setError("");
          }}
        />
      </div>

      <div className="border-t pt-5">
        <ConfigurationFields
          values={values}
          onChange={update}
          domainInvalid={domainInvalid}
          disabled={pending}
        />
      </div>

      <CredentialsPanel
        dkimPublicKey={dkimPublicKey}
        recordName={recordName}
        accessToken={accessToken}
        expiresAt={expiresAt}
        onGenerateDkim={handleGenerateDkim}
        onGenerateAccessToken={handleGenerateAccessToken}
        dkimPending={dkimPending}
        tokenPending={tokenPending}
      />

      <div className="flex shrink-0 justify-end gap-2 border-t pt-4">
        <Button
          type="button"
          variant="outline"
          disabled={pending}
          onClick={onClose}
        >
          {common("cancel")}
        </Button>

        <Button type="submit" disabled={pending}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          {editing ? t("dialog.saveChanges") : t("dialog.create")}
        </Button>
      </div>
    </form>
  );
}

export function EmailConfigurationDrawer({
  open,
  onOpenChange,
  configuration,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  configuration?: EmailConfiguration | null;
}) {
  const t = useTranslations("configurations");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={
        configuration
          ? t("dialog.editTitle")
          : t("dialog.addTitle")
      }
      description={t("dialog.description")}
      width="2xl"
    >
      <ConfigurationDrawerBody
        key={configuration ? `edit-${configuration.id}` : "create"}
        configuration={configuration ?? null}
        onClose={() => onOpenChange(false)}
      />
    </DrawerWrapper>
  );
}
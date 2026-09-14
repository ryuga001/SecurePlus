"use client";

import { useTranslations } from "next-intl";
import { Field } from "@/components/auth/auth-form";
import { Input } from "@/components/ui/input";

export type ConfigurationFormValues = {
  name: string;
  domain: string;
};

export function ConfigurationFields({
  values,
  onChange,
  domainInvalid,
  disabled,
}: {
  values: ConfigurationFormValues;
  onChange: <K extends keyof ConfigurationFormValues>(
    field: K,
    value: ConfigurationFormValues[K]
  ) => void;
  domainInvalid?: boolean;
  disabled?: boolean;
}) {
  const t = useTranslations("configurations");

  return (
    <div className="flex flex-col gap-4">
      <Field id="configuration-name" label={t("dialog.name")}>
        <Input
          id="configuration-name"
          required
          autoFocus
          placeholder={t("dialog.namePlaceholder")}
          value={values.name}
          disabled={disabled}
          onChange={(event) => onChange("name", event.target.value)}
        />
      </Field>

      <Field
        id="configuration-domain"
        label={t("dialog.domain")}
        hint={t("dialog.domainHint")}
      >
        <Input
          id="configuration-domain"
          required
          spellCheck={false}
          autoComplete="off"
          placeholder="example.com"
          value={values.domain}
          disabled={disabled}
          aria-invalid={domainInvalid || undefined}
          onChange={(event) => onChange("domain", event.target.value)}
        />
      </Field>
    </div>
  );
}

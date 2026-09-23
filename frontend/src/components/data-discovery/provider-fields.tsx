"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import { Field, PasswordInput } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { JsonKeyFileInput } from "@/components/ui/json-key-file-input";
import { credentialFields, type ConfigurationType } from "@/lib/data-discovery";
import type { ServiceAccountKey } from "@/lib/gcp-service-account";
import { cn } from "@/lib/utils";

export function ProviderFields({
  configurationType,
  values,
  onValueChange,
  keyFile,
  onKeyFileChange,
  onKeyFileError,
  replacing,
  onReplaceToggle,
  hasStoredSecret,
  storedDerived,
  errors,
  disabled,
}: {
  configurationType: ConfigurationType;
  values: Record<string, string>;
  onValueChange: (key: string, value: string) => void;
  keyFile: ServiceAccountKey | null;
  onKeyFileChange: (value: ServiceAccountKey | null) => void;
  onKeyFileError: (message: string) => void;
  replacing: ReadonlySet<string>;
  onReplaceToggle: (key: string, replacing: boolean) => void;
  hasStoredSecret: boolean;
  storedDerived?: { projectId: string; clientEmail: string };
  errors: Record<string, string>;
  disabled?: boolean;
}) {
  const t = useTranslations("data-discovery");

  return (
    <>
      {credentialFields(configurationType).map((field) => {
        const fieldId = `configuration-${field.key}`;
        const label = t(`fields.${field.labelKey}`);
        const hint = field.hintKey ? t(`fieldHints.${field.hintKey}`) : undefined;
        const stored = hasStoredSecret && !replacing.has(field.key);

        if (field.kind === "text") {
          return (
            <Field key={field.key} id={fieldId} label={label} hint={hint} error={errors[field.key]}>
              <Input
                id={fieldId}
                value={values[field.key] ?? ""}
                placeholder={field.placeholder}
                disabled={disabled}
                spellCheck={false}
                aria-invalid={errors[field.key] ? true : undefined}
                className={cn(field.mono && "font-mono")}
                onChange={(event) => onValueChange(field.key, event.target.value)}
              />
            </Field>
          );
        }

        if (field.kind === "secret") {
          return (
            <Field
              key={field.key}
              id={fieldId}
              label={label}
              hint={hint}
              error={errors[field.key]}
              action={
                hasStoredSecret ? (
                  <Button
                    type="button"
                    variant="link"
                    size="sm"
                    disabled={disabled}
                    onClick={() => onReplaceToggle(field.key, stored)}
                  >
                    {stored ? t("configurations.dialog.changeSecret") : t("configurations.dialog.cancelChange")}
                  </Button>
                ) : undefined
              }
            >
              {stored ? (
                <Input id={fieldId} value="••••••••••••" disabled readOnly />
              ) : (
                <PasswordInput
                  id={fieldId}
                  value={values[field.key] ?? ""}
                  disabled={disabled}
                  autoComplete="off"
                  aria-invalid={errors[field.key] ? true : undefined}
                  onChange={(event) => onValueChange(field.key, event.target.value)}
                />
              )}
            </Field>
          );
        }

        return (
          <Field
            key={field.key}
            id={fieldId}
            label={label}
            hint={hint}
            error={errors[field.key]}
            action={
              hasStoredSecret ? (
                <Button
                  type="button"
                  variant="link"
                  size="sm"
                  disabled={disabled}
                  onClick={() => onReplaceToggle(field.key, stored)}
                >
                  {stored ? t("configurations.dialog.changeSecret") : t("configurations.dialog.cancelChange")}
                </Button>
              ) : undefined
            }
          >
            {stored ? (
              <Input id={fieldId} value="••••••••••••" disabled readOnly />
            ) : (
              <JsonKeyFileInput
                id={fieldId}
                value={keyFile}
                onChange={onKeyFileChange}
                onError={onKeyFileError}
                storedDerived={storedDerived}
                disabled={disabled}
                invalid={Boolean(errors[field.key])}
                labels={{
                  dropzone: t("configurations.dialog.keyFile.dropzone"),
                  browse: t("configurations.dialog.keyFile.browse"),
                  hint: t("configurations.dialog.keyFile.hint"),
                  replace: t("configurations.dialog.keyFile.replace"),
                  remove: t("configurations.dialog.keyFile.remove"),
                  projectId: t("fields.projectId"),
                  clientEmail: t("fields.clientEmail"),
                  errors: {
                    tooLarge: t("configurations.dialog.keyFile.errors.tooLarge"),
                    wrongType: t("configurations.dialog.keyFile.errors.wrongType"),
                    unreadable: t("configurations.dialog.keyFile.errors.unreadable"),
                    notJson: t("configurations.dialog.keyFile.errors.notJson"),
                    notServiceAccount: t("configurations.dialog.keyFile.errors.notServiceAccount"),
                    missingFields: t("configurations.dialog.keyFile.errors.missingFields"),
                  },
                }}
              />
            )}
          </Field>
        );
      })}
    </>
  );
}

"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { apiErrorMessage, apiErrorStatus } from "@/lib/api-error";
import {
  useCreateRuleMutation,
  useUpdateRuleMutation,
  type Rule,
  type RuleType,
} from "@/store/api/rules-api";

function RuleForm({
  rule,
  onClose,
}: {
  rule: Rule | null;
  onClose: () => void;
}) {
  const t = useTranslations("rules");
  const common = useTranslations("common");

  const TYPE_OPTIONS = [
    { label: t("type.keyword"), value: "KEYWORD" },
    { label: t("type.regex"), value: "REGEX" },
  ];

  const [name, setName] = React.useState(rule?.rule_name ?? "");
  const [type, setType] = React.useState<RuleType>(
    rule?.type ?? "KEYWORD",
  );
  const [value, setValue] = React.useState(rule?.value ?? "");
  const [error, setError] = React.useState("");
  const [valueInvalid, setValueInvalid] = React.useState(false);

  const [createRule, { isLoading: creating }] = useCreateRuleMutation();
  const [updateRule, { isLoading: updating }] = useUpdateRuleMutation();

  const pending = creating || updating;

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const input = {
      rule_name: name.trim(),
      type,
      value: value.trim(),
    };

    setError("");
    setValueInvalid(false);

    if (!input.rule_name) {
      setError(t("dialog.nameRequired"));
      return;
    }

    if (!input.value) {
      setValueInvalid(true);
      setError(
        type === "REGEX"
          ? t("dialog.patternRequired")
          : t("dialog.keywordRequired"),
      );
      return;
    }

    try {
      if (rule) {
        await updateRule({ id: rule.id, ...input }).unwrap();
        toast.success(t("dialog.updated"));
      } else {
        await createRule(input).unwrap();
        toast.success(t("dialog.created"));
      }

      onClose();
    } catch (caught) {
      setValueInvalid(apiErrorStatus(caught) === 400);
      setError(apiErrorMessage(caught, t("dialog.saveFailed")));
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-5">
      <FormError message={error} />

      <Field id="rule-name" label={t("dialog.name")}>
        <Input
          id="rule-name"
          required
          autoFocus
          placeholder={t("dialog.namePlaceholder")}
          value={name}
          disabled={pending}
          onChange={(event) => {
            setName(event.target.value);
            setError("");
          }}
        />
      </Field>

      <Field id="rule-type" label={t("dialog.matchType")}>
        <Select
          id="rule-type"
          value={type}
          options={TYPE_OPTIONS}
          disabled={pending}
          onChange={(event) => {
            setType(event.target.value as RuleType);
            setValueInvalid(false);
            setError("");
          }}
        />
      </Field>

      <Field
        id="rule-value"
        label={
          type === "REGEX"
            ? t("dialog.pattern")
            : t("dialog.keyword")
        }
        hint={
          type === "REGEX"
            ? t("dialog.patternHint")
            : t("dialog.keywordHint")
        }
      >
        <Textarea
          id="rule-value"
          required
          spellCheck={false}
          className={type === "REGEX" ? "font-mono" : undefined}
          placeholder={
            type === "REGEX"
              ? "\\d{16}"
              : "confidential"
          }
          value={value}
          disabled={pending}
          aria-invalid={valueInvalid || undefined}
          onChange={(event) => {
            setValue(event.target.value);
            setValueInvalid(false);
            setError("");
          }}
        />
      </Field>

      <div className="flex justify-end gap-2 border-t pt-4">
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
          {rule
            ? t("dialog.saveChanges")
            : t("dialog.create")}
        </Button>
      </div>
    </form>
  );
}

export function RuleDialog({
  open,
  onOpenChange,
  rule,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule?: Rule | null;
}) {
  const t = useTranslations("rules");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={
        rule
          ? t("dialog.editTitle")
          : t("dialog.addTitle")
      }
      description="A rule matches message content by keyword or by regular expression."
      width="lg"
    >
      <RuleForm
        key={rule?.id ?? "create"}
        rule={rule ?? null}
        onClose={() => onOpenChange(false)}
      />
    </DrawerWrapper>
  );
}
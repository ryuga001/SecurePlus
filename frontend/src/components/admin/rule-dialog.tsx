"use client";

import { Loader2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
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

const TYPE_OPTIONS = [
  { label: "Keyword", value: "KEYWORD" },
  { label: "Regular expression", value: "REGEX" },
];

function RuleForm({ rule, onClose }: { rule: Rule | null; onClose: () => void }) {
  const [name, setName] = React.useState(rule?.rule_name ?? "");
  const [type, setType] = React.useState<RuleType>(rule?.type ?? "KEYWORD");
  const [value, setValue] = React.useState(rule?.value ?? "");
  const [error, setError] = React.useState("");
  const [valueInvalid, setValueInvalid] = React.useState(false);

  const [createRule, { isLoading: creating }] = useCreateRuleMutation();
  const [updateRule, { isLoading: updating }] = useUpdateRuleMutation();

  const pending = creating || updating;

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const input = { rule_name: name.trim(), type, value };

    if (!input.rule_name) {
      setError("Enter a rule name.");
      return;
    }
    if (!input.value.trim()) {
      setValueInvalid(true);
      setError(type === "REGEX" ? "Enter a pattern to match." : "Enter a keyword to match.");
      return;
    }

    setError("");
    setValueInvalid(false);

    try {
      if (rule) {
        await updateRule({ id: rule.id, ...input }).unwrap();
        toast.success("Rule updated");
      } else {
        await createRule(input).unwrap();
        toast.success("Rule created");
      }

      onClose();
    } catch (caught) {
      setValueInvalid(apiErrorStatus(caught) === 400);
      setError(apiErrorMessage(caught, "Could not save this rule"));
    }
  }

  return (
    <>
      <DialogTitle>{rule ? "Edit rule" : "Add rule"}</DialogTitle>
      <DialogDescription>
        A rule matches message content by keyword or by regular expression.
      </DialogDescription>

      <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-4">
        <FormError message={error} />

        <Field id="rule-name" label="Rule name">
          <Input
            id="rule-name"
            required
            autoFocus
            placeholder="Card number"
            value={name}
            disabled={pending}
            onChange={(event) => {
              setName(event.target.value);
              setError("");
            }}
          />
        </Field>

        <Field id="rule-type" label="Match type">
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
          label={type === "REGEX" ? "Pattern" : "Keyword"}
          hint={
            type === "REGEX"
              ? "Go RE2 syntax, for example \\d{16}. Whitespace is significant."
              : "Matched case-insensitively against message content."
          }
        >
          <Textarea
            id="rule-value"
            required
            spellCheck={false}
            className={type === "REGEX" ? "font-mono" : undefined}
            placeholder={type === "REGEX" ? "\\d{16}" : "confidential"}
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
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={pending}>
            {pending ? <Loader2 className="animate-spin" /> : null}
            {rule ? "Save changes" : "Create rule"}
          </Button>
        </div>
      </form>
    </>
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
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100svh-4rem)] max-w-lg overflow-y-auto">
        <RuleForm rule={rule ?? null} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}

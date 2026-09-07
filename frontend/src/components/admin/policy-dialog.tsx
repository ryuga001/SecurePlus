"use client";

import { Loader2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import { CheckboxList } from "@/components/ui/checkbox-list";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { apiErrorMessage, apiErrorCode } from "@/lib/api-error";
import { useListEmailGroupsQuery, type EmailGroup } from "@/store/api/email-groups-api";
import {
  useCreatePolicyMutation,
  useGetPolicyQuery,
  useUpdatePolicyMutation,
  type Policy,
} from "@/store/api/policies-api";
import { useListRulesQuery, type Rule } from "@/store/api/rules-api";

const ALL = { page: 1, pageSize: 100, filters: {} };

const STATUS_OPTIONS = [
  { label: "Enabled", value: "true" },
  { label: "Disabled", value: "false" },
];

function toggle(ids: number[], id: number, checked: boolean) {
  return checked ? [...ids, id] : ids.filter((value) => value !== id);
}

function PolicyForm({
  policy,
  rules,
  groups,
  onClose,
}: {
  policy: Policy | null;
  rules: Rule[];
  groups: EmailGroup[];
  onClose: () => void;
}) {
  const [createPolicy, { isLoading: creating }] = useCreatePolicyMutation();
  const [updatePolicy, { isLoading: updating }] = useUpdatePolicyMutation();

  const policyId = policy?.id ?? null;

  const [name, setName] = React.useState(policy?.policy_name ?? "");
  const [active, setActive] = React.useState(policy?.active ?? true);
  const [ruleIDs, setRuleIDs] = React.useState<number[]>(
    (policy?.rules ?? []).map((rule) => rule.id)
  );
  const [groupIDs, setGroupIDs] = React.useState<number[]>(
    (policy?.groups ?? []).map((group) => group.id)
  );
  const [error, setError] = React.useState("");

  const pending = creating || updating;

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmed = name.trim();

    if (!trimmed) {
      setError("Enter a policy name.");
      return;
    }
    if (ruleIDs.length === 0) {
      setError("Select at least one rule.");
      return;
    }
    if (groupIDs.length === 0) {
      setError("Select at least one group.");
      return;
    }

    setError("");

    const input = {
      policy_name: trimmed,
      active,
      group_ids: groupIDs,
      rule_ids: ruleIDs,
    };

    try {
      if (policyId === null) {
        await createPolicy(input).unwrap();
        toast.success("Policy created");
      } else {
        await updatePolicy({ id: policyId, ...input }).unwrap();
        toast.success("Policy updated");
      }

      onClose();
    } catch (caught) {
      const code = apiErrorCode(caught);
      setError(
        apiErrorMessage(
          caught,
          code === "group_not_found" || code === "rule_not_found"
            ? "One of the selected items is no longer available"
            : "Could not save this policy"
        )
      );
    }
  }

  return (
    <>
      <DialogTitle>{policyId === null ? "Add policy" : "Edit policy"}</DialogTitle>
      <DialogDescription>
        An email policy applies its rules to the users in the groups you choose.
      </DialogDescription>

      <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-5">
          <FormError message={error} />

          <div className="grid gap-4 sm:grid-cols-[1fr_10rem]">
            <Field id="policy-name" label="Policy name">
              <Input
                id="policy-name"
                required
                autoFocus
                placeholder="Outbound PII"
                value={name}
                disabled={pending}
                onChange={(event) => {
                  setName(event.target.value);
                  setError("");
                }}
              />
            </Field>

            <Field id="policy-status" label="Status">
              <Select
                id="policy-status"
                value={String(active)}
                options={STATUS_OPTIONS}
                disabled={pending}
                onChange={(event) => setActive(event.target.value === "true")}
              />
            </Field>
          </div>

          <Field id="policy-rules" label="Rules" hint="What this policy looks for.">
            <CheckboxList
              options={rules.map((rule) => ({
                id: rule.id,
                label: rule.rule_name,
                hint: rule.type === "REGEX" ? "regex" : "keyword",
              }))}
              selected={ruleIDs}
              disabled={pending}
              searchPlaceholder="Search rules"
              emptyMessage="Create a rule first"
              onToggle={(id, checked) => {
                setRuleIDs((current) => toggle(current, id, checked));
                setError("");
              }}
            />
          </Field>

          <Field id="policy-groups" label="Groups" hint="Whose mail this policy applies to.">
            <CheckboxList
              options={groups.map((group) => ({
                id: group.id,
                label: group.name,
                hint: `${group.member_count} members`,
              }))}
              selected={groupIDs}
              disabled={pending}
              searchPlaceholder="Search groups"
              emptyMessage="Create a group first"
              onToggle={(id, checked) => {
                setGroupIDs((current) => toggle(current, id, checked));
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
              {policyId === null ? "Create policy" : "Save changes"}
            </Button>
          </div>
        </form>

    </>
  );
}

function PolicyLoader({ policyId, onClose }: { policyId: number | null; onClose: () => void }) {
  const { data: policy, isLoading: loadingPolicy } = useGetPolicyQuery(policyId as number, {
    skip: policyId === null,
  });
  const { data: rules, isLoading: loadingRules } = useListRulesQuery(ALL);
  const { data: groups, isLoading: loadingGroups } = useListEmailGroupsQuery(ALL);

  if (loadingPolicy || loadingRules || loadingGroups) {
    return (
      <>
        <DialogTitle>{policyId === null ? "Add policy" : "Edit policy"}</DialogTitle>
        <DialogDescription>Loading rules and groups...</DialogDescription>

        <div className="mt-6 flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          One moment
        </div>
      </>
    );
  }

  return (
    <PolicyForm
      policy={policy ?? null}
      rules={rules?.items ?? []}
      groups={groups?.items ?? []}
      onClose={onClose}
    />
  );
}

export function PolicyDialog({
  open,
  onOpenChange,
  policyId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  policyId?: number | null;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100svh-4rem)] max-w-xl overflow-y-auto">
        <PolicyLoader policyId={policyId ?? null} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}

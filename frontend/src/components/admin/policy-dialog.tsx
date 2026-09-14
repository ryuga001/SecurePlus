"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import { CheckboxList } from "@/components/ui/checkbox-list";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { TagInput } from "@/components/ui/tag-input";
import { apiErrorMessage, apiErrorCode } from "@/lib/api-error";
import { useListEmailGroupsQuery, type EmailGroup } from "@/store/api/email-groups-api";
import {
  useCreatePolicyMutation,
  useGetPolicyQuery,
  useListFileTypesQuery,
  useUpdatePolicyMutation,
  type FileType,
  type Policy,
  type PolicyAction,
  type RestrictionMode,
} from "@/store/api/policies-api";
import { useListRulesQuery, type Rule } from "@/store/api/rules-api";

const ALL = { page: 1, pageSize: 100, filters: {} };

function toggle(ids: number[], id: number, checked: boolean) {
  return checked ? [...ids, id] : ids.filter((value) => value !== id);
}

function PolicyForm({
  policy,
  rules,
  groups,
  fileTypes,
  onClose,
}: {
  policy: Policy | null;
  rules: Rule[];
  groups: EmailGroup[];
  fileTypes: FileType[];
  onClose: () => void;
}) {
  const t = useTranslations("policies");
  const status = useTranslations("status");
  const common = useTranslations("common");

  const STATUS_OPTIONS = [
    { label: status("enabled"), value: "true" },
    { label: status("disabled"), value: "false" },
  ];

  const ACTION_OPTIONS = [
    { label: status("audit"), value: "AUDIT" },
    { label: status("block"), value: "BLOCK" },
    { label: status("quarantine"), value: "QUARANTINE" },
    { label: status("redact"), value: "REDACT" },
  ];

  const RESTRICTION_OPTIONS = [
    { label: t("dialog.restrictionNone"), value: "NONE" },
    { label: t("dialog.restrictionBlock"), value: "BLOCK" },
    { label: t("dialog.restrictionAllow"), value: "ALLOW" },
  ];

  const restrictionHint = (mode: RestrictionMode) =>
    mode === "BLOCK"
      ? t("dialog.hintBlock")
      : mode === "ALLOW"
        ? t("dialog.hintAllow")
        : t("dialog.hintNone");

  const [createPolicy, { isLoading: creating }] = useCreatePolicyMutation();
  const [updatePolicy, { isLoading: updating }] = useUpdatePolicyMutation();

  const policyId = policy?.id ?? null;

  const [name, setName] = React.useState(policy?.policy_name ?? "");
  const [action, setAction] = React.useState<PolicyAction>(policy?.action ?? "AUDIT");
  const [active, setActive] = React.useState(policy?.active ?? true);
  const [ruleIDs, setRuleIDs] = React.useState<number[]>(
    (policy?.rules ?? []).map((rule) => rule.id)
  );
  const [groupIDs, setGroupIDs] = React.useState<number[]>(
    (policy?.groups ?? []).map((group) => group.id)
  );
  const [domainMode, setDomainMode] = React.useState<RestrictionMode>(
    policy?.domain_restriction.mode ?? "NONE"
  );
  const [domains, setDomains] = React.useState<string[]>(
    policy?.domain_restriction.values ?? []
  );
  const [attachmentMode, setAttachmentMode] = React.useState<RestrictionMode>(
    policy?.attachment_restriction.mode ?? "NONE"
  );
  const [extensions, setExtensions] = React.useState<string[]>(
    policy?.attachment_restriction.values ?? []
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
    if (domainMode !== "NONE" && domains.length === 0) {
      setError("Add at least one domain, or set the domain restriction to none.");
      return;
    }
    if (attachmentMode !== "NONE" && extensions.length === 0) {
      setError("Select at least one file type, or set the attachment restriction to none.");
      return;
    }

    setError("");

    const input = {
      policy_name: trimmed,
      action,
      active,
      group_ids: groupIDs,
      rule_ids: ruleIDs,
      domain_restriction: {
        mode: domainMode,
        values: domainMode === "NONE" ? [] : domains,
      },
      attachment_restriction: {
        mode: attachmentMode,
        values: attachmentMode === "NONE" ? [] : extensions,
      },
    };

    try {
      if (policyId === null) {
        await createPolicy(input).unwrap();
        toast.success(t("dialog.created"));
      } else {
        await updatePolicy({ id: policyId, ...input }).unwrap();
        toast.success(t("dialog.updated"));
      }

      onClose();
    } catch (caught) {
      const code = apiErrorCode(caught);
      setError(
        apiErrorMessage(
          caught,
          code === "group_not_found" || code === "rule_not_found"
            ? t("dialog.staleSelection")
            : t("dialog.saveFailed")
        )
      );
    }
  }

  return (
    <>
      <DialogTitle>{policyId === null ? t("dialog.addTitle") : t("dialog.editTitle")}</DialogTitle>
      <DialogDescription>{t("dialog.description")}</DialogDescription>

      <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-5">
          <FormError message={error} />

          <div className="grid gap-4 sm:grid-cols-[1fr_10rem]">
            <Field id="policy-name" label={t("dialog.name")}>
              <Input
                id="policy-name"
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

            <Field id="policy-status" label={t("dialog.status")}>
              <Select
                id="policy-status"
                value={String(active)}
                options={STATUS_OPTIONS}
                disabled={pending}
                onChange={(event) => setActive(event.target.value === "true")}
              />
            </Field>
          </div>

          <Field
            id="policy-action"
            label={t("dialog.action")}
            hint={t("dialog.actionHint")}
          >
            <Select
              id="policy-action"
              value={action}
              options={ACTION_OPTIONS}
              disabled={pending}
              onChange={(event) => setAction(event.target.value as PolicyAction)}
            />
          </Field>

          <Field id="policy-rules" label={t("dialog.rules")} hint={t("dialog.rulesHint")}>
            <CheckboxList
              options={rules.map((rule) => ({
                id: rule.id,
                label: rule.rule_name,
                hint: rule.type === "REGEX" ? "regex" : "keyword",
              }))}
              selected={ruleIDs}
              disabled={pending}
              searchPlaceholder={t("dialog.searchRules")}
              emptyMessage={t("dialog.noRules")}
              onToggle={(id, checked) => {
                setRuleIDs((current) => toggle(current, id, checked));
                setError("");
              }}
            />
          </Field>

          <Field id="policy-groups" label={t("dialog.groups")} hint={t("dialog.groupsHint")}>
            <CheckboxList
              options={groups.map((group) => ({
                id: group.id,
                label: group.name,
                hint: t("dialog.memberCount", { count: group.member_count }),
              }))}
              selected={groupIDs}
              disabled={pending}
              searchPlaceholder={t("dialog.searchGroups")}
              emptyMessage={t("dialog.noGroups")}
              onToggle={(id, checked) => {
                setGroupIDs((current) => toggle(current, id, checked));
                setError("");
              }}
            />
          </Field>

          <div className="flex flex-col gap-3 border-t pt-5">
            <Field
              id="policy-domain-mode"
              label={t("dialog.domainRestriction")}
              hint={restrictionHint(domainMode)}
            >
              <Select
                id="policy-domain-mode"
                value={domainMode}
                options={RESTRICTION_OPTIONS}
                disabled={pending}
                onChange={(event) => {
                  setDomainMode(event.target.value as RestrictionMode);
                  setError("");
                }}
              />
            </Field>

            {domainMode === "NONE" ? null : (
              <TagInput
                values={domains}
                onChange={(values) => {
                  setDomains(values);
                  setError("");
                }}
                disabled={pending}
                placeholder={t("dialog.domainPlaceholder")}
                addLabel={t("dialog.addDomain")}
                emptyMessage={t("dialog.noDomains")}
              />
            )}
          </div>

          <div className="flex flex-col gap-3 border-t pt-5">
            <Field
              id="policy-attachment-mode"
              label={t("dialog.attachmentRestriction")}
              hint={restrictionHint(attachmentMode)}
            >
              <Select
                id="policy-attachment-mode"
                value={attachmentMode}
                options={RESTRICTION_OPTIONS}
                disabled={pending}
                onChange={(event) => {
                  setAttachmentMode(event.target.value as RestrictionMode);
                  setError("");
                }}
              />
            </Field>

            {attachmentMode === "NONE" ? null : (
              <CheckboxList
                options={fileTypes.map((fileType) => ({
                  id: fileType.id,
                  label: `.${fileType.extension}`,
                  hint: fileType.label,
                }))}
                selected={fileTypes
                  .filter((fileType) => extensions.includes(fileType.extension))
                  .map((fileType) => fileType.id)}
                disabled={pending}
                searchPlaceholder={t("dialog.searchFileTypes")}
                emptyMessage={t("dialog.noFileTypes")}
                onToggle={(id, checked) => {
                  const match = fileTypes.find((fileType) => fileType.id === id);
                  if (!match) return;

                  setExtensions((current) =>
                    checked
                      ? [...current, match.extension]
                      : current.filter((value) => value !== match.extension)
                  );
                  setError("");
                }}
              />
            )}
          </div>

          <div className="flex justify-end gap-2 border-t pt-4">
            <Button type="button" variant="outline" onClick={onClose}>
              {common("cancel")}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="animate-spin" /> : null}
              {policyId === null ? t("dialog.create") : t("dialog.saveChanges")}
            </Button>
          </div>
        </form>

    </>
  );
}

function PolicyLoader({ policyId, onClose }: { policyId: number | null; onClose: () => void }) {
  const t = useTranslations("policies");

  const { data: policy, isLoading: loadingPolicy } = useGetPolicyQuery(policyId as number, {
    skip: policyId === null,
  });
  const { data: rules, isLoading: loadingRules } = useListRulesQuery(ALL);
  const { data: groups, isLoading: loadingGroups } = useListEmailGroupsQuery(ALL);
  const { data: fileTypes, isLoading: loadingFileTypes } = useListFileTypesQuery();

  if (loadingPolicy || loadingRules || loadingGroups || loadingFileTypes) {
    return (
      <>
        <DialogTitle>{policyId === null ? t("dialog.addTitle") : t("dialog.editTitle")}</DialogTitle>
        <DialogDescription>{t("dialog.loading")}</DialogDescription>

        <div className="mt-6 flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          {t("dialog.oneMoment")}
        </div>
      </>
    );
  }

  return (
    <PolicyForm
      policy={policy ?? null}
      rules={rules?.items ?? []}
      groups={groups?.items ?? []}
      fileTypes={fileTypes?.items ?? []}
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

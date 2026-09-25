"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { PolicyCombobox, type PolicyRef } from "@/components/data-discovery/policy-combobox";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { apiErrorCode, apiErrorMessage } from "@/lib/api-error";
import { SCANNABLE_SOURCE_TYPES } from "@/lib/data-discovery";
import { useCreateDiscoveryScanMutation } from "@/store/api/data-discovery-scans-api";

const POLICY_ERRORS: Record<string, string> = {
  discovery_scan_active: "alreadyActive",
  discovery_policy_inactive: "policyInactive",
  source_not_scannable: "notScannable",
  not_found: "policyMissing",
};

function ScanForm({
  canSave,
  onClose,
  onStarted,
}: {
  canSave: boolean;
  onClose: () => void;
  onStarted: (scanId: number) => void;
}) {
  const t = useTranslations("data-discovery.scans.dialog");
  const shared = useTranslations("data-discovery");
  const common = useTranslations("common");

  const [policy, setPolicy] = React.useState<PolicyRef | null>(null);
  const [policyError, setPolicyError] = React.useState("");
  const [error, setError] = React.useState("");

  const [createScan, { isLoading: pending }] = useCreateDiscoveryScanMutation();

  const sources = SCANNABLE_SOURCE_TYPES.map((source) => shared(`sourceType.${source}`)).join(", ");

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!policy) {
      setPolicyError(t("policyRequired"));
      return;
    }

    setPolicyError("");
    setError("");

    try {
      const scan = await createScan({ policy_id: policy.id }).unwrap();

      toast.success(t("started", { name: policy.name }));
      onStarted(scan.id);
      onClose();
    } catch (caught) {
      const known = POLICY_ERRORS[apiErrorCode(caught) ?? ""];

      if (known) {
        setPolicyError(t(known));
        return;
      }

      setError(apiErrorMessage(caught, t("startFailed")));
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-5">
      <FormError message={error} />

      <Field
        id="scan-policy"
        label={t("policy")}
        hint={t("policyHint", { sources })}
        error={policyError}
      >
        <PolicyCombobox
          id="scan-policy"
          value={policy}
          activeOnly
          sourceTypes={SCANNABLE_SOURCE_TYPES}
          disabled={pending}
          invalid={Boolean(policyError)}
          placeholder={t("searchPolicies")}
          onChange={(next) => {
            setPolicy(next);
            setPolicyError("");
            setError("");
          }}
        />
      </Field>

      {policy ? (
        <div className="grid grid-cols-2 gap-4 border bg-surface-container-low p-4 text-sm">
          <div>
            <p className="text-xs font-medium tracking-wide text-text-tertiary uppercase">
              {t("targets")}
            </p>
            <p className="mt-1 font-medium">{policy.target_count ?? "—"}</p>
          </div>

          <div>
            <p className="text-xs font-medium tracking-wide text-text-tertiary uppercase">
              {t("rules")}
            </p>
            <p className="mt-1 font-medium">{policy.rule_count ?? "—"}</p>
          </div>
        </div>
      ) : null}

      <p className="text-sm text-muted-foreground">{t("runsInBackground")}</p>

      <div className="flex shrink-0 justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
          {common("cancel")}
        </Button>

        <Button type="submit" disabled={pending || !canSave}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          {t("start")}
        </Button>
      </div>
    </form>
  );
}

export function ScanDrawer({
  open,
  onOpenChange,
  canSave,
  onStarted,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  canSave: boolean;
  onStarted: (scanId: number) => void;
}) {
  const t = useTranslations("data-discovery.scans.dialog");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={t("title")}
      description={t("description")}
      width="xl"
    >
      <ScanForm canSave={canSave} onClose={() => onOpenChange(false)} onStarted={onStarted} />
    </DrawerWrapper>
  );
}

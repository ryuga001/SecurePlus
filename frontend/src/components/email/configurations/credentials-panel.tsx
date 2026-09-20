"use client";

import { Check, Copy, KeyRound, Loader2, ShieldCheck } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { copyText } from "@/lib/clipboard";
import { cn } from "@/lib/utils";

function SecretValue({
  label,
  value,
  hint,
  tone,
}: {
  label: string;
  value: string;
  hint?: string;
  tone?: "default" | "warning";
}) {
  const t = useTranslations("configurations");
  const [copied, setCopied] = React.useState(false);

  React.useEffect(() => {
    if (!copied) return;

    const timer = setTimeout(() => setCopied(false), 2000);

    return () => clearTimeout(timer);
  }, [copied]);

  async function onCopy() {
    const done = await copyText(value);

    if (done) {
      setCopied(true);
      toast.success(`${label} copied`);
      return;
    }

    toast.error(t("dialog.copyFailed"));
  }

  return (
    <div
      className={cn(
        "border p-3",
        tone === "warning" && "border-amber-500/40 bg-amber-50/60 dark:bg-amber-950/20",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          {label}
        </span>

        <Button type="button" variant="ghost" size="xs" onClick={onCopy}>
          {copied ? <Check /> : <Copy />}
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>

      <p className="mt-2 max-h-32 overflow-y-auto font-mono text-xs break-all">{value}</p>

      {hint ? <p className="mt-2 text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

export function CredentialsPanel({
  dkimPublicKey,
  recordName,
  accessToken,
  expiresAt,
  onGenerateDkim,
  onGenerateAccessToken,
  dkimPending,
  tokenPending,
}: {
  dkimPublicKey: string;
  recordName: string;
  accessToken: string;
  expiresAt: string;
  onGenerateDkim: () => void;
  onGenerateAccessToken: () => void;
  dkimPending: boolean;
  tokenPending: boolean;
}) {
  const t = useTranslations("configurations");

  return (
    <div className="flex flex-col gap-4 border-t pt-5">
      <div>
        <h3 className="text-sm font-medium">{t("dialog.credentialsTitle")}</h3>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("dialog.credentialsHint")}
        </p>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="outline"
          disabled={dkimPending}
          onClick={onGenerateDkim}
        >
          {dkimPending ? <Loader2 className="animate-spin" /> : <KeyRound />}
          {t("dialog.generateDkim")}
        </Button>

        <Button
          type="button"
          variant="outline"
          disabled={tokenPending}
          onClick={onGenerateAccessToken}
        >
          {tokenPending ? <Loader2 className="animate-spin" /> : <ShieldCheck />}
          {t("dialog.generateToken")}
        </Button>
      </div>

      {dkimPublicKey ? (
        <SecretValue
          label={t("dialog.dkimPublicKey")}
          value={dkimPublicKey}
          hint={recordName ? t("dialog.dkimRecordHint", { domain: recordName }) : t("dialog.dkimRecordHintGeneric")}
        />
      ) : null}

      {accessToken ? (
        <SecretValue
          label={t("dialog.accessToken")}
          value={accessToken}
          tone="warning"
          hint={
            expiresAt
              ? t("dialog.tokenExpiryHint", { date: new Date(expiresAt).toLocaleString() })
              : t("dialog.tokenOnceHint")
          }
        />
      ) : null}
    </div>
  );
}
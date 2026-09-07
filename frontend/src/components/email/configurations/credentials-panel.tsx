"use client";

import { Check, Copy, KeyRound, Loader2, ShieldCheck } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { apiErrorMessage } from "@/lib/api-error";
import { copyText } from "@/lib/clipboard";
import { cn } from "@/lib/utils";
import {
  useGenerateAccessTokenMutation,
  useGenerateDkimKeyMutation,
} from "@/store/api/email-configurations-api";

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

    toast.error("Could not copy — select the value and copy manually");
  }

  return (
    <div
      className={cn(
        "border p-3",
        tone === "warning" && "border-amber-500/40 bg-amber-50/60 dark:bg-amber-950/20"
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
  configurationId,
  initialDkimPublicKey,
  onSecretRevealed,
}: {
  configurationId: number | null;
  initialDkimPublicKey?: string | null;
  onSecretRevealed?: () => void;
}) {
  const [dkimPublicKey, setDkimPublicKey] = React.useState(initialDkimPublicKey ?? "");
  const [dkimPrivateKey, setDkimPrivateKey] = React.useState("");
  const [recordName, setRecordName] = React.useState("");
  const [accessToken, setAccessToken] = React.useState("");
  const [expiresAt, setExpiresAt] = React.useState("");

  const [generateDkimKey, { isLoading: dkimPending }] = useGenerateDkimKeyMutation();
  const [generateAccessToken, { isLoading: tokenPending }] = useGenerateAccessTokenMutation();

  const locked = configurationId === null;

  async function onGenerateDkim() {
    if (configurationId === null) return;

    try {
      const result = await generateDkimKey(configurationId).unwrap();
      setDkimPublicKey(result.dkim_public_key);
      setDkimPrivateKey(result.dkim_private_key);
      setRecordName(result.record_name);
      onSecretRevealed?.();
      toast.success("DKIM key pair generated");
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not generate a DKIM key"));
    }
  }

  async function onGenerateToken() {
    if (configurationId === null) return;

    try {
      const result = await generateAccessToken(configurationId).unwrap();
      setAccessToken(result.access_token);
      setExpiresAt(result.expires_at);
      onSecretRevealed?.();
      toast.success("Access token generated");
    } catch (error) {
      toast.error(apiErrorMessage(error, "Could not generate an access token"));
    }
  }

  return (
    <div className="flex flex-col gap-4 border-t pt-5">
      <div>
        <h3 className="text-sm font-medium">Credentials</h3>
        {locked ? (
          <p className="mt-1 text-xs text-muted-foreground">
            Save the configuration to enable key generation.
          </p>
        ) : null}
      </div>

      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="outline"
          disabled={locked || dkimPending}
          onClick={onGenerateDkim}
        >
          {dkimPending ? <Loader2 className="animate-spin" /> : <KeyRound />}
          Generate DKIM
        </Button>

        <Button
          type="button"
          variant="outline"
          disabled={locked || tokenPending}
          onClick={onGenerateToken}
        >
          {tokenPending ? <Loader2 className="animate-spin" /> : <ShieldCheck />}
          Generate Access Token
        </Button>
      </div>

      {dkimPublicKey ? (
        <SecretValue
          label="DKIM public key"
          value={dkimPublicKey}
          hint={
            recordName
              ? `Publish this as a TXT record on ${recordName}.`
              : "Publish this as a TXT record on your DNS."
          }
        />
      ) : null}

      {dkimPrivateKey ? (
        <SecretValue
          label="DKIM private key"
          value={dkimPrivateKey}
          tone="warning"
          hint="Stored with the configuration and shown once. It signs your outbound mail — never publish it."
        />
      ) : null}

      {accessToken ? (
        <SecretValue
          label="Access token"
          value={accessToken}
          tone="warning"
          hint={
            expiresAt
              ? `Shown once only. Expires ${new Date(expiresAt).toLocaleString()}.`
              : "Shown once only. Copy it now."
          }
        />
      ) : null}
    </div>
  );
}

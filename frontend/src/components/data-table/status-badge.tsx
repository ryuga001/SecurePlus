"use client";

import { useTranslations } from "next-intl";

import { Badge } from "@/components/ui/badge";

const toneVariant: Record<string, "success" | "warning" | "info" | "destructive" | "neutral"> = {
  active: "success",
  enabled: "success",
  success: "success",
  pending: "warning",
  processing: "warning",
  quarantine: "warning",
  content: "warning",
  warning: "warning",
  disabled: "neutral",
  inactive: "neutral",
  pass: "success",
  invoked: "success",
  invited: "info",
  info: "info",
  audit: "info",
  redact: "info",
  flagged: "destructive",
  restriction: "destructive",
  blocked: "destructive",
  failed: "destructive",
  error: "destructive",
  running: "info",
  partial: "warning",
  completed: "success",
  succeeded: "success",
};

export function StatusBadge({ status }: { status: string }) {
  const t = useTranslations("status");
  const key = status?.toLowerCase?.() ?? "";
  const variant = toneVariant[key] ?? "neutral";

  return (
    <Badge variant={variant} className="capitalize">
      {t.has(key) ? t(key) : status}
    </Badge>
  );
}
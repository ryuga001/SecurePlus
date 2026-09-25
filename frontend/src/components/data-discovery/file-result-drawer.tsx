"use client";

import { useTranslations } from "next-intl";

import {
  useScanErrorLabel,
  useZonedDate,
} from "@/components/data-discovery/scan-details-drawer";
import { StatusBadge } from "@/components/data-table/status-badge";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Badge } from "@/components/ui/badge";
import { formatBytes, formatDuration } from "@/lib/data-discovery";
import type { DiscoveryFileResult } from "@/store/api/data-discovery-scans-api";

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <p className="text-xs font-medium tracking-wide text-text-tertiary uppercase">{label}</p>
      <div className="mt-1 truncate text-sm">{value === "" || value === undefined ? "—" : value}</div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="border-t pt-4">
      <h3 className="mb-3 text-sm font-semibold">{title}</h3>
      {children}
    </section>
  );
}

function Details({ file, target }: { file: DiscoveryFileResult; target?: string }) {
  const t = useTranslations("data-discovery.scans.files.details");
  const errorLabel = useScanErrorLabel();
  const formatDate = useZonedDate();

  return (
    <div className="mt-6 flex flex-col gap-6">
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <Field label={t("status")} value={<StatusBadge status={file.status} />} />
        <Field label={t("findings")} value={file.findings_total.toLocaleString()} />
        <Field label={t("error")} value={errorLabel(file.error_code)} />
        <Field label={t("target")} value={target ? <span className="font-mono text-xs">{target}</span> : ""} />
        <Field label={t("type")} value={file.extension.toUpperCase()} />
        <Field label={t("mimeType")} value={file.mime_type} />
        <Field label={t("size")} value={formatBytes(file.size_bytes)} />
        <Field label={t("read")} value={formatBytes(file.bytes_processed)} />
        <Field label={t("took")} value={formatDuration(file.duration_ms)} />
        <Field label={t("modified")} value={formatDate(file.modified_at)} />
        <Field label={t("processed")} value={formatDate(file.processed_at)} />
      </div>

      <div className="min-w-0">
        <p className="text-xs font-medium tracking-wide text-text-tertiary uppercase">{t("location")}</p>
        <p className="mt-1 font-mono text-xs break-all">{file.file_key}</p>
      </div>

      <Section title={t("rules", { count: file.findings.length })}>
        {file.findings.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {file.status === "FAILED" ? t("notEvaluated") : t("noFindings")}
          </p>
        ) : (
          <div className="overflow-x-auto border">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="border-b bg-surface-container-low text-left">
                  {[t("rule"), t("ruleType"), t("matches"), t("offsets")].map((label) => (
                    <th
                      key={label}
                      className="px-3 py-2 text-xs font-semibold whitespace-nowrap text-text-tertiary uppercase"
                    >
                      {label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {file.findings.map((finding) => (
                  <tr key={finding.rule_id} className="border-b last:border-b-0">
                    <td className="px-3 py-2 font-medium">{finding.rule_name}</td>
                    <td className="px-3 py-2">
                      <Badge variant="secondary" size="sm">
                        {finding.rule_type}
                      </Badge>
                    </td>
                    <td className="px-3 py-2 text-right">{finding.count.toLocaleString()}</td>
                    <td className="px-3 py-2 font-mono text-xs text-muted-foreground">
                      {finding.offsets.length > 0
                        ? finding.offsets.map((offset) => offset.toLocaleString()).join(", ")
                        : "—"}
                      {finding.count > finding.offsets.length ? " …" : ""}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <p className="mt-3 text-xs text-muted-foreground">{t("privacy")}</p>
      </Section>
    </div>
  );
}

export function FileResultDrawer({
  file,
  target,
  onOpenChange,
}: {
  file: DiscoveryFileResult | null;
  target?: string;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("data-discovery.scans.files.details");

  return (
    <DrawerWrapper
      open={file !== null}
      title={file ? file.file_name : t("title")}
      description={t("description")}
      onClose={() => onOpenChange(false)}
      width="2xl"
    >
      {file ? <Details file={file} target={target} /> : null}
    </DrawerWrapper>
  );
}

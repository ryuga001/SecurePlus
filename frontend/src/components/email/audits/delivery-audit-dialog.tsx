"use client";

import { CircleAlert, Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";

import { StatusBadge } from "@/components/data-table/status-badge";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { apiErrorMessage } from "@/lib/api-error";
import { useGetDeliveryAuditQuery, type DeliveryAudit } from "@/store/api/delivery-audits-api";

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{label}</p>
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

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(2)} MB`;
}

function duration(startedAt: string, finishedAt: string) {
  const ms = new Date(finishedAt).getTime() - new Date(startedAt).getTime();
  if (!Number.isFinite(ms) || ms < 0) return "—";
  return ms < 1000 ? `${ms} ms` : `${(ms / 1000).toFixed(2)} s`;
}

function Details({ audit }: { audit: DeliveryAudit }) {
  const t = useTranslations("audits.delivery.dialog");

  return (
    <div className="mt-5 flex flex-col gap-5">
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <Field label={t("status")} value={<StatusBadge status={audit.status} />} />
        <Field label={t("received")} value={new Date(audit.created_at).toLocaleString()} />
        <Field label={t("lastUpdate")} value={new Date(audit.updated_at).toLocaleString()} />
        <Field label={t("from")} value={audit.from} />
        <Field label={t("senderDomain")} value={audit.sender_domain} />
        <Field label={t("size")} value={formatBytes(audit.size)} />
        <Field
          label={t("correlationId")}
          value={<span className="font-mono text-xs">{audit.correlation_id}</span>}
        />
        <Field
          label={t("messageId")}
          value={<span className="font-mono text-xs">{audit.message_id}</span>}
        />
        <Field
          label={t("dkim")}
          value={
            audit.dkim.signed
              ? t("signedAs", {
                  identity: `${audit.dkim.selector}._domainkey.${audit.dkim.domain}`,
                })
              : t("notSigned")
          }
        />
      </div>

      {audit.failure ? (
        <div className="border border-destructive/30 bg-destructive/8 p-4">
          <p className="text-sm font-semibold text-destructive">
            {t("failureTitle", { type: audit.failure.type })}
            {audit.failure.smtp_code > 0 ? ` · SMTP ${audit.failure.smtp_code}` : ""}
          </p>
          <p className="mt-1 text-sm break-words text-destructive/90">
            {audit.failure.reason || t("noReason")}
          </p>
        </div>
      ) : null}

      <Section title={t("recipients", { count: audit.recipients.length })}>
        <div className="overflow-x-auto border">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b bg-muted/60 text-left">
                <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                  {t("address")}
                </th>
                <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                  {t("status")}
                </th>
                <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                  {t("code")}
                </th>
                <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                  {t("detail")}
                </th>
              </tr>
            </thead>
            <tbody>
              {audit.recipients.map((recipient) => (
                <tr key={recipient.email} className="border-b last:border-b-0">
                  <td className="px-3 py-2">{recipient.email}</td>
                  <td className="px-3 py-2">
                    <StatusBadge status={recipient.status} />
                  </td>
                  <td className="px-3 py-2 font-mono text-xs">{recipient.smtp_code || "—"}</td>
                  <td className="max-w-xs px-3 py-2 text-muted-foreground">
                    {recipient.error || "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Section>

      <Section title={t("attempts", { count: audit.attempts.length })}>
        {audit.attempts.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("noAttempts")}
          </p>
        ) : (
          <div className="overflow-x-auto border">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="border-b bg-muted/60 text-left">
                  <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                    #
                  </th>
                  <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                    {t("mxHost")}
                  </th>
                  <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                    {t("tls")}
                  </th>
                  <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                    {t("code")}
                  </th>
                  <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                    {t("took")}
                  </th>
                  <th className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase">
                    {t("detail")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {audit.attempts.map((attempt) => (
                  <tr key={attempt.number} className="border-b last:border-b-0">
                    <td className="px-3 py-2">{attempt.number}</td>
                    <td className="px-3 py-2">{attempt.mx_host || "—"}</td>
                    <td className="px-3 py-2">{attempt.tls || t("none")}</td>
                    <td className="px-3 py-2 font-mono text-xs">{attempt.smtp_code || "—"}</td>
                    <td className="px-3 py-2 text-muted-foreground">
                      {duration(attempt.started_at, attempt.finished_at)}
                    </td>
                    <td className="max-w-xs px-3 py-2 text-muted-foreground">
                      {attempt.error || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Section>
    </div>
  );
}

export function DeliveryAuditDrawer({
  correlationId,
  onOpenChange,
}: {
  correlationId: string | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("audits.delivery.dialog");

  const { data, isLoading, isError, error } = useGetDeliveryAuditQuery(
    correlationId as string,
    {
      skip: !correlationId,
    },
  );

  return (
    <DrawerWrapper
      open={correlationId !== null}
      title={t("title")}
      description={t("description")}
      onClose={() => onOpenChange(false)}
    >
      {isLoading ? (
        <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
          <Loader2 className="size-6 animate-spin text-primary" />

          <p className="text-sm">{t("loading")}</p>
        </div>
      ) : null}

      {!isLoading && isError ? (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <CircleAlert className="size-6 text-destructive" />

          <p className="max-w-md text-sm text-muted-foreground">
            {apiErrorMessage(error, t("loadFailed"))}
          </p>
        </div>
      ) : null}

      {!isLoading && !isError && data ? (
        <Details audit={data} />
      ) : null}
    </DrawerWrapper>
  );
}
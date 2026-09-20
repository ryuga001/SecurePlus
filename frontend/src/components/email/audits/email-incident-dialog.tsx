"use client";

import { useTranslations } from "next-intl";
import { CircleAlert, Loader2 } from "lucide-react";

import { StatusBadge } from "@/components/data-table/status-badge";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { apiErrorMessage } from "@/lib/api-error";
import { useGetEmailIncidentQuery, type EmailIncident } from "@/store/api/email-incidents-api";

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{label}</p>
      <div className="mt-1 truncate text-sm">
        {value === "" || value === undefined ? "—" : value}
      </div>
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

function Th({ children }: { children: React.ReactNode }) {
  return (
    <th className="px-3 py-2 text-left text-xs font-semibold text-muted-foreground uppercase">
      {children}
    </th>
  );
}

function Details({ incident }: { incident: EmailIncident }) {
  const t = useTranslations("audits.incidents.dialog");
  const restriction = incident.trigger === "RESTRICTION";

  return (
    <div className="mt-5 flex flex-col gap-5">
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <Field label={t("decision")} value={<StatusBadge status={incident.decision} />} />
        <Field label={t("trigger")} value={<StatusBadge status={incident.trigger} />} />
        <Field label={t("action")} value={<StatusBadge status={incident.effective_action} />} />
        <Field label={t("actionStatus")} value={<StatusBadge status={incident.action_status} />} />
        <Field label={t("raised")} value={new Date(incident.created_at).toLocaleString()} />
        <Field label={t("policiesEvaluated")} value={incident.evaluated_policy_count} />
        <Field label={t("from")} value={incident.from} />
        <Field label={t("senderDomain")} value={incident.sender_domain} />
        <Field
          label={t("recipients")}
          value={incident.recipients.map((recipient) => recipient.email).join(", ")}
        />
        <Field
          label={t("correlationId")}
          value={<span className="font-mono text-xs">{incident.correlation_id}</span>}
        />
        <Field
          label={t("messageId")}
          value={<span className="font-mono text-xs">{incident.message_id}</span>}
        />
        <Field
          label={t("triggeredPolicies")}
          value={incident.triggered_policy_ids.join(", ")}
        />
      </div>

      {incident.action_status === "FAILED" ? (
        <div className="border border-destructive/30 bg-destructive/8 p-4">
          <p className="text-sm font-semibold text-destructive">
            {t("executorFailed", { action: incident.action_invoked })}
          </p>
          <p className="mt-1 text-sm break-words text-destructive/90">
            {incident.action_error || t("noReason")}
          </p>
        </div>
      ) : null}

      {incident.withheld_recipients.length > 0 ? (
        <Section title={t("withheldTitle", { count: incident.withheld_recipients.length })}>
          <div className="overflow-x-auto border">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="border-b bg-muted/60">
                  <Th>{t("address")}</Th>
                  <Th>{t("domain")}</Th>
                  <Th>{t("policy")}</Th>
                </tr>
              </thead>
              <tbody>
                {incident.withheld_recipients.map((recipient) => (
                  <tr key={recipient.email} className="border-b last:border-b-0">
                    <td className="px-3 py-2">{recipient.email}</td>
                    <td className="px-3 py-2">{recipient.domain}</td>
                    <td className="px-3 py-2">{recipient.policy_name}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="mt-2 text-xs text-muted-foreground">
            These addresses were removed from the envelope. The message was still delivered to
            every other recipient.
          </p>
        </Section>
      ) : null}

      {restriction ? (
        <Section title={t("violationsTitle", { count: incident.restriction_violations.length })}>
          {incident.restriction_violations.length === 0 ? (
            <p className="text-sm text-muted-foreground">No violations recorded.</p>
          ) : (
            <div className="overflow-x-auto border">
              <table className="w-full border-collapse text-sm">
                <thead>
                  <tr className="border-b bg-muted/60">
                    <Th>{t("kind")}</Th>
                    <Th>{t("mode")}</Th>
                    <Th>{t("value")}</Th>
                    <Th>{t("file")}</Th>
                    <Th>{t("policy")}</Th>
                  </tr>
                </thead>
                <tbody>
                  {incident.restriction_violations.map((violation, index) => (
                    <tr key={`${violation.kind}-${violation.value}-${index}`} className="border-b last:border-b-0">
                      <td className="px-3 py-2">{violation.kind}</td>
                      <td className="px-3 py-2">{violation.mode}</td>
                      <td className="px-3 py-2 font-mono text-xs">{violation.value || "—"}</td>
                      <td className="px-3 py-2">{violation.filename || "—"}</td>
                      <td className="px-3 py-2">{violation.policy_name}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Section>
      ) : (
        <Section title={t("matchesTitle", { count: incident.matches.length })}>
          {incident.matches.length === 0 ? (
            <p className="text-sm text-muted-foreground">No rules matched.</p>
          ) : (
            <div className="overflow-x-auto border">
              <table className="w-full border-collapse text-sm">
                <thead>
                  <tr className="border-b bg-muted/60">
                    <Th>{t("rule")}</Th>
                    <Th>{t("type")}</Th>
                    <Th>{t("configured")}</Th>
                    <Th>{t("hits")}</Th>
                    <Th>{t("where")}</Th>
                    <Th>{t("policy")}</Th>
                  </tr>
                </thead>
                <tbody>
                  {incident.matches.map((match) => (
                    <tr key={match.rule_id} className="border-b last:border-b-0">
                      <td className="px-3 py-2">{match.rule_name}</td>
                      <td className="px-3 py-2">{match.rule_type}</td>
                      <td className="px-3 py-2 font-mono text-xs">{match.configured_value}</td>
                      <td className="px-3 py-2">{match.occurrences}</td>
                      <td className="px-3 py-2 text-muted-foreground">
                        {match.locations.join(", ") || "—"}
                      </td>
                      <td className="px-3 py-2">{match.policy_name}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <p className="mt-2 text-xs text-muted-foreground">
            Only the configured rule and how many times it matched are stored. The matched text
            itself is never kept.
          </p>
        </Section>
      )}
    </div>
  );
}

export function EmailIncidentDrawer({
  correlationId,
  onOpenChange,
}: {
  correlationId: string | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("audits.incidents.dialog");

  const { data, isLoading, isError, error } = useGetEmailIncidentQuery(
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
        <Details incident={data} />
      ) : null}
    </DrawerWrapper>
  );
}
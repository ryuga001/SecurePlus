"use client";

import { useTranslations } from "next-intl";
import * as React from "react";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import { StatusBadge } from "@/components/data-table/status-badge";
import type { ColumnConfig, FilterConfig } from "@/components/data-table/types";
import { EmailIncidentDrawer } from "@/components/email/audits/email-incident-dialog";
import {
  useListEmailIncidentsQuery,
  type EmailIncidentListItem,
} from "@/store/api/email-incidents-api";

export default function EmailIncidentsPage() {
  const t = useTranslations("audits");
  const i = useTranslations("audits.incidents");
  const status = useTranslations("status");
  const common = useTranslations("common");

  const [selected, setSelected] = React.useState<string | null>(null);

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: i("searchPlaceholder"),
      width: "w-80",
    },
    {
      key: "trigger",
      label: i("triggerFilter"),
      type: "select",
      options: [
        { label: status("restriction"), value: "RESTRICTION" },
        { label: status("content"), value: "CONTENT" },
      ],
    },
    {
      key: "action",
      label: i("actionFilter"),
      type: "select",
      options: [
        { label: status("block"), value: "BLOCK" },
        { label: status("quarantine"), value: "QUARANTINE" },
        { label: status("redact"), value: "REDACT" },
        { label: status("audit"), value: "AUDIT" },
      ],
    },
    { key: "created", label: i("raisedFilter"), type: "dateRange" },
  ];

  const columns: ColumnConfig<EmailIncidentListItem>[] = [
    {
      key: "created_at",
      label: i("columns.raised"),
      type: "datetime",
      sortable: true,
      width: "12rem",
    },
    { key: "from", label: i("columns.from"), sortable: true },
    {
      key: "recipients",
      label: i("columns.to"),
      render: (_value, row) => (
        <span className="block max-w-xs truncate">
          {row.recipients[0] ?? "—"}
          {row.recipient_count > 1 ? (
            <span className="text-muted-foreground"> +{row.recipient_count - 1}</span>
          ) : null}
        </span>
      ),
    },
    {
      key: "trigger",
      label: i("columns.trigger"),
      sortable: true,
      render: (_value, row) => <StatusBadge status={row.trigger} />,
    },
    {
      key: "effective_action",
      label: i("columns.action"),
      sortable: true,
      render: (_value, row) => (
        <div className="flex items-center gap-2">
          <StatusBadge status={row.effective_action} />
          {row.action_status === "FAILED" ? (
            <span className="text-xs text-destructive">{i("executorFailed")}</span>
          ) : null}
        </div>
      ),
    }
  ];

  return (
    <Consolepage
      heading={t("heading")}
      subheading={i("subheading")}
      data={
        <>
          <DataTable
            query={useListEmailIncidentsQuery}
            columns={columns}
            filters={filters}
            getRowId={(row) => row.correlation_id}
            onRowClick={(row) => setSelected(row.correlation_id)}
            defaultSort={{ key: "created_at", direction: "desc" }}
            emptyMessage={i("empty")}
          />

          <EmailIncidentDrawer
            correlationId={selected}
            onOpenChange={(open) => {
              if (!open) setSelected(null);
            }}
          />
        </>
      }
    />
  );
}

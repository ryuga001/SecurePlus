"use client";

import * as React from "react";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import { StatusBadge } from "@/components/data-table/status-badge";
import type { ColumnConfig, FilterConfig } from "@/components/data-table/types";
import { EmailIncidentDialog } from "@/components/email/audits/email-incident-dialog";
import {
  useListEmailIncidentsQuery,
  type EmailIncidentListItem,
} from "@/store/api/email-incidents-api";

const filters: FilterConfig[] = [
  {
    key: "search",
    label: "Search",
    type: "text",
    placeholder: "Sender, recipient, rule or policy",
    width: "w-80",
  },
  {
    key: "trigger",
    label: "Trigger",
    type: "select",
    options: [
      { label: "Restriction", value: "RESTRICTION" },
      { label: "Content rule", value: "CONTENT" },
    ],
  },
  {
    key: "action",
    label: "Action",
    type: "select",
    options: [
      { label: "Block", value: "BLOCK" },
      { label: "Quarantine", value: "QUARANTINE" },
      { label: "Redact", value: "REDACT" },
      { label: "Audit", value: "AUDIT" },
    ],
  },
  { key: "created", label: "Raised", type: "dateRange" },
];

const columns: ColumnConfig<EmailIncidentListItem>[] = [
  { key: "created_at", label: "Raised", type: "datetime", sortable: true, width: "12rem" },
  { key: "from", label: "From", sortable: true },
  {
    key: "recipients",
    label: "To",
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
    label: "Trigger",
    sortable: true,
    render: (_value, row) => <StatusBadge status={row.trigger} />,
  },
  {
    key: "effective_action",
    label: "Action",
    sortable: true,
    render: (_value, row) => (
      <div className="flex items-center gap-2">
        <StatusBadge status={row.effective_action} />
        {row.action_status === "FAILED" ? (
          <span className="text-xs text-destructive">executor failed</span>
        ) : null}
      </div>
    ),
  },
  {
    key: "evidence",
    label: "Evidence",
    align: "right",
    width: "10rem",
    accessor: (row) => row.match_count + row.violation_count,
    render: (_value, row) => (
      <span className="text-muted-foreground">
        {row.trigger === "RESTRICTION"
          ? `${row.violation_count} violation${row.violation_count === 1 ? "" : "s"}`
          : `${row.match_count} rule${row.match_count === 1 ? "" : "s"}`}
      </span>
    ),
  },
  {
    key: "withheld_count",
    label: "Withheld",
    type: "number",
    align: "right",
    width: "7rem",
  },
];

export default function EmailIncidentsPage() {
  const [selected, setSelected] = React.useState<string | null>(null);

  return (
    <Consolepage
      heading="Email audits"
      subheading="Policy violations raised against outbound mail."
      data={
        <>
          <DataTable
            query={useListEmailIncidentsQuery}
            columns={columns}
            filters={filters}
            getRowId={(row) => row.correlation_id}
            onRowClick={(row) => setSelected(row.correlation_id)}
            defaultSort={{ key: "created_at", direction: "desc" }}
            emptyMessage="No incidents raised yet"
          />

          <EmailIncidentDialog
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

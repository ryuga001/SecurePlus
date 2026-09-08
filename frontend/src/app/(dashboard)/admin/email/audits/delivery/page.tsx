"use client";

import * as React from "react";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { DataTable } from "@/components/data-table/data-table";
import { StatusBadge } from "@/components/data-table/status-badge";
import type { ColumnConfig, FilterConfig } from "@/components/data-table/types";
import { DeliveryAuditDialog } from "@/components/email/audits/delivery-audit-dialog";
import {
  useListDeliveryAuditsQuery,
  type DeliveryAuditListItem,
} from "@/store/api/delivery-audits-api";

const filters: FilterConfig[] = [
  {
    key: "search",
    label: "Search",
    type: "text",
    placeholder: "Sender, recipient or message id",
    width: "w-80",
  },
  {
    key: "status",
    label: "Status",
    type: "select",
    options: [
      { label: "Processing", value: "PROCESSING" },
      { label: "Delivered", value: "SUCCESS" },
      { label: "Failed", value: "FAILED" },
    ],
  },
  {
    key: "failure",
    label: "Failure",
    type: "select",
    options: [
      { label: "Processing", value: "PROCESSING" },
      { label: "DKIM", value: "DKIM" },
      { label: "Relay", value: "RELAY" },
      { label: "Rule", value: "RULE" },
      { label: "Unknown", value: "UNKNOWN" },
    ],
  },
  { key: "created", label: "Received", type: "dateRange" },
];

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(2)} MB`;
}

const columns: ColumnConfig<DeliveryAuditListItem>[] = [
  { key: "created_at", label: "Received", type: "datetime", sortable: true, width: "12rem" },
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
    key: "status",
    label: "Status",
    sortable: true,
    render: (_value, row) => (
      <div className="flex items-center gap-2">
        <StatusBadge status={row.status} />
        {row.failure_type ? (
          <span className="text-xs text-muted-foreground">{row.failure_type}</span>
        ) : null}
      </div>
    ),
  },
  { key: "attempt_count", label: "Attempts", type: "number", align: "right", width: "7rem" },
  {
    key: "size",
    label: "Size",
    align: "right",
    width: "7rem",
    sortable: true,
    render: (value) => formatBytes(Number(value)),
  },
];

export default function DeliveryAuditPage() {
  const [selected, setSelected] = React.useState<string | null>(null);

  return (
    <Consolepage
      heading="Email audits"
      subheading="Every message this workspace accepted, and what happened to it."
      data={
        <>
          <DataTable
            query={useListDeliveryAuditsQuery}
            columns={columns}
            filters={filters}
            getRowId={(row) => row.correlation_id}
            onRowClick={(row) => setSelected(row.correlation_id)}
            defaultSort={{ key: "created_at", direction: "desc" }}
            emptyMessage="No messages have been relayed yet"
          />

          <DeliveryAuditDialog
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

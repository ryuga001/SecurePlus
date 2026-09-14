"use client";

import { useTranslations } from "next-intl";
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

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(2)} MB`;
}

export default function DeliveryAuditPage() {
  const t = useTranslations("audits");
  const d = useTranslations("audits.delivery");
  const status = useTranslations("status");
  const common = useTranslations("common");

  const [selected, setSelected] = React.useState<string | null>(null);

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: d("searchPlaceholder"),
      width: "w-80",
    },
    {
      key: "status",
      label: d("statusFilter"),
      type: "select",
      options: [
        { label: status("processing"), value: "PROCESSING" },
        { label: status("success"), value: "SUCCESS" },
        { label: status("failed"), value: "FAILED" },
      ],
    },
    {
      key: "failure",
      label: d("failureFilter"),
      type: "select",
      options: [
        { label: d("failure.processing"), value: "PROCESSING" },
        { label: d("failure.dkim"), value: "DKIM" },
        { label: d("failure.relay"), value: "RELAY" },
        { label: d("failure.rule"), value: "RULE" },
        { label: d("failure.unknown"), value: "UNKNOWN" },
      ],
    },
    { key: "created", label: d("receivedFilter"), type: "dateRange" },
  ];

  const columns: ColumnConfig<DeliveryAuditListItem>[] = [
    {
      key: "created_at",
      label: d("columns.received"),
      type: "datetime",
      sortable: true,
      width: "12rem",
    },
    { key: "from", label: d("columns.from"), sortable: true },
    {
      key: "recipients",
      label: d("columns.to"),
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
      label: d("columns.status"),
      sortable: true,
      render: (_value, row) => (
        <div className="flex items-center gap-2">
          <StatusBadge status={row.status} />
          {row.failure_type ? (
            <span className="text-xs text-muted-foreground">
              {d.has(`failure.${row.failure_type.toLowerCase()}`)
                ? d(`failure.${row.failure_type.toLowerCase()}`)
                : row.failure_type}
            </span>
          ) : null}
        </div>
      ),
    },
    {
      key: "attempt_count",
      label: d("columns.attempts"),
      type: "number",
      align: "right",
      width: "7rem",
    },
    {
      key: "size",
      label: d("columns.size"),
      align: "right",
      width: "7rem",
      sortable: true,
      render: (value) => formatBytes(Number(value)),
    },
  ];

  return (
    <Consolepage
      heading={t("heading")}
      subheading={d("subheading")}
      data={
        <>
          <DataTable
            query={useListDeliveryAuditsQuery}
            columns={columns}
            filters={filters}
            getRowId={(row) => row.correlation_id}
            onRowClick={(row) => setSelected(row.correlation_id)}
            defaultSort={{ key: "created_at", direction: "desc" }}
            emptyMessage={d("empty")}
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

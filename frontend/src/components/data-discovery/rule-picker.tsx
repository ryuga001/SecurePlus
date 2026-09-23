"use client";

import { Minus, Plus, X } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";

import { DataTable } from "@/components/data-table/data-table";
import type {
  ColumnConfig,
  FilterConfig,
  RowAction,
} from "@/components/data-table/types";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { useListRulesQuery, type Rule } from "@/store/api/rules-api";

export type SelectedRule = { id: number; label: string };

export function RulePicker({
  value,
  onChange,
  disabled,
  invalid,
}: {
  value: SelectedRule[];
  onChange: (next: SelectedRule[]) => void;
  disabled?: boolean;
  invalid?: boolean;
}) {
  const t = useTranslations("data-discovery.policies.dialog");
  const common = useTranslations("common");
  const policies = useTranslations("policies");

  const selected = React.useMemo(() => new Set(value.map((item) => item.id)), [value]);

  const columns: ColumnConfig<Rule>[] = [
    {
      key: "rule_name",
      label: policies("columns.rule"),
      render: (raw) => <span className="font-medium">{String(raw)}</span>,
    },
    {
      key: "type",
      label: policies("columns.type"),
      render: (raw) => <Badge variant="secondary">{String(raw)}</Badge>,
    },
    {
      key: "updated_at",
      label: policies("columns.lastModified"),
      type: "datetime",
      sortable: true,
    },
  ];

  const filters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("searchRules"),
      width: "w-64",
    },
  ];

  const rowActions: RowAction<Rule>[] = [
    {
      key: "add-rule",
      label: common("add"),
      icon: Plus,
      variant: "outline",
      hidden: (row) => selected.has(row.id),
      disabled: () => Boolean(disabled),
      onClick: (row) => onChange([...value, { id: row.id, label: row.rule_name }]),
    },
    {
      key: "remove-rule",
      label: common("remove"),
      icon: Minus,
      variant: "outline",
      hidden: (row) => !selected.has(row.id),
      disabled: () => Boolean(disabled),
      onClick: (row) => onChange(value.filter((item) => item.id !== row.id)),
    },
  ];

  return (
    <div className={cn("overflow-hidden rounded-lg border", invalid && "border-error")}>
      <div className="max-h-32 overflow-y-auto border-b px-4 py-3">
        {value.length === 0 ? (
          <p className="text-xs text-muted-foreground">{t("noSelectedRules")}</p>
        ) : (
          <div className="flex flex-wrap gap-1.5">
            {value.map((item) => (
              <Badge key={item.id} variant="secondary" className="gap-1 pr-1">
                <span className="max-w-40 truncate">{item.label}</span>
                <button
                  type="button"
                  aria-label={t("removeRule", { name: item.label })}
                  disabled={disabled}
                  onClick={() => onChange(value.filter((row) => row.id !== item.id))}
                  className="rounded-full p-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-40"
                >
                  <X className="size-3" />
                </button>
              </Badge>
            ))}
          </div>
        )}
      </div>

      <DataTable
        query={useListRulesQuery}
        columns={columns}
        filters={filters}
        rowActions={rowActions}
        getRowId={(row) => row.id}
        emptyMessage={t("noRules")}
        defaultPageSize={10}
        className="border-0"
      />
    </div>
  );
}

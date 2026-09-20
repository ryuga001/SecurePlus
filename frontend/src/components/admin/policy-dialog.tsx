"use client";

import { Check, ListFilter, Loader2, Minus, Plus, ShieldOff, X } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { DataTable } from "@/components/data-table/data-table";
import type {
  ColumnConfig,
  DataTableQueryHook,
  FilterConfig,
  RowAction,
} from "@/components/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Switch } from "@/components/ui/switch";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { apiErrorMessage, apiErrorCode } from "@/lib/api-error";
import {
  useListEmailGroupsQuery,
  type EmailGroup,
} from "@/store/api/email-groups-api";
import {
  useCreatePolicyMutation,
  useGetPolicyQuery,
  useListFileTypesQuery,
  useUpdatePolicyMutation,
  type FileType,
  type Policy,
  type PolicyAction,
  type RestrictionMode,
} from "@/store/api/policies-api";
import {
  useListRulesQuery,
  type Rule,
} from "@/store/api/rules-api";

type PoliciesTranslations = ReturnType<typeof useTranslations<"policies">>;

const FILTER_OPTIONS: { value: RestrictionMode; label: string; hint: string }[] = [
  { value: "NONE", label: "none", hint: "hintNone" },
  { value: "ALLOW", label: "allow", hint: "hintAllow" },
  { value: "BLOCK", label: "block", hint: "hintBlock" },
];

function toggle(
  ids: number[],
  id: number,
  checked: boolean,
) {
  return checked
    ? ids.includes(id)
      ? ids
      : [...ids, id]
    : ids.filter((value) => value !== id);
}

function normalizeDomains(value: string[]) {
  return Array.from(
    new Set(
      value
        .flatMap((item) => item.split(/[\s,;]+/))
        .map((item) =>
          item
            .trim()
            .toLowerCase()
            .replace(/^https?:\/\//, "")
            .replace(/^www\./, "")
            .replace(/\/.*$/, ""),
        )
        .filter(Boolean),
    ),
  );
}

function useSelectedFilter<Row extends { id: number }>(
  baseQuery: DataTableQueryHook<Row>,
  selectedIds: number[],
  onlySelected: boolean,
): DataTableQueryHook<Row> {
  return React.useCallback(
    (args) => {
      const result = baseQuery(args);

      if (!onlySelected || selectedIds.length === 0) return result;

      const data = result.data;
      if (!data) return result;

      const items = Array.isArray(data) ? data : data.items;
      const picked = new Set(selectedIds);
      const filtered = items.filter((row) => picked.has(row.id));

      return {
        ...result,
        data: Array.isArray(data)
          ? filtered
          : { ...data, items: filtered },
      };
    },
    [baseQuery, selectedIds, onlySelected],
  );
}

function SelectionFilter({
  count,
  onlySelected,
  onOnlySelectedChange,
  t,
}: {
  count: number;
  onlySelected: boolean;
  onOnlySelectedChange: (value: boolean) => void;
  t: PoliciesTranslations;
}) {
  return (
    <div className="flex items-center gap-3">
      <div className="text-sm tabular-nums">
        <span className="font-semibold">{count}</span>{" "}
        <span className="text-muted-foreground">
          {t("dialog.selected")}
        </span>
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={count === 0}
              className="min-w-32 justify-between"
            >
              <ListFilter />
              {onlySelected
                ? t("dialog.onlySelected")
                : t("dialog.showAll")}
            </Button>
          }
        />

        <DropdownMenuContent align="end">
          <DropdownMenuItem
            disabled={count === 0}
            onClick={() => onOnlySelectedChange(true)}
          >
            {onlySelected ? <Check /> : <span className="size-4" />}
            {t("dialog.onlySelected")}
          </DropdownMenuItem>

          <DropdownMenuItem onClick={() => onOnlySelectedChange(false)}>
            {!onlySelected ? <Check /> : <span className="size-4" />}
            {t("dialog.showAll")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

function RestrictionModeSelector({
  value,
  disabled,
  onChange,
  t,
}: {
  value: RestrictionMode;
  disabled: boolean;
  onChange: (value: RestrictionMode) => void;
  t: PoliciesTranslations;
}) {
  return (
    <RadioGroup
      value={value}
      disabled={disabled}
      onValueChange={(next) => onChange(next as RestrictionMode)}
      name="restriction-mode"
      className="flex flex-col gap-2"
    >
      {FILTER_OPTIONS.map((option) => {
        const selected = value === option.value;

        return (
          <label
            key={option.value}
            className={cn(
              "flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition",
              selected
                ? "border-primary/60 bg-primary/5"
                : "hover:bg-muted/50",
            )}
          >
            <RadioGroupItem value={option.value} />
            <div className="min-w-0">
              <div className="text-sm font-medium">
                {t(`dialog.restriction${option.label === "none" ? "None" : option.label === "allow" ? "Allow" : "Block"}`)}
              </div>
              <div className="mt-0.5 text-xs leading-relaxed text-muted-foreground">
                {t(`dialog.${option.hint}`)}
              </div>
            </div>
          </label>
        );
      })}
    </RadioGroup>
  );
}

function RestrictionOff({
  title,
  hint,
}: {
  title: string;
  hint: string;
}) {
  return (
    <div className="flex min-h-40 flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-6 text-center">
      <ShieldOff className="size-5 text-muted-foreground" />
      <p className="text-sm font-medium">{title}</p>
      <p className="max-w-xs text-xs text-muted-foreground">{hint}</p>
    </div>
  );
}

function DomainRestriction({
  mode,
  domains,
  disabled,
  onModeChange,
  onDomainsChange,
  t,
}: {
  mode: RestrictionMode;
  domains: string[];
  disabled: boolean;
  onModeChange: (mode: RestrictionMode) => void;
  onDomainsChange: (domains: string[]) => void;
  t: PoliciesTranslations;
}) {
  const [input, setInput] = React.useState("");

  function addDomains(value: string) {
    const parsed = normalizeDomains([value]);

    if (!parsed.length) return;

    onDomainsChange(normalizeDomains([...domains, ...parsed]));
    setInput("");
  }

  function removeDomain(domain: string) {
    onDomainsChange(domains.filter((value) => value !== domain));
  }

  const hint =
    mode === "ALLOW"
      ? t("dialog.hintAllow")
      : mode === "BLOCK"
        ? t("dialog.hintBlock")
        : t("dialog.hintNone");

  return (
    <div className="overflow-hidden rounded-lg border">
      <div className="flex items-start justify-between gap-4 border-b px-5 py-4">
        <div>
          <h3 className="text-sm font-semibold">
            {t("dialog.domainRestriction")}
          </h3>
          <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
        </div>

        {mode !== "NONE" ? (
          <Badge variant="secondary" className="shrink-0">
            {domains.length} {t("dialog.selected")}
          </Badge>
        ) : null}
      </div>

      <div className="grid gap-4 p-5 md:grid-cols-[14rem_minmax(0,1fr)]">
        <RestrictionModeSelector
          value={mode}
          disabled={disabled}
          onChange={onModeChange}
          t={t}
        />

        {mode === "NONE" ? (
          <RestrictionOff
            title={t("dialog.restrictionOff")}
            hint={t("dialog.domainRestrictionOff")}
          />
        ) : (
          <div className="flex min-h-40 flex-col gap-3">
            <div className="flex gap-2">
              <Input
                value={input}
                disabled={disabled}
                placeholder={t("dialog.domainPlaceholder")}
                onChange={(event) => setInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    event.preventDefault();
                    addDomains(input);
                  }
                }}
              />

              <Button
                type="button"
                variant="outline"
                disabled={disabled || !input.trim()}
                onClick={() => addDomains(input)}
              >
                {t("dialog.addDomain")}
              </Button>
            </div>

            <Textarea
              disabled={disabled}
              placeholder={t("dialog.domainPlaceholder")}
              className="min-h-24 font-mono text-sm"
              onPaste={(event) => {
                const pasted = event.clipboardData.getData("text");
                const parsed = normalizeDomains([pasted]);

                if (!parsed.length) return;

                event.preventDefault();

                onDomainsChange(
                  normalizeDomains([...domains, ...parsed]),
                );
              }}
            />

            {domains.length === 0 ? (
              <div className="flex min-h-24 flex-1 items-center justify-center rounded-md border border-dashed px-4 text-sm text-muted-foreground">
                {t("dialog.noDomains")}
              </div>
            ) : (
              <div className="max-h-48 overflow-y-auto rounded-md border p-3">
                <div className="flex flex-wrap gap-1.5">
                  {domains.map((domain) => (
                    <Badge key={domain} variant="secondary" className="gap-1 pr-1">
                      {domain}
                      <button
                        type="button"
                        aria-label={domain}
                        disabled={disabled}
                        onClick={() => removeDomain(domain)}
                        className="rounded-full p-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-40"
                      >
                        <X />
                      </button>
                    </Badge>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function AttachmentRestriction({
  mode,
  extensions,
  fileTypes,
  disabled,
  onModeChange,
  onExtensionsChange,
  t,
}: {
  mode: RestrictionMode;
  extensions: string[];
  fileTypes: FileType[];
  disabled: boolean;
  onModeChange: (mode: RestrictionMode) => void;
  onExtensionsChange: (extensions: string[]) => void;
  t: PoliciesTranslations;
}) {
  const hint =
    mode === "ALLOW"
      ? t("dialog.hintAllow")
      : mode === "BLOCK"
        ? t("dialog.hintBlock")
        : t("dialog.hintNone");

  return (
    <div className="overflow-hidden rounded-lg border">
      <div className="flex items-start justify-between gap-4 border-b px-5 py-4">
        <div>
          <h3 className="text-sm font-semibold">
            {t("dialog.attachmentRestriction")}
          </h3>
          <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
        </div>

        {mode !== "NONE" ? (
          <Badge variant="secondary" className="shrink-0">
            {extensions.length} {t("dialog.selected")}
          </Badge>
        ) : null}
      </div>

      <div className="grid gap-4 p-5 md:grid-cols-[14rem_minmax(0,1fr)]">
        <RestrictionModeSelector
          value={mode}
          disabled={disabled}
          onChange={onModeChange}
          t={t}
        />

        {mode === "NONE" ? (
          <RestrictionOff
            title={t("dialog.restrictionOff")}
            hint={t("dialog.attachmentRestrictionOff")}
          />
        ) : fileTypes.length === 0 ? (
          <div className="flex min-h-40 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground">
            {t("dialog.noFileTypes")}
          </div>
        ) : (
          <div className="min-h-40 rounded-lg border p-3">
            <div className="grid max-h-64 grid-cols-2 gap-2 overflow-y-auto sm:grid-cols-3">
              {fileTypes.map((fileType) => {
                const selected = extensions.includes(fileType.extension);

                return (
                  <label
                    key={fileType.id}
                    className={cn(
                      "flex cursor-pointer items-center gap-2 rounded-md border p-2 transition hover:bg-muted/50",
                      selected && "border-primary/60 bg-primary/5",
                    )}
                  >
                    <Switch
                      checked={selected}
                      disabled={disabled}
                      onCheckedChange={(checked) => {
                        onExtensionsChange(
                          checked
                            ? Array.from(
                                new Set([
                                  ...extensions,
                                  fileType.extension,
                                ]),
                              )
                            : extensions.filter(
                                (value) =>
                                  value !== fileType.extension,
                              ),
                        );
                      }}
                    />

                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium">
                        .{fileType.extension}
                      </div>
                      <div className="truncate text-xs text-muted-foreground">
                        {fileType.label}
                      </div>
                    </div>
                  </label>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function PolicyForm({
  policy,
  fileTypes,
  onClose,
}: {
  policy: Policy | null;
  fileTypes: FileType[];
  onClose: () => void;
}) {
  const t = useTranslations("policies");
  const status = useTranslations("status");
  const common = useTranslations("common");

  const [createPolicy, { isLoading: creating }] =
    useCreatePolicyMutation();

  const [updatePolicy, { isLoading: updating }] =
    useUpdatePolicyMutation();

  const policyId = policy?.id ?? null;

  const [name, setName] = React.useState(policy?.policy_name ?? "");
  const [action, setAction] = React.useState<PolicyAction>(
    policy?.action ?? "AUDIT",
  );
  const [ruleIDs, setRuleIDs] = React.useState<number[]>(
    (policy?.rules ?? []).map((rule) => rule.id),
  );
  const [groupIDs, setGroupIDs] = React.useState<number[]>(
    (policy?.groups ?? []).map((group) => group.id),
  );
  const [domainMode, setDomainMode] =
    React.useState<RestrictionMode>(
      policy?.domain_restriction.mode ?? "NONE",
    );
  const [domains, setDomains] = React.useState<string[]>(
    policy?.domain_restriction.values ?? [],
  );
  const [attachmentMode, setAttachmentMode] =
    React.useState<RestrictionMode>(
      policy?.attachment_restriction.mode ?? "NONE",
    );
  const [extensions, setExtensions] = React.useState<string[]>(
    policy?.attachment_restriction.values ?? [],
  );
  const [rulesOnlySelected, setRulesOnlySelected] =
    React.useState(false);
  const [groupsOnlySelected, setGroupsOnlySelected] =
    React.useState(false);
  const [error, setError] = React.useState("");

  const pending = creating || updating;

  const rulesOnly = rulesOnlySelected && ruleIDs.length > 0;
  const groupsOnly = groupsOnlySelected && groupIDs.length > 0;

  const rulesQuery = useSelectedFilter(
    useListRulesQuery,
    ruleIDs,
    rulesOnly,
  );
  const groupsQuery = useSelectedFilter(
    useListEmailGroupsQuery,
    groupIDs,
    groupsOnly,
  );

  const actionOptions: { label: string; value: PolicyAction }[] = [
    { label: status("audit"), value: "AUDIT" },
    { label: status("block"), value: "BLOCK" },
    { label: status("quarantine"), value: "QUARANTINE" },
    { label: status("redact"), value: "REDACT" },
  ];

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmedName = name.trim();

    if (!trimmedName) {
      setError(t("dialog.nameRequired"));
      return;
    }

    if (ruleIDs.length === 0) {
      setError(t("dialog.rulesRequired"));
      return;
    }

    if (groupIDs.length === 0) {
      setError(t("dialog.groupsRequired"));
      return;
    }

    if (domainMode !== "NONE" && domains.length === 0) {
      setError(t("dialog.domainsRequired"));
      return;
    }

    if (attachmentMode !== "NONE" && extensions.length === 0) {
      setError(t("dialog.fileTypesRequired"));
      return;
    }

    setError("");

    const input = {
      policy_name: trimmedName,
      action,
      active: true,
      group_ids: groupIDs,
      rule_ids: ruleIDs,
      domain_restriction: {
        mode: domainMode,
        values: domainMode === "NONE" ? [] : domains,
      },
      attachment_restriction: {
        mode: attachmentMode,
        values: attachmentMode === "NONE" ? [] : extensions,
      },
    };

    try {
      if (policyId === null) {
        await createPolicy(input).unwrap();
        toast.success(t("dialog.created"));
      } else {
        await updatePolicy({ id: policyId, ...input }).unwrap();
        toast.success(t("dialog.updated"));
      }

      onClose();
    } catch (caught) {
      const code = apiErrorCode(caught);

      setError(
        apiErrorMessage(
          caught,
          code === "group_not_found" || code === "rule_not_found"
            ? t("dialog.staleSelection")
            : t("dialog.saveFailed"),
        ),
      );
    }
  }

  const ruleColumns: ColumnConfig<Rule>[] = [
    {
      key: "rule_name",
      label: t("columns.rule"),
      render: (value) => (
        <span className="font-medium">{String(value)}</span>
      ),
    },
    {
      key: "type",
      label: t("columns.type"),
      render: (value) => (
        <Badge variant="secondary">{String(value)}</Badge>
      ),
    },
    {
      key: "updated_at",
      label: t("columns.lastModified"),
      type: "datetime",
      sortable: true,
    },
  ];

  const groupColumns: ColumnConfig<EmailGroup>[] = [
    {
      key: "name",
      label: t("columns.group"),
      render: (value) => (
        <span className="font-medium">{String(value)}</span>
      ),
    },
    {
      key: "member_count",
      label: t("columns.members"),
      render: (value) => (
        <Badge variant="secondary">{String(value)}</Badge>
      ),
    },
    {
      key: "updated_at",
      label: t("columns.lastModified"),
      type: "datetime",
      sortable: true,
    },
  ];

  const ruleFilters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("dialog.searchRules"),
      width: "w-64",
    },
  ];

  const groupFilters: FilterConfig[] = [
    {
      key: "search",
      label: common("search"),
      type: "text",
      placeholder: t("dialog.searchGroups"),
      width: "w-64",
    },
  ];

  const ruleRowActions: RowAction<Rule>[] = [
    {
      key: "add-rule",
      label: common("add"),
      icon: Plus,
      variant: "outline",
      hidden: (row) => ruleIDs.includes(row.id),
      disabled: () => pending,
      onClick: (row) => {
        setRuleIDs((ids) => toggle(ids, row.id, true));
        setError("");
      },
    },
    {
      key: "remove-rule",
      label: common("remove"),
      icon: Minus,
      variant: "outline",
      hidden: (row) => !ruleIDs.includes(row.id),
      disabled: () => pending,
      onClick: (row) => {
        setRuleIDs((ids) => toggle(ids, row.id, false));
        setError("");
      },
    },
  ];

  const groupRowActions: RowAction<EmailGroup>[] = [
    {
      key: "add-group",
      label: common("add"),
      icon: Plus,
      variant: "outline",
      hidden: (row) => groupIDs.includes(row.id),
      disabled: () => pending,
      onClick: (row) => {
        setGroupIDs((ids) => toggle(ids, row.id, true));
        setError("");
      },
    },
    {
      key: "remove-group",
      label: common("remove"),
      icon: Minus,
      variant: "outline",
      hidden: (row) => !groupIDs.includes(row.id),
      disabled: () => pending,
      onClick: (row) => {
        setGroupIDs((ids) => toggle(ids, row.id, false));
        setError("");
      },
    },
  ];

  return (
    <form
      onSubmit={onSubmit}
      className="flex min-h-0 flex-1 flex-col"
    >
      <FormError message={error} />

      <Tabs
        defaultValue="general"
        className="mt-4 flex min-h-0 flex-1 flex-col"
      >
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="general">
            {t("tabs.general")}
          </TabsTrigger>
          <TabsTrigger value="restrictions">
            {t("tabs.restrictions")}
          </TabsTrigger>
          <TabsTrigger value="rules">
            {t("tabs.rules")}
          </TabsTrigger>
          <TabsTrigger value="groups">
            {t("tabs.groups")}
          </TabsTrigger>
        </TabsList>

        <div className="min-h-0 flex-1 overflow-y-auto py-6">
          <TabsContent value="general" className="mt-0 space-y-4">
            <Field id="policy-name" label={t("dialog.name")}>
              <Input
                id="policy-name"
                required
                autoFocus
                placeholder={t("dialog.namePlaceholder")}
                value={name}
                disabled={pending}
                onChange={(event) => {
                  setName(event.target.value);
                  setError("");
                }}
              />
            </Field>

            <Field
              id="policy-action"
              label={t("dialog.action")}
              hint={t("dialog.actionHint")}
            >
              <RadioGroup
                value={action}
                onValueChange={(next) =>
                  setAction(next as PolicyAction)
                }
                className="grid gap-2 sm:grid-cols-2"
              >
                {actionOptions.map((option) => (
                  <label
                    key={option.value}
                    className={cn(
                      "flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition",
                      action === option.value
                        ? "border-primary bg-primary/5"
                        : "hover:bg-muted/50",
                    )}
                  >
                    <RadioGroupItem value={option.value} />
                    <span className="text-sm font-medium">
                      {option.label}
                    </span>
                  </label>
                ))}
              </RadioGroup>
            </Field>

            <div className="rounded-lg border bg-surface-container-low p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <div className="text-sm font-medium">
                    {t("dialog.status")}
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {t("dialog.alwaysEnabled")}
                  </div>
                </div>

                <div className="flex shrink-0 items-center gap-2">
                  <Switch checked disabled />
                  <span className="text-sm font-medium">
                    {status("enabled")}
                  </span>
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="restrictions" className="mt-0 space-y-6">
            <DomainRestriction
              mode={domainMode}
              domains={domains}
              disabled={pending}
              onModeChange={(mode) => {
                setDomainMode(mode);
                setError("");
              }}
              onDomainsChange={(values) => {
                setDomains(values);
                setError("");
              }}
              t={t}
            />

            <AttachmentRestriction
              mode={attachmentMode}
              extensions={extensions}
              fileTypes={fileTypes}
              disabled={pending}
              onModeChange={(mode) => {
                setAttachmentMode(mode);
                setError("");
              }}
              onExtensionsChange={(values) => {
                setExtensions(values);
                setError("");
              }}
              t={t}
            />
          </TabsContent>

          <TabsContent value="rules" className="mt-0 space-y-4">
            <div>
              <h3 className="text-sm font-semibold">
                {t("dialog.rules")}
              </h3>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("dialog.rulesHint")}
              </p>
            </div>

            <div className="overflow-hidden rounded-lg border">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <SelectionFilter
                  count={ruleIDs.length}
                  onlySelected={rulesOnly}
                  onOnlySelectedChange={setRulesOnlySelected}
                  t={t}
                />

                <span className="text-xs text-muted-foreground">
                  {t("dialog.clickToSelect")}
                </span>
              </div>

              <DataTable
                query={rulesQuery}
                columns={ruleColumns}
                filters={ruleFilters}
                rowActions={ruleRowActions}
                getRowId={(row) => row.id}
                emptyMessage={t("dialog.noRules")}
                className="border-0"
              />
            </div>
          </TabsContent>

          <TabsContent value="groups" className="mt-0 space-y-4">
            <div>
              <h3 className="text-sm font-semibold">
                {t("dialog.groups")}
              </h3>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("dialog.groupsHint")}
              </p>
            </div>

            <div className="overflow-hidden rounded-lg border">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <SelectionFilter
                  count={groupIDs.length}
                  onlySelected={groupsOnly}
                  onOnlySelectedChange={setGroupsOnlySelected}
                  t={t}
                />

                <span className="text-xs text-muted-foreground">
                  {t("dialog.clickToSelect")}
                </span>
              </div>

              <DataTable
                query={groupsQuery}
                columns={groupColumns}
                filters={groupFilters}
                rowActions={groupRowActions}
                getRowId={(row) => row.id}
                emptyMessage={t("dialog.noGroups")}
                className="border-0"
              />
            </div>
          </TabsContent>
        </div>
      </Tabs>

      <div className="flex shrink-0 justify-end gap-2 border-t pt-4">
        <Button
          type="button"
          variant="outline"
          disabled={pending}
          onClick={onClose}
        >
          {common("cancel")}
        </Button>

        <Button type="submit" disabled={pending}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          {policyId === null
            ? t("dialog.create")
            : t("dialog.saveChanges")}
        </Button>
      </div>
    </form>
  );
}

function PolicyLoader({
  policyId,
  onClose,
}: {
  policyId: number | null;
  onClose: () => void;
}) {
  const t = useTranslations("policies");

  const {
    data: policy,
    isLoading: loadingPolicy,
  } = useGetPolicyQuery(policyId as number, {
    skip: policyId === null,
  });

  const {
    data: fileTypes,
    isLoading: loadingFileTypes,
  } = useListFileTypesQuery();

  if (loadingPolicy || loadingFileTypes) {
    return (
      <div className="flex min-h-48 items-center justify-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        {t("dialog.oneMoment")}
      </div>
    );
  }

  return (
    <PolicyForm
      key={policyId ?? "create"}
      policy={policy ?? null}
      fileTypes={fileTypes?.items ?? []}
      onClose={onClose}
    />
  );
}

export function PolicyDialog({
  open,
  onOpenChange,
  policyId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  policyId?: number | null;
}) {
  const t = useTranslations("policies");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={
        policyId == null
          ? t("dialog.addTitle")
          : t("dialog.editTitle")
      }
      description={t("dialog.description")}
      width="3xl"
    >
      <PolicyLoader
        policyId={policyId ?? null}
        onClose={() => onOpenChange(false)}
      />
    </DrawerWrapper>
  );
}
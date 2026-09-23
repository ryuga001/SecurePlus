export const POLICY_TABS = ["general", "source", "fileTypes", "rules"] as const;

export type PolicyTab = (typeof POLICY_TABS)[number];

export type PolicyFieldKey =
  | "name"
  | "description"
  | "status"
  | "source_type"
  | "configuration_id"
  | "targets"
  | "file_types"
  | "rule_ids";

export const TAB_BY_FIELD: Record<PolicyFieldKey, PolicyTab> = {
  name: "general",
  description: "general",
  status: "general",
  source_type: "source",
  configuration_id: "source",
  targets: "source",
  file_types: "fileTypes",
  rule_ids: "rules",
};

const BACKEND_FIELD_ALIASES: Record<string, PolicyFieldKey> = {
  policy_name: "name",
  configuration: "configuration_id",
  configuration_id: "configuration_id",
  target_list: "targets",
  targets: "targets",
  file_types: "file_types",
  rules: "rule_ids",
  rule_ids: "rule_ids",
  source_type: "source_type",
  name: "name",
  description: "description",
  status: "status",
};

const FIELD_PATH = /^([a-z_]+)(?:\[(\d+)\]|\.(\d+))?(?:\.([a-z_]+))?$/i;

export function parseBackendField(field: string): {
  root?: PolicyFieldKey;
  index?: number;
  leaf?: string;
} {
  const match = FIELD_PATH.exec(field);
  if (!match) return {};

  const [, rawRoot, bracketIndex, dotIndex, leaf] = match;
  const root = BACKEND_FIELD_ALIASES[rawRoot];
  if (!root) return {};

  const index = bracketIndex ?? dotIndex;

  return {
    root,
    index: index === undefined ? undefined : Number(index),
    leaf,
  };
}

export function tabForBackendField(field: string): PolicyTab | undefined {
  const { root } = parseBackendField(field);

  return root ? TAB_BY_FIELD[root] : undefined;
}

export function firstInvalidTab(
  errors: Partial<Record<PolicyFieldKey, string>>,
): PolicyTab | undefined {
  for (const tab of POLICY_TABS) {
    const hasError = (Object.keys(errors) as PolicyFieldKey[]).some(
      (field) => errors[field] && TAB_BY_FIELD[field] === tab,
    );

    if (hasError) return tab;
  }

  return undefined;
}

export function firstInvalidField(
  errors: Partial<Record<PolicyFieldKey, string>>,
): PolicyFieldKey | undefined {
  const order = (Object.keys(TAB_BY_FIELD) as PolicyFieldKey[]).sort(
    (a, b) => POLICY_TABS.indexOf(TAB_BY_FIELD[a]) - POLICY_TABS.indexOf(TAB_BY_FIELD[b]),
  );

  return order.find((field) => errors[field]);
}

export function tabErrorCount(
  errors: Partial<Record<PolicyFieldKey, string>>,
  tab: PolicyTab,
) {
  return (Object.keys(errors) as PolicyFieldKey[]).filter(
    (field) => errors[field] && TAB_BY_FIELD[field] === tab,
  ).length;
}

export type PrivilegeEntry = {
  name: string;
  group: string;
  action: string;
};

export type PrivilegeGroup = {
  key: string;
  entries: PrivilegeEntry[];
};

const GROUP_ORDER = [
  "email.provider",
  "email.user",
  "email.group",
  "rule",
  "policy",
  "email.alert",
  "email.audit",
  "email.incident",
  "discovery.configuration",
  "discovery.policy",
  "discovery.scan",
  "branding",
  "organization",
];

const ACTION_ORDER = ["view", "create", "edit", "delete", "test"];

function rank(order: string[], value: string) {
  const index = order.indexOf(value);
  return index === -1 ? order.length : index;
}

export function parsePrivilege(name: string): PrivilegeEntry {
  const parts = name.split(".");
  const action = parts.at(-1) ?? name;
  const body = parts[0] === "admin" ? parts.slice(1, -1) : parts.slice(0, -1);

  return { name, group: body.join(".") || name, action };
}

export function groupPrivileges(names: readonly string[]): PrivilegeGroup[] {
  const groups = new Map<string, PrivilegeEntry[]>();

  for (const name of names) {
    const entry = parsePrivilege(name);
    groups.set(entry.group, [...(groups.get(entry.group) ?? []), entry]);
  }

  return Array.from(groups, ([key, entries]) => ({
    key,
    entries: entries.sort(
      (a, b) => rank(ACTION_ORDER, a.action) - rank(ACTION_ORDER, b.action) || a.action.localeCompare(b.action),
    ),
  })).sort((a, b) => rank(GROUP_ORDER, a.key) - rank(GROUP_ORDER, b.key) || a.key.localeCompare(b.key));
}

export function messageKey(value: string) {
  return value.replaceAll(".", "_");
}

export function humanize(value: string) {
  const text = value.replaceAll(/[._-]+/g, " ").trim();
  return text.charAt(0).toUpperCase() + text.slice(1);
}

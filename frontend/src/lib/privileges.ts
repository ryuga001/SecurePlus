export const ALERT_PRIVILEGES = {
  view: "admin.email.alert.view",
  create: "admin.email.alert.create",
  edit: "admin.email.alert.edit",
  delete: "admin.email.alert.delete",
} as const;

export const DISCOVERY_CONFIGURATION_PRIVILEGES = {
  view: "admin.discovery.configuration.view",
  create: "admin.discovery.configuration.create",
  edit: "admin.discovery.configuration.edit",
  delete: "admin.discovery.configuration.delete",
  test: "admin.discovery.configuration.test",
} as const;

export const DISCOVERY_POLICY_PRIVILEGES = {
  view: "admin.discovery.policy.view",
  create: "admin.discovery.policy.create",
  edit: "admin.discovery.policy.edit",
  delete: "admin.discovery.policy.delete",
} as const;

export function hasPrivilege(granted: readonly string[] | undefined, required: string) {
  return granted?.includes(required) ?? false;
}

export function hasAnyPrivilege(
  granted: readonly string[] | undefined,
  required: readonly string[],
) {
  return required.some((item) => hasPrivilege(granted, item));
}

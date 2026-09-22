export const ALERT_PRIVILEGES = {
  view: "admin.email.alert.view",
  create: "admin.email.alert.create",
  edit: "admin.email.alert.edit",
  delete: "admin.email.alert.delete",
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

"use client";

import { Loader2, Lock } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { PrivilegeMatrix } from "@/components/console/privilege-matrix";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { FormSkeleton } from "@/components/loading/skeletons";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { apiErrorCode, apiErrorMessage } from "@/lib/api-error";
import {
  useCreateRoleMutation,
  useGetRoleQuery,
  useListAssignablePrivilegesQuery,
  useUpdateRoleMutation,
  type Role,
} from "@/store/api/roles-api";

const NAME_MIN = 2;
const NAME_MAX = 50;
const DESCRIPTION_MAX = 255;

type FieldKey = "name" | "description" | "privileges";
type FieldErrors = Partial<Record<FieldKey, string>>;

function normalize(value: string) {
  return value.trim().split(/\s+/).filter(Boolean).join(" ");
}

function RoleForm({
  role,
  available,
  onClose,
}: {
  role: Role | null;
  available: string[];
  onClose: () => void;
}) {
  const t = useTranslations("console.roles.dialog");
  const common = useTranslations("common");

  const [createRole, { isLoading: creating }] = useCreateRoleMutation();
  const [updateRole, { isLoading: updating }] = useUpdateRoleMutation();

  const [name, setName] = React.useState(role?.name ?? "");
  const [description, setDescription] = React.useState(role?.description ?? "");
  const [privileges, setPrivileges] = React.useState<string[]>(role?.privileges ?? []);
  const [fieldErrors, setFieldErrors] = React.useState<FieldErrors>({});
  const [error, setError] = React.useState("");

  const readOnly = role?.system ?? false;
  const pending = creating || updating;

  function clear(field: FieldKey) {
    setFieldErrors((current) => ({ ...current, [field]: undefined }));
    setError("");
  }

  function validate(): FieldErrors {
    const errors: FieldErrors = {};
    const length = Array.from(normalize(name)).length;

    if (length < NAME_MIN || length > NAME_MAX) {
      errors.name = t("errors.nameLength", { min: NAME_MIN, max: NAME_MAX });
    }
    if (Array.from(description.trim()).length > DESCRIPTION_MAX) {
      errors.description = t("errors.descriptionLength", { max: DESCRIPTION_MAX });
    }
    if (privileges.length === 0) {
      errors.privileges = t("errors.privilegesRequired");
    }

    return errors;
  }

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    if (readOnly) return;

    const errors = validate();
    const first = (["name", "description", "privileges"] as const).find((field) => errors[field]);

    if (first) {
      setFieldErrors(errors);
      document.getElementById(`role-${first}`)?.focus();
      return;
    }

    const body = { name: normalize(name), description: description.trim(), privileges };

    try {
      if (role) {
        await updateRole({ id: role.id, ...body }).unwrap();
        toast.success(t("updated", { name: body.name }));
      } else {
        await createRole(body).unwrap();
        toast.success(t("created", { name: body.name }));
      }

      onClose();
    } catch (cause) {
      switch (apiErrorCode(cause)) {
        case "role_name_taken":
          setFieldErrors({ name: t("errors.nameTaken") });
          document.getElementById("role-name")?.focus();
          break;
        case "privilege_not_found":
          setFieldErrors({ privileges: t("errors.privilegeUnavailable") });
          break;
        case "system_role_immutable":
          setError(t("errors.systemRole"));
          break;
        default:
          setError(apiErrorMessage(cause, t("saveFailed")));
      }
    }
  }

  return (
    <form noValidate onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col gap-5">
      {readOnly ? (
        <div className="flex items-start gap-2.5 rounded-md border bg-surface-container-low p-3 text-sm text-muted-foreground">
          <Lock className="mt-0.5 size-4 shrink-0" />
          <span>{t("systemNotice")}</span>
        </div>
      ) : null}

      <FormError message={error} />

      <Field id="role-name" label={t("name")} error={fieldErrors.name} hint={t("nameHint", { max: NAME_MAX })}>
        <Input
          id="role-name"
          autoFocus={!readOnly}
          maxLength={NAME_MAX}
          placeholder={t("namePlaceholder")}
          value={name}
          readOnly={readOnly}
          disabled={pending}
          aria-invalid={fieldErrors.name ? true : undefined}
          onChange={(event) => {
            setName(event.target.value);
            clear("name");
          }}
        />
      </Field>

      <Field id="role-description" label={t("descriptionField")} error={fieldErrors.description}>
        <Textarea
          id="role-description"
          rows={2}
          maxLength={DESCRIPTION_MAX}
          placeholder={t("descriptionPlaceholder")}
          value={description}
          readOnly={readOnly}
          disabled={pending}
          aria-invalid={fieldErrors.description ? true : undefined}
          onChange={(event) => {
            setDescription(event.target.value);
            clear("description");
          }}
        />
      </Field>

      <div className="flex flex-col gap-2">
        <div className="flex flex-col gap-0.5">
          <span className="text-[0.8125rem] leading-4.5 font-medium">{t("privileges")}</span>
          <span className="text-xs text-muted-foreground">{t("privilegesHint")}</span>
        </div>

        {fieldErrors.privileges ? (
          <p role="alert" className="text-xs text-error-text">
            {fieldErrors.privileges}
          </p>
        ) : null}

        <PrivilegeMatrix
          id="role-privileges"
          available={available}
          value={privileges}
          readOnly={readOnly}
          disabled={pending}
          invalid={Boolean(fieldErrors.privileges)}
          onChange={(next) => {
            setPrivileges(next);
            clear("privileges");
          }}
        />
      </div>

      <div className="mt-auto flex shrink-0 justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
          {readOnly ? common("close") : common("cancel")}
        </Button>

        {!readOnly ? (
          <Button type="submit" disabled={pending}>
            {pending ? <Loader2 className="animate-spin" /> : null}
            {role ? t("saveChanges") : t("create")}
          </Button>
        ) : null}
      </div>
    </form>
  );
}

function RoleLoader({ roleId, onClose }: { roleId: number | null; onClose: () => void }) {
  const t = useTranslations("console.roles.dialog");
  const common = useTranslations("common");

  const role = useGetRoleQuery(roleId ?? 0, { skip: roleId === null });
  const privileges = useListAssignablePrivilegesQuery();

  if (role.isLoading || privileges.isLoading) {
    return <FormSkeleton fields={3} label={t("loading")} />;
  }

  if (role.isError || privileges.isError) {
    return (
      <div className="flex flex-col gap-4">
        <FormError message={apiErrorMessage(role.error ?? privileges.error, t("loadFailed"))} />

        <div className="flex justify-end">
          <Button type="button" variant="outline" onClick={onClose}>
            {common("close")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <RoleForm
      key={roleId ?? "create"}
      role={roleId === null ? null : (role.data ?? null)}
      available={privileges.data ?? []}
      onClose={onClose}
    />
  );
}

export function RoleDrawer({
  open,
  onOpenChange,
  roleId,
  system = false,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  roleId: number | null;
  system?: boolean;
}) {
  const t = useTranslations("console.roles.dialog");

  const title = roleId === null ? t("addTitle") : system ? t("viewTitle") : t("editTitle");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={title}
      description={t("subtitle")}
      width="2xl"
    >
      {open ? <RoleLoader roleId={roleId} onClose={() => onOpenChange(false)} /> : null}
    </DrawerWrapper>
  );
}

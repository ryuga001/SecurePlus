"use client";

import { CircleAlert, Loader2, Lock } from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { apiErrorCode, apiErrorMessage } from "@/lib/api-error";
import { useListRoleOptionsQuery } from "@/store/api/roles-api";
import {
  useCreateUserMutation,
  useUpdateUserMutation,
  type User,
} from "@/store/api/users-api";

const NAME_MAX = 50;
const EMAIL_MAX = 100;
const EMAIL_PATTERN = /^[^\s@<>,;"]+@[^\s@<>,;"]+\.[^\s@<>,;"]+$/;

type FieldKey = "first_name" | "last_name" | "email" | "role_id";
type FieldErrors = Partial<Record<FieldKey, string>>;

function normalize(value: string) {
  return value.trim().split(/\s+/).filter(Boolean).join(" ");
}

function RolePicker({
  value,
  disabled,
  invalid,
  onChange,
}: {
  value: string;
  disabled: boolean;
  invalid: boolean;
  onChange: (value: string) => void;
}) {
  const t = useTranslations("console.users.dialog");
  const table = useTranslations("table");
  const { data, isLoading, isError, refetch } = useListRoleOptionsQuery();

  if (isLoading) return <Skeleton className="h-10 w-full" />;

  if (isError) {
    return (
      <div className="flex items-center justify-between gap-3 rounded-md border border-error-border bg-error-container px-3 py-2 text-sm text-error-text">
        <span className="flex items-center gap-2">
          <CircleAlert className="size-4 shrink-0" />
          {t("rolesFailed")}
        </span>
        <Button type="button" size="sm" variant="outline" onClick={() => refetch()}>
          {table("tryAgain")}
        </Button>
      </div>
    );
  }

  const options = data ?? [];

  if (options.length === 0) {
    return (
      <p className="rounded-md border border-dashed px-3 py-2.5 text-sm text-muted-foreground">
        {t("noRoles")}{" "}
        <Link href="/admin/console/user-roles" className="font-medium text-primary hover:underline">
          {t("createRole")}
        </Link>
      </p>
    );
  }

  return (
    <Select
      id="user-role_id"
      value={value}
      disabled={disabled}
      aria-invalid={invalid || undefined}
      placeholder={t("rolePlaceholder")}
      options={options.map((option) => ({
        value: String(option.id),
        label: option.system ? `${option.name} · ${t("system")}` : option.name,
      }))}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}

function UserForm({
  user,
  self,
  onClose,
}: {
  user: User | null;
  self: boolean;
  onClose: () => void;
}) {
  const t = useTranslations("console.users.dialog");
  const common = useTranslations("common");

  const [createUser, { isLoading: creating }] = useCreateUserMutation();
  const [updateUser, { isLoading: updating }] = useUpdateUserMutation();

  const [firstName, setFirstName] = React.useState(user?.first_name ?? "");
  const [lastName, setLastName] = React.useState(user?.last_name ?? "");
  const [email, setEmail] = React.useState(user?.email ?? "");
  const [roleId, setRoleId] = React.useState(user?.role ? String(user.role.id) : "");
  const [fieldErrors, setFieldErrors] = React.useState<FieldErrors>({});
  const [error, setError] = React.useState("");

  const pending = creating || updating;

  function clear(field: FieldKey) {
    setFieldErrors((current) => ({ ...current, [field]: undefined }));
    setError("");
  }

  function validate(): FieldErrors {
    const errors: FieldErrors = {};

    for (const [field, value] of [
      ["first_name", firstName],
      ["last_name", lastName],
    ] as const) {
      const size = Array.from(normalize(value)).length;

      if (size === 0) errors[field] = t("errors.nameRequired");
      else if (size > NAME_MAX) errors[field] = t("errors.nameLength", { max: NAME_MAX });
    }

    if (!user) {
      const address = email.trim();
      if (!EMAIL_PATTERN.test(address) || address.length > EMAIL_MAX) errors.email = t("errors.emailInvalid");
    }

    if (!roleId) errors.role_id = t("errors.roleRequired");

    return errors;
  }

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault();

    const errors = validate();
    const first = (["first_name", "last_name", "email", "role_id"] as const).find((field) => errors[field]);

    if (first) {
      setFieldErrors(errors);
      document.getElementById(`user-${first}`)?.focus();
      return;
    }

    const names = { first_name: normalize(firstName), last_name: normalize(lastName), role_id: Number(roleId) };

    try {
      if (user) {
        await updateUser({ id: user.id, ...names }).unwrap();
        toast.success(t("updated", { name: `${names.first_name} ${names.last_name}` }));
      } else {
        const result = await createUser({ ...names, email: email.trim() }).unwrap();

        if (result.invitation_sent) toast.success(t("invited", { email: result.user.email }));
        else toast.warning(t("inviteFailed", { email: result.user.email }));
      }

      onClose();
    } catch (cause) {
      switch (apiErrorCode(cause)) {
        case "email_taken":
          setFieldErrors({ email: t("errors.emailTaken") });
          document.getElementById("user-email")?.focus();
          break;
        case "invalid_email":
          setFieldErrors({ email: t("errors.emailInvalid") });
          document.getElementById("user-email")?.focus();
          break;
        case "role_not_found":
          setFieldErrors({ role_id: t("errors.roleUnavailable") });
          break;
        case "last_administrator":
          setFieldErrors({ role_id: t("errors.lastAdministrator") });
          break;
        case "self_modification":
          setFieldErrors({ role_id: t("errors.selfRole") });
          break;
        default:
          setError(apiErrorMessage(cause, t("saveFailed")));
      }
    }
  }

  return (
    <form noValidate onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col gap-5">
      <FormError message={error} />

      <div className="grid gap-5 sm:grid-cols-2">
        <Field id="user-first_name" label={t("firstName")} error={fieldErrors.first_name}>
          <Input
            id="user-first_name"
            autoFocus
            autoComplete="off"
            maxLength={NAME_MAX}
            value={firstName}
            disabled={pending}
            aria-invalid={fieldErrors.first_name ? true : undefined}
            onChange={(event) => {
              setFirstName(event.target.value);
              clear("first_name");
            }}
          />
        </Field>

        <Field id="user-last_name" label={t("lastName")} error={fieldErrors.last_name}>
          <Input
            id="user-last_name"
            autoComplete="off"
            maxLength={NAME_MAX}
            value={lastName}
            disabled={pending}
            aria-invalid={fieldErrors.last_name ? true : undefined}
            onChange={(event) => {
              setLastName(event.target.value);
              clear("last_name");
            }}
          />
        </Field>
      </div>

      <Field
        id="user-email"
        label={t("email")}
        error={fieldErrors.email}
        hint={user ? t("emailLocked") : t("emailHint")}
      >
        <div className="relative">
          <Input
            id="user-email"
            type="email"
            spellCheck={false}
            autoComplete="off"
            maxLength={EMAIL_MAX}
            placeholder={t("emailPlaceholder")}
            value={email}
            readOnly={Boolean(user)}
            disabled={pending}
            className={user ? "bg-muted pr-10 text-muted-foreground" : undefined}
            aria-invalid={fieldErrors.email ? true : undefined}
            onChange={(event) => {
              setEmail(event.target.value);
              clear("email");
            }}
          />
          {user ? (
            <Lock
              aria-hidden
              className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground"
            />
          ) : null}
        </div>
      </Field>

      <Field
        id="user-role_id"
        label={t("role")}
        error={fieldErrors.role_id}
        hint={self ? t("roleLockedSelf") : t("roleHint")}
      >
        <RolePicker
          value={roleId}
          disabled={pending || self}
          invalid={Boolean(fieldErrors.role_id)}
          onChange={(next) => {
            setRoleId(next);
            clear("role_id");
          }}
        />
      </Field>

      <div className="mt-auto flex shrink-0 justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
          {common("cancel")}
        </Button>

        <Button type="submit" disabled={pending}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          {user ? t("saveChanges") : t("create")}
        </Button>
      </div>
    </form>
  );
}

export function UserDrawer({
  open,
  onOpenChange,
  user,
  self,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: User | null;
  self: boolean;
}) {
  const t = useTranslations("console.users.dialog");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={user ? t("editTitle") : t("addTitle")}
      description={user ? t("editDescription") : t("addDescription")}
      width="lg"
    >
      {open ? (
        <UserForm key={user?.id ?? "create"} user={user} self={self} onClose={() => onOpenChange(false)} />
      ) : null}
    </DrawerWrapper>
  );
}

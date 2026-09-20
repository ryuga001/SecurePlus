"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { DrawerWrapper } from "@/components/drawer/drawer";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiErrorMessage, apiErrorStatus } from "@/lib/api-error";
import {
  useCreateEmailUserMutation,
  useUpdateEmailUserMutation,
  type EmailUser,
} from "@/store/api/email-users-api";

function EmailUserForm({
  user,
  onClose,
}: {
  user: EmailUser | null;
  onClose: () => void;
}) {
  const t = useTranslations("emailUsers");
  const common = useTranslations("common");

  const [email, setEmail] = React.useState(user?.email ?? "");
  const [firstName, setFirstName] = React.useState(user?.first_name ?? "");
  const [lastName, setLastName] = React.useState(user?.last_name ?? "");
  const [error, setError] = React.useState("");
  const [emailInvalid, setEmailInvalid] = React.useState(false);
  const [firstNameInvalid, setFirstNameInvalid] = React.useState(false);
  const [lastNameInvalid, setLastNameInvalid] = React.useState(false);

  const [createUser, { isLoading: creating }] = useCreateEmailUserMutation();
  const [updateUser, { isLoading: updating }] = useUpdateEmailUserMutation();

  const pending = creating || updating;

  function clearErrors() {
    setError("");
    setEmailInvalid(false);
    setFirstNameInvalid(false);
    setLastNameInvalid(false);
  }

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const normalizedEmail = email.trim().toLowerCase();
    const normalizedFirstName = firstName.trim();
    const normalizedLastName = lastName.trim();

    clearErrors();

    if (!normalizedEmail) {
      setEmailInvalid(true);
      setError(t("dialog.emailRequired"));
      return;
    }

    if (!normalizedEmail.includes("@")) {
      setEmailInvalid(true);
      setError(t("dialog.emailRequired"));
      return;
    }

    if (!normalizedFirstName) {
      setFirstNameInvalid(true);
      setError(t("dialog.nameRequired"));
      return;
    }

    if (!normalizedLastName) {
      setLastNameInvalid(true);
      setError(t("dialog.nameRequired"));
      return;
    }

    const input = {
      email: normalizedEmail,
      first_name: normalizedFirstName,
      last_name: normalizedLastName,
    };

    try {
      if (user) {
        await updateUser({
          id: user.id,
          ...input,
        }).unwrap();

        toast.success(t("dialog.updated"));
      } else {
        await createUser(input).unwrap();
        toast.success(t("dialog.created"));
      }

      onClose();
    } catch (caught) {
      const status = apiErrorStatus(caught);

      setEmailInvalid(status === 400 || status === 409);
      setError(apiErrorMessage(caught, t("dialog.saveFailed")));
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4">
      <FormError message={error} />
      <Field id="user-first-name" label={t("dialog.firstName")}>
        <Input
          id="user-first-name"
          required
          placeholder={t("dialog.firstName")}
          value={firstName}
          disabled={pending}
          aria-invalid={firstNameInvalid || undefined}
          onChange={(event) => {
            setFirstName(event.target.value);
            setFirstNameInvalid(false);
            setError("");
          }}
        />
      </Field>

      <Field id="user-last-name" label={t("dialog.lastName")}>
        <Input
          id="user-last-name"
          required
          placeholder={t("dialog.lastName")}
          value={lastName}
          disabled={pending}
          aria-invalid={lastNameInvalid || undefined}
          onChange={(event) => {
            setLastName(event.target.value);
            setLastNameInvalid(false);
            setError("");
          }}
        />
      </Field>
      <Field id="user-email" label={t("dialog.email")}>
        <Input
          id="user-email"
          type="email"
          required
          autoFocus
          spellCheck={false}
          placeholder={t("dialog.emailPlaceholder")}
          value={email}
          disabled={pending}
          aria-invalid={emailInvalid || undefined}
          onChange={(event) => {
            setEmail(event.target.value);
            setEmailInvalid(false);
            setError("");
          }}
        />
      </Field>

      <div className="mt-1 flex justify-end gap-2 border-t pt-4">
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
          {user ? t("dialog.saveChanges") : t("dialog.create")}
        </Button>
      </div>
    </form>
  );
}

export function EmailUserDialog({
  open,
  onOpenChange,
  user,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user?: EmailUser | null;
}) {
  const t = useTranslations("emailUsers");

  return (
    <DrawerWrapper
      open={open}
      onClose={() => onOpenChange(false)}
      title={user ? t("dialog.editTitle") : t("dialog.addTitle")}
      width="lg"
    >
      <EmailUserForm user={user ?? null} onClose={() => onOpenChange(false)} />
    </DrawerWrapper>
  );
}

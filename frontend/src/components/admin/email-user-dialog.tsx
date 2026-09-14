"use client";

import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { Field, FormError } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { apiErrorMessage, apiErrorStatus } from "@/lib/api-error";
import {
  useCreateEmailUserMutation,
  useUpdateEmailUserMutation,
  type EmailUser,
} from "@/store/api/email-users-api";

function EmailUserForm({ user, onClose }: { user: EmailUser | null; onClose: () => void }) {
  const t = useTranslations("emailUsers");
  const common = useTranslations("common");
  const [email, setEmail] = React.useState(user?.email ?? "");
  const [firstName, setFirstName] = React.useState(user?.first_name ?? "");
  const [lastName, setLastName] = React.useState(user?.last_name ?? "");
  const [error, setError] = React.useState("");
  const [emailInvalid, setEmailInvalid] = React.useState(false);

  const [createUser, { isLoading: creating }] = useCreateEmailUserMutation();
  const [updateUser, { isLoading: updating }] = useUpdateEmailUserMutation();

  const pending = creating || updating;

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const input = {
      email: email.trim().toLowerCase(),
      first_name: firstName.trim(),
      last_name: lastName.trim(),
    };

    if (!input.email.includes("@")) {
      setEmailInvalid(true);
      setError(t("dialog.emailRequired"));
      return;
    }
    if (!input.first_name || !input.last_name) {
      setError(t("dialog.nameRequired"));
      return;
    }

    setError("");
    setEmailInvalid(false);

    try {
      if (user) {
        await updateUser({ id: user.id, ...input }).unwrap();
        toast.success(t("dialog.updated"));
      } else {
        await createUser(input).unwrap();
        toast.success(t("dialog.created"));
      }

      onClose();
    } catch (caught) {
      setEmailInvalid(apiErrorStatus(caught) === 409 || apiErrorStatus(caught) === 400);
      setError(apiErrorMessage(caught, t("dialog.saveFailed")));
    }
  }

  return (
    <>
      <DialogTitle>{user ? t("dialog.editTitle") : t("dialog.addTitle")}</DialogTitle>
      <DialogDescription>
        People whose mail this workspace protects. Email addresses are unique across the platform.
      </DialogDescription>

      <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-4">
        <FormError message={error} />

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

        <div className="grid gap-4 sm:grid-cols-2">
          <Field id="user-first-name" label={t("dialog.firstName")}>
            <Input
              id="user-first-name"
              required
              value={firstName}
              disabled={pending}
              onChange={(event) => setFirstName(event.target.value)}
            />
          </Field>

          <Field id="user-last-name" label={t("dialog.lastName")}>
            <Input
              id="user-last-name"
              required
              value={lastName}
              disabled={pending}
              onChange={(event) => setLastName(event.target.value)}
            />
          </Field>
        </div>

        <div className="flex justify-end gap-2 border-t pt-4">
          <Button type="button" variant="outline" onClick={onClose}>
            {common("cancel")}
          </Button>
          <Button type="submit" disabled={pending}>
            {pending ? <Loader2 className="animate-spin" /> : null}
            {user ? t("dialog.saveChanges") : t("dialog.create")}
          </Button>
        </div>
      </form>
    </>
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
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100svh-4rem)] max-w-lg overflow-y-auto">
        <EmailUserForm user={user ?? null} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}

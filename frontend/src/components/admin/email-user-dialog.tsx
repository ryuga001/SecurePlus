"use client";

import { Loader2 } from "lucide-react";
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
      setError("Enter a valid email address.");
      return;
    }
    if (!input.first_name || !input.last_name) {
      setError("Enter both a first and a last name.");
      return;
    }

    setError("");
    setEmailInvalid(false);

    try {
      if (user) {
        await updateUser({ id: user.id, ...input }).unwrap();
        toast.success("User updated");
      } else {
        await createUser(input).unwrap();
        toast.success("User added");
      }

      onClose();
    } catch (caught) {
      setEmailInvalid(apiErrorStatus(caught) === 409 || apiErrorStatus(caught) === 400);
      setError(apiErrorMessage(caught, "Could not save this user"));
    }
  }

  return (
    <>
      <DialogTitle>{user ? "Edit user" : "Add user"}</DialogTitle>
      <DialogDescription>
        People whose mail this workspace protects. Email addresses are unique across the platform.
      </DialogDescription>

      <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-4">
        <FormError message={error} />

        <Field id="user-email" label="Email">
          <Input
            id="user-email"
            type="email"
            required
            autoFocus
            spellCheck={false}
            placeholder="alice@example.com"
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
          <Field id="user-first-name" label="First name">
            <Input
              id="user-first-name"
              required
              value={firstName}
              disabled={pending}
              onChange={(event) => setFirstName(event.target.value)}
            />
          </Field>

          <Field id="user-last-name" label="Last name">
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
            Cancel
          </Button>
          <Button type="submit" disabled={pending}>
            {pending ? <Loader2 className="animate-spin" /> : null}
            {user ? "Save changes" : "Add user"}
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

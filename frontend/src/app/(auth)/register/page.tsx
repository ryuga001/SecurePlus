"use client";

import { ArrowRight, CheckCircle2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";

import {
  AuthHeading,
  Field,
  FormError,
  OtpInput,
  PasswordInput,
} from "@/components/auth/auth-form";
import { useAuth } from "@/components/auth-provider";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ApiError, api } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const { setIdentity } = useAuth();

  const [email, setEmail] = useState("");
  const [verifiedEmail, setVerifiedEmail] = useState("");
  const [token, setToken] = useState("");

  const [codeOpen, setCodeOpen] = useState(false);
  const [code, setCode] = useState("");
  const [codeError, setCodeError] = useState("");
  const [verifying, setVerifying] = useState(false);
  const [sending, setSending] = useState(false);

  const [details, setDetails] = useState({
    org_name: "",
    first_name: "",
    last_name: "",
    password: "",
    confirm: "",
  });
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const verified = Boolean(token);
  const locked = !verified;

  function update(field: keyof typeof details, value: string) {
    setDetails((current) => ({ ...current, [field]: value }));
  }

  function message(err: unknown) {
    return err instanceof ApiError ? err.message : "Something went wrong";
  }

  async function sendCode() {
    if (!email) return;

    setError("");
    setSending(true);

    try {
      await api.startRegistration(email);
      setCode("");
      setCodeError("");
      setCodeOpen(true);
      toast.success(`Verification code sent to ${email}`);
    } catch (err) {
      setError(message(err));
    } finally {
      setSending(false);
    }
  }

  async function submitCode(event: React.FormEvent) {
    event.preventDefault();
    setCodeError("");
    setVerifying(true);

    try {
      const result = await api.verifyEmail(email, code.trim());
      setToken(result.registration_token);
      setVerifiedEmail(email);
      setCodeOpen(false);
      toast.success("Email verified");
    } catch (err) {
      setCodeError(message(err));
    } finally {
      setVerifying(false);
    }
  }

  async function resend() {
    setCodeError("");

    try {
      await api.resendCode(email);
      toast.success("New code sent");
    } catch (err) {
      setCodeError(message(err));
    }
  }

  function changeEmail() {
    setToken("");
    setVerifiedEmail("");
    setCode("");
  }

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError("");

    if (!verified) {
      setError("Verify your email before creating the account");
      return;
    }

    if (details.password !== details.confirm) {
      setError("Passwords do not match");
      return;
    }

    setPending(true);

    try {
      const identity = await api.completeRegistration({
        registration_token: token,
        org_name: details.org_name,
        first_name: details.first_name,
        last_name: details.last_name,
        password: details.password,
      });

      setIdentity(identity);
      toast.success("Account created");
      router.push("/dashboard");
    } catch (err) {
      setError(message(err));
    } finally {
      setPending(false);
    }
  }

  return (
    <div>
      <AuthHeading
        title="Create your account"
        description="Verify your work email, then set up your organisation."
      />

      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field
          id="email"
          label="Work email"
          action={
            verified ? (
              <button
                type="button"
                onClick={changeEmail}
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                Change
              </button>
            ) : null
          }
          hint={
            verified
              ? undefined
              : "We will send a 6-digit code to confirm this address."
          }
        >
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Input
                id="email"
                type="email"
                required
                autoFocus
                autoComplete="email"
                placeholder="you@company.com"
                value={verified ? verifiedEmail : email}
                disabled={verified}
                onChange={(e) => setEmail(e.target.value)}
                className={verified ? "pr-10 disabled:opacity-100" : undefined}
              />
              {verified ? (
                <CheckCircle2 className="absolute top-1/2 right-3 size-4.5 -translate-y-1/2 text-success-text" />
              ) : null}
            </div>

            {verified ? null : (
              <Button
                type="button"
                variant="secondary"
                size="lg"
                className="shrink-0"
                onClick={sendCode}
                disabled={!email || sending}
              >
                {sending ? "Sending..." : "Verify"}
              </Button>
            )}
          </div>
        </Field>

        <fieldset disabled={locked} className="flex flex-col gap-4 disabled:opacity-55">
          <Field id="org_name" label="Organisation name">
            <Input
              id="org_name"
              required
              placeholder="Acme Technologies Pvt Ltd"
              value={details.org_name}
              onChange={(e) => update("org_name", e.target.value)}
            />
          </Field>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field id="first_name" label="First name">
              <Input
                id="first_name"
                required
                autoComplete="given-name"
                value={details.first_name}
                onChange={(e) => update("first_name", e.target.value)}
              />
            </Field>

            <Field id="last_name" label="Last name">
              <Input
                id="last_name"
                required
                autoComplete="family-name"
                value={details.last_name}
                onChange={(e) => update("last_name", e.target.value)}
              />
            </Field>
          </div>

          <Field id="password" label="Password" hint="Use at least 10 characters.">
            <PasswordInput
              id="password"
              required
              minLength={10}
              autoComplete="new-password"
              value={details.password}
              onChange={(e) => update("password", e.target.value)}
            />
          </Field>

          <Field id="confirm" label="Confirm password">
            <PasswordInput
              id="confirm"
              required
              autoComplete="new-password"
              value={details.confirm}
              onChange={(e) => update("confirm", e.target.value)}
            />
          </Field>
        </fieldset>

        <FormError message={error} />

        <Button type="submit" size="lg" className="w-full" disabled={locked || pending}>
          {pending ? "Creating account..." : "Create account"}
          {pending ? null : <ArrowRight />}
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-muted-foreground">
        Already have an account?{" "}
        <Link href="/login" className="font-medium text-primary hover:underline">
          Sign in
        </Link>
      </p>

      <Dialog open={codeOpen} onOpenChange={setCodeOpen}>
        <DialogContent>
          <DialogTitle>Check your inbox</DialogTitle>
          <DialogDescription>
            Enter the 6-digit code we sent to{" "}
            <span className="font-medium text-foreground">{email}</span>
          </DialogDescription>

          <form onSubmit={submitCode} className="mt-6 flex flex-col gap-4">
            <OtpInput value={code} onChange={setCode} autoFocus disabled={verifying} />

            <FormError message={codeError} />

            <Button
              type="submit"
              size="lg"
              className="w-full"
              disabled={code.length < 6 || verifying}
            >
              {verifying ? "Verifying..." : "Verify email"}
            </Button>

            <div className="flex items-center justify-between text-sm">
              <button
                type="button"
                onClick={resend}
                className="font-medium text-primary hover:underline"
              >
                Resend code
              </button>
              <button
                type="button"
                onClick={() => setCodeOpen(false)}
                className="text-muted-foreground hover:text-foreground"
              >
                Cancel
              </button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";

import { useAuth } from "@/components/auth-provider";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError, api } from "@/lib/api";

type Step = "email" | "code" | "details";

export default function RegisterPage() {
  const router = useRouter();
  const { setIdentity } = useAuth();

  const [step, setStep] = useState<Step>("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [token, setToken] = useState("");
  const [details, setDetails] = useState({
    org_name: "",
    first_name: "",
    last_name: "",
    password: "",
    confirm: "",
  });
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  function update(field: keyof typeof details, value: string) {
    setDetails((current) => ({ ...current, [field]: value }));
  }

  function fail(err: unknown) {
    setError(err instanceof ApiError ? err.message : "Something went wrong");
  }

  async function submitEmail(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    setPending(true);

    try {
      await api.startRegistration(email);
      toast.success("Verification code sent");
      setStep("code");
    } catch (err) {
      fail(err);
    } finally {
      setPending(false);
    }
  }

  async function submitCode(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    setPending(true);

    try {
      const verified = await api.verifyEmail(email, code.trim());
      setToken(verified.registration_token);
      toast.success("Email verified");
      setStep("details");
    } catch (err) {
      fail(err);
    } finally {
      setPending(false);
    }
  }

  async function submitDetails(event: React.FormEvent) {
    event.preventDefault();
    setError("");

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
      fail(err);
    } finally {
      setPending(false);
    }
  }

  async function resend() {
    setError("");

    try {
      await api.resendCode(email);
      toast.success("New code sent");
    } catch (err) {
      fail(err);
    }
  }

  if (step === "email") {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Verify your email</CardTitle>
          <CardDescription>Step 1 of 3 — we will send you a code</CardDescription>
        </CardHeader>

        <form onSubmit={submitEmail}>
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                required
                autoFocus
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>

            {error ? <p className="text-sm text-destructive">{error}</p> : null}
          </CardContent>

          <CardFooter className="mt-4 flex flex-col gap-3">
            <Button type="submit" className="w-full" disabled={pending}>
              {pending ? "Sending code..." : "Send verification code"}
            </Button>

            <p className="text-sm text-muted-foreground">
              Already have an account?{" "}
              <Link href="/login" className="text-foreground hover:underline">
                Sign in
              </Link>
            </p>
          </CardFooter>
        </form>
      </Card>
    );
  }

  if (step === "code") {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Enter your code</CardTitle>
          <CardDescription>Step 2 of 3 — sent to {email}</CardDescription>
        </CardHeader>

        <form onSubmit={submitCode}>
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="code">Verification code</Label>
              <Input
                id="code"
                required
                autoFocus
                inputMode="numeric"
                maxLength={6}
                autoComplete="one-time-code"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                className="font-mono tracking-[0.4em]"
              />
            </div>

            {error ? <p className="text-sm text-destructive">{error}</p> : null}
          </CardContent>

          <CardFooter className="mt-4 flex flex-col gap-3">
            <Button type="submit" className="w-full" disabled={pending}>
              {pending ? "Verifying..." : "Verify email"}
            </Button>

            <div className="flex w-full justify-between text-sm text-muted-foreground">
              <button type="button" onClick={resend} className="hover:text-foreground">
                Resend code
              </button>
              <button
                type="button"
                onClick={() => setStep("email")}
                className="hover:text-foreground"
              >
                Change email
              </button>
            </div>
          </CardFooter>
        </form>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Finish setting up</CardTitle>
        <CardDescription>Step 3 of 3 — {email} verified</CardDescription>
      </CardHeader>

      <form onSubmit={submitDetails}>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="org_name">Organization name</Label>
            <Input
              id="org_name"
              required
              autoFocus
              value={details.org_name}
              onChange={(e) => update("org_name", e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-2">
              <Label htmlFor="first_name">First name</Label>
              <Input
                id="first_name"
                required
                value={details.first_name}
                onChange={(e) => update("first_name", e.target.value)}
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="last_name">Last name</Label>
              <Input
                id="last_name"
                required
                value={details.last_name}
                onChange={(e) => update("last_name", e.target.value)}
              />
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              required
              minLength={10}
              autoComplete="new-password"
              value={details.password}
              onChange={(e) => update("password", e.target.value)}
            />
            <p className="text-xs text-muted-foreground">At least 10 characters</p>
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="confirm">Confirm password</Label>
            <Input
              id="confirm"
              type="password"
              required
              autoComplete="new-password"
              value={details.confirm}
              onChange={(e) => update("confirm", e.target.value)}
            />
          </div>

          {error ? <p className="text-sm text-destructive">{error}</p> : null}
        </CardContent>

        <CardFooter className="mt-4">
          <Button type="submit" className="w-full" disabled={pending}>
            {pending ? "Creating account..." : "Create account"}
          </Button>
        </CardFooter>
      </form>
    </Card>
  );
}

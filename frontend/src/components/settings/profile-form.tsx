"use client";

import { Building2, CircleAlert, Loader2, Lock } from "lucide-react";
import { useTranslations } from "next-intl";
import * as React from "react";
import { toast } from "sonner";

import { useAuth } from "@/components/auth-provider";
import { FormError } from "@/components/auth/auth-form";
import { LoadingRegion, SettingsCardSkeleton } from "@/components/loading/skeletons";
import {
  SettingsActionBar,
  SettingsRow,
  useUnsavedChangesWarning,
} from "@/components/settings/settings-layout";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { usePrivileges } from "@/hooks/use-privileges";
import { apiErrorCode, apiErrorMessage } from "@/lib/api-error";
import { ORGANIZATION_PRIVILEGES } from "@/lib/privileges";
import {
  useGetProfileQuery,
  useUpdateProfileMutation,
  type Profile,
  type ProfileUpdate,
} from "@/store/api/profile-api";

const ORG_NAME_MIN = 2;
const ORG_NAME_MAX = 100;
const PERSON_NAME_MAX = 50;

const FIELDS = ["org_name", "first_name", "last_name"] as const;

type Field = (typeof FIELDS)[number];
type Values = Record<Field, string>;
type Errors = Partial<Record<Field, string>>;
type Translate = ReturnType<typeof useTranslations<"settings.profile">>;

function normalize(value: string) {
  return value.trim().split(/\s+/).filter(Boolean).join(" ");
}

function length(value: string) {
  return Array.from(value).length;
}

function valuesOf(profile: Profile): Values {
  return {
    org_name: profile.org_name,
    first_name: profile.first_name,
    last_name: profile.last_name,
  };
}

function changesOf(values: Values, profile: Profile): ProfileUpdate {
  const changes: ProfileUpdate = {};

  for (const field of FIELDS) {
    const next = normalize(values[field]);
    if (next !== profile[field]) changes[field] = next;
  }

  return changes;
}

function validate(values: Values, t: Translate): Errors {
  const errors: Errors = {};

  const org = length(normalize(values.org_name));
  if (org < ORG_NAME_MIN || org > ORG_NAME_MAX) {
    errors.org_name = t("errors.orgNameLength", { min: ORG_NAME_MIN, max: ORG_NAME_MAX });
  }

  for (const field of ["first_name", "last_name"] as const) {
    const size = length(normalize(values[field]));

    if (size === 0) errors[field] = t(field === "first_name" ? "errors.firstNameRequired" : "errors.lastNameRequired");
    else if (size > PERSON_NAME_MAX) errors[field] = t("errors.nameTooLong", { max: PERSON_NAME_MAX });
  }

  return errors;
}

function initials(profile: Profile) {
  const letters = [profile.first_name, profile.last_name]
    .map((part) => Array.from(part.trim())[0] ?? "")
    .join("");

  return letters.toLocaleUpperCase() || "?";
}

export function ProfileForm() {
  const t = useTranslations("settings.profile");
  const table = useTranslations("table");

  const { data, isLoading, isError, error, refetch } = useGetProfileQuery();
  const { has, loading: privilegesLoading } = usePrivileges();

  if (isLoading || privilegesLoading) return <ProfileSkeleton />;

  if (isError || !data) {
    return (
      <div className="flex min-h-64 flex-col items-center justify-center gap-3 border bg-surface p-8 text-center text-sm">
        <CircleAlert className="size-5 text-error" />
        <p className="max-w-md text-muted-foreground">{apiErrorMessage(error, t("loadFailed"))}</p>
        <Button type="button" variant="outline" onClick={() => refetch()}>
          {table("tryAgain")}
        </Button>
      </div>
    );
  }

  return (
    <ProfileFields
      key={JSON.stringify(data)}
      profile={data}
      canEditOrganization={has(ORGANIZATION_PRIVILEGES.edit)}
    />
  );
}

function ProfileFields({
  profile,
  canEditOrganization,
}: {
  profile: Profile;
  canEditOrganization: boolean;
}) {
  const t = useTranslations("settings.profile");
  const common = useTranslations("common");
  const { identity, reload } = useAuth();

  const [updateProfile, { isLoading: saving }] = useUpdateProfileMutation();

  const [values, setValues] = React.useState<Values>(() => valuesOf(profile));
  const [touched, setTouched] = React.useState<Partial<Record<Field, boolean>>>({});
  const [submitted, setSubmitted] = React.useState(false);
  const [serverErrors, setServerErrors] = React.useState<Errors>({});
  const [formError, setFormError] = React.useState<string>();

  const changes = changesOf(values, profile);
  const dirty = Object.keys(changes).length > 0;
  const errors = validate(values, t);

  useUnsavedChangesWarning(dirty);

  function errorFor(field: Field) {
    return serverErrors[field] ?? (touched[field] || submitted ? errors[field] : undefined);
  }

  function describedBy(field: Field) {
    return errorFor(field) ? `${field}-error` : `${field}-hint`;
  }

  function change(field: Field, value: string) {
    setValues((current) => ({ ...current, [field]: value }));
    setServerErrors((current) => ({ ...current, [field]: undefined }));
  }

  function blur(field: Field) {
    setTouched((current) => ({ ...current, [field]: true }));
    setValues((current) => ({ ...current, [field]: normalize(current[field]) }));
  }

  function discard() {
    setValues(valuesOf(profile));
    setTouched({});
    setSubmitted(false);
    setServerErrors({});
    setFormError(undefined);
  }

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    setSubmitted(true);
    setFormError(undefined);

    const invalid = FIELDS.find((field) => errors[field] && (field !== "org_name" || canEditOrganization));
    if (invalid) {
      document.getElementById(invalid)?.focus();
      return;
    }

    if (!dirty) return;

    try {
      await updateProfile(changes).unwrap();
      await reload();
      toast.success(t("updated"));
    } catch (cause) {
      switch (apiErrorCode(cause)) {
        case "org_name_taken":
          setServerErrors({ org_name: t("errors.orgNameTaken") });
          document.getElementById("org_name")?.focus();
          break;
        case "forbidden":
          setFormError(t("errors.forbidden"));
          break;
        default:
          setFormError(apiErrorMessage(cause, t("saveFailed")));
      }
    }
  }

  return (
    <form noValidate onSubmit={onSubmit} className="flex flex-col gap-6">
      <FormError message={formError} />

      <Card>
        <CardContent className="flex-row flex-wrap items-center gap-4 pt-(--card-py)">
          <span
            aria-hidden
            className="flex size-12 shrink-0 items-center justify-center rounded-full bg-primary-container text-base font-semibold text-on-primary-container"
          >
            {initials(profile)}
          </span>

          <div className="min-w-0 flex-1">
            <p className="truncate text-base font-semibold">
              {profile.first_name} {profile.last_name}
            </p>
            <p className="truncate text-sm text-muted-foreground">{profile.admin_email}</p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {identity?.user.role ? <Badge variant="secondary">{identity.user.role}</Badge> : null}
            <Badge variant="outline" className="max-w-64">
              <Building2 />
              <span className="truncate">{profile.org_name}</span>
            </Badge>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("organizationTitle")}</CardTitle>
          <CardDescription>{t("organizationDescription")}</CardDescription>
        </CardHeader>

        <CardContent className="gap-6">
          <SettingsRow
            id="org_name"
            label={t("orgName")}
            hint={t("orgNameHint", { min: ORG_NAME_MIN, max: ORG_NAME_MAX })}
            error={canEditOrganization ? errorFor("org_name") : undefined}
          >
            <Input
              id="org_name"
              name="org_name"
              autoComplete="organization"
              maxLength={ORG_NAME_MAX}
              value={values.org_name}
              disabled={!canEditOrganization || saving}
              aria-invalid={canEditOrganization && errorFor("org_name") ? true : undefined}
              aria-describedby={describedBy("org_name")}
              onChange={(event) => change("org_name", event.target.value)}
              onBlur={() => blur("org_name")}
            />
            {!canEditOrganization ? (
              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Lock className="size-3.5 shrink-0" />
                {t("orgNameLocked")}
              </p>
            ) : null}
          </SettingsRow>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("administratorTitle")}</CardTitle>
          <CardDescription>{t("administratorDescription")}</CardDescription>
        </CardHeader>

        <CardContent className="gap-6">
          <SettingsRow id="first_name" label={t("firstName")} hint={t("nameHint", { max: PERSON_NAME_MAX })} error={errorFor("first_name")}>
            <Input
              id="first_name"
              name="first_name"
              autoComplete="given-name"
              maxLength={PERSON_NAME_MAX}
              value={values.first_name}
              disabled={saving}
              aria-invalid={errorFor("first_name") ? true : undefined}
              aria-describedby={describedBy("first_name")}
              onChange={(event) => change("first_name", event.target.value)}
              onBlur={() => blur("first_name")}
            />
          </SettingsRow>

          <SettingsRow id="last_name" label={t("lastName")} hint={t("nameHint", { max: PERSON_NAME_MAX })} error={errorFor("last_name")}>
            <Input
              id="last_name"
              name="last_name"
              autoComplete="family-name"
              maxLength={PERSON_NAME_MAX}
              value={values.last_name}
              disabled={saving}
              aria-invalid={errorFor("last_name") ? true : undefined}
              aria-describedby={describedBy("last_name")}
              onChange={(event) => change("last_name", event.target.value)}
              onBlur={() => blur("last_name")}
            />
          </SettingsRow>

          <SettingsRow id="admin_email" label={t("adminEmail")} hint={t("adminEmailHint")}>
            <div className="relative">
              <Input
                id="admin_email"
                type="email"
                readOnly
                value={profile.admin_email}
                aria-describedby="admin_email-hint"
                className="bg-muted pr-10 text-muted-foreground"
              />
              <Lock
                aria-hidden
                className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground"
              />
            </div>
          </SettingsRow>
        </CardContent>
      </Card>

      <SettingsActionBar dirty={dirty} unsavedLabel={t("unsavedChanges")}>
        <Button type="button" variant="ghost" disabled={!dirty || saving} onClick={discard}>
          {t("discard")}
        </Button>
        <Button type="submit" disabled={!dirty || saving}>
          {saving ? <Loader2 className="size-4 animate-spin" /> : null}
          {saving ? common("saving") : t("saveChanges")}
        </Button>
      </SettingsActionBar>
    </form>
  );
}

function ProfileSkeleton() {
  const t = useTranslations("settings.profile");

  return (
    <LoadingRegion label={t("loading")} className="flex flex-col gap-6">
      <div className="flex items-center gap-4 rounded-lg border bg-surface p-6">
        <Skeleton className="size-12 shrink-0 rounded-full" />
        <div className="flex flex-1 flex-col gap-2">
          <Skeleton className="h-4.5 w-44" />
          <Skeleton className="h-3.5 w-56" />
        </div>
        <Skeleton className="h-6 w-28" />
      </div>
      <SettingsCardSkeleton rows={1} />
      <SettingsCardSkeleton rows={3} />
    </LoadingRegion>
  );
}

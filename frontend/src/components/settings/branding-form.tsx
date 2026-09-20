"use client";

import {
  BadgeCheck,
  ImageOff,
  Loader2,
  Search,
  Trash2,
  Upload,
} from "lucide-react";
import { useTranslations } from "next-intl";
import Image from "next/image";
import * as React from "react";
import { toast } from "sonner";

import { useAuth } from "@/components/auth-provider";
import { FormError } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { apiErrorMessage } from "@/lib/api-error";
import type { Branding, Language, Theme } from "@/lib/api";
import { timezoneOptions } from "@/lib/datetime";
import { cn } from "@/lib/utils";
import {
  useRemoveLogoMutation,
  useUpdateBrandingMutation,
  useUploadLogoMutation,
} from "@/store/api/branding-api";

const MAX_LOGO_BYTES = 1024 * 1024;

const ACCEPTED_LOGO_TYPES = ["image/png", "image/jpeg", "image/webp"];

const LANGUAGES: { value: Language; label: string }[] = [
  { value: "ENGLISH", label: "English" },
  { value: "JAPANESE", label: "日本語" },
  { value: "SPANISH", label: "Español" },
];

const DEFAULT_THEME: Theme = "LIGHT";
const DEFAULT_LANGUAGE: Language = "ENGLISH";
const DEFAULT_TIMEZONE = "UTC";

export function BrandingForm() {
  const { identity, loading, reload } = useAuth();
  const t = useTranslations("settings.branding");
  const branding = identity?.branding;

  if (loading) return <BrandingSkeleton />;

  if (!branding) {
    return (
      <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
        {t("unavailable")}
      </div>
    );
  }

  const version = [branding.theme, branding.language, branding.timezone, branding.logo_url].join("|");

  return <BrandingFields key={version} branding={branding} reload={reload} />;
}

function BrandingFields({
  branding,
  reload,
}: {
  branding: Branding;
  reload: () => Promise<void>;
}) {
  const t = useTranslations("settings.branding");
  const common = useTranslations("common");

  const [updateBranding, { isLoading: saving }] = useUpdateBrandingMutation();
  const [uploadLogo, { isLoading: uploading }] = useUploadLogoMutation();
  const [removeLogo, { isLoading: removing }] = useRemoveLogoMutation();

  const [theme, setTheme] = React.useState<Theme>(branding.theme);
  const [language, setLanguage] = React.useState<Language>(branding.language);
  const [timezone, setTimezone] = React.useState(branding.timezone);
  const [error, setError] = React.useState<string>();

  const fileInput = React.useRef<HTMLInputElement>(null);

  const busy = saving || uploading || removing;

  const dirty =
    theme !== branding.theme ||
    language !== branding.language ||
    timezone !== branding.timezone;

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(undefined);

    try {
      await updateBranding({ theme, language, timezone }).unwrap();
      await reload();
      toast.success(t("updated"));
    } catch (cause) {
      setError(apiErrorMessage(cause, t("saveFailed")));
    }
  }

  function onResetDefaults() {
    setError(undefined);
    setTheme(DEFAULT_THEME);
    setLanguage(DEFAULT_LANGUAGE);
    setTimezone(DEFAULT_TIMEZONE);
  }

  async function onLogoSelected(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";

    if (!file) return;

    setError(undefined);

    if (file.size > MAX_LOGO_BYTES || !ACCEPTED_LOGO_TYPES.includes(file.type)) {
      setError(t("invalidLogo"));
      return;
    }

    try {
      await uploadLogo(file).unwrap();
      await reload();
      toast.success(t("logoUpdated"));
    } catch (cause) {
      setError(apiErrorMessage(cause, t("uploadFailed")));
    }
  }

  async function onLogoRemoved() {
    setError(undefined);

    try {
      await removeLogo().unwrap();
      await reload();
      toast.success(t("logoRemoved"));
    } catch (cause) {
      setError(apiErrorMessage(cause, t("uploadFailed")));
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-6">
      <FormError message={error} />

      <Card>
        <CardHeader>
          <CardTitle>{t("logoTitle")}</CardTitle>
          <CardDescription>{t("logoDescription")}</CardDescription>
          <CardAction>
            <span className="inline-flex items-center gap-1.5 border px-2 py-0.5 text-xs text-muted-foreground">
              <BadgeCheck className="size-3.5" />
              {t("publicBadge")}
            </span>
          </CardAction>
        </CardHeader>

        <CardContent className="flex flex-wrap items-center gap-5">
          <LogoPreview branding={branding} />

          <div className="flex min-w-64 flex-col items-start gap-2">
            <div className="flex items-center gap-2">
              <input
                ref={fileInput}
                id="logo"
                type="file"
                className="sr-only"
                accept={ACCEPTED_LOGO_TYPES.join(",")}
                disabled={busy}
                onChange={onLogoSelected}
              />
              <Button
                type="button"
                variant="outline"
                disabled={busy}
                onClick={() => fileInput.current?.click()}
              >
                {uploading ? <Loader2 className="size-4 animate-spin" /> : <Upload className="size-4" />}
                {uploading ? t("uploading") : t("uploadLogo")}
              </Button>
              {branding.logo_url ? (
                <Button
                  type="button"
                  variant="outline"
                  className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                  disabled={busy}
                  onClick={onLogoRemoved}
                >
                  {removing ? <Loader2 className="size-4 animate-spin" /> : <Trash2 className="size-4" />}
                  {common("remove")}
                </Button>
              ) : null}
            </div>
            <p className="text-xs text-muted-foreground">{t("logoFormats")}</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("themeTitle")}</CardTitle>
          <CardDescription>{t("themeDescription")}</CardDescription>
        </CardHeader>

        <CardContent>
          <fieldset className="grid gap-3 sm:grid-cols-2">
            {(
              [
                {
                  value: "LIGHT",
                  label: t("themeLight"),
                  hint: t("themeLightHint"),
                  dark: false,
                },
                {
                  value: "DARK",
                  label: t("themeDark"),
                  hint: t("themeDarkHint"),
                  dark: true,
                },
              ] as { value: Theme; label: string; hint: string; dark: boolean }[]
            ).map((option) => {
              const selected = theme === option.value;

              return (
                <label
                  key={option.value}
                  className={cn(
                    "flex cursor-pointer flex-col border transition-colors",
                    selected
                      ? "border-primary ring-1 ring-primary"
                      : "hover:bg-muted/60"
                  )}
                >
                  <input
                    type="radio"
                    name="theme"
                    className="sr-only"
                    value={option.value}
                    checked={selected}
                    disabled={busy}
                    onChange={() => setTheme(option.value)}
                  />
                  <ThemePreview dark={option.dark} selected={selected} />
                  <span className="flex items-center justify-between border-t px-4 py-3">
                    <span className="flex flex-col">
                      <span className="text-sm font-medium">{option.label}</span>
                      <span className="text-xs text-muted-foreground">{option.hint}</span>
                    </span>
                    <span
                      aria-hidden
                      className={cn(
                        "flex size-4 items-center justify-center rounded-full border transition-colors",
                        selected ? "border-primary bg-primary" : "border-border"
                      )}
                    >
                      {selected ? <span className="size-1.5 rounded-full bg-primary-foreground" /> : null}
                    </span>
                  </span>
                </label>
              );
            })}
          </fieldset>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("localizationTitle")}</CardTitle>
          <CardDescription>{t("localizationDescription")}</CardDescription>
        </CardHeader>

        <CardContent className="flex flex-col gap-6">
          <LocalizationRow
            id="language"
            label={t("language")}
            hint={t("languageHint")}
          >
            <Select
              id="language"
              value={language}
              disabled={busy}
              options={LANGUAGES}
              onChange={(event) => setLanguage(event.target.value as Language)}
            />
          </LocalizationRow>

          <LocalizationRow
            id="timezone-search"
            label={t("timezone")}
            hint={t("timezoneHint")}
          >
            <TimezonePicker value={timezone} disabled={busy} onChange={setTimezone} />
          </LocalizationRow>
        </CardContent>
      </Card>

      <div
        data-slot="settings-action-bar"
        className="sticky bottom-0 -mx-6 -mb-6 flex flex-wrap items-center justify-between gap-3 border-t bg-card px-6 py-4"
      >
        <span
          aria-live="polite"
          className={cn(
            "flex items-center gap-2 text-sm",
            dirty ? "text-foreground" : "text-transparent"
          )}
        >
          <span className={cn("size-2 rounded-full", dirty ? "bg-amber-500" : "bg-transparent")} />
          {t("unsavedChanges")}
        </span>

        <span className="flex items-center gap-2">
          <Button type="button" variant="ghost" disabled={busy} onClick={onResetDefaults}>
            {t("resetDefaults")}
          </Button>
          <Button type="submit" disabled={!dirty || busy}>
            {saving ? <Loader2 className="size-4 animate-spin" /> : null}
            {saving ? common("saving") : t("saveChanges")}
          </Button>
        </span>
      </div>
    </form>
  );
}

function LocalizationRow({
  id,
  label,
  hint,
  children,
}: {
  id: string;
  label: string;
  hint: string;
  children: React.ReactNode;
}) {
  return (
    <div className="grid gap-2 md:grid-cols-[minmax(0,220px)_1fr] md:items-start md:gap-8">
      <div className="flex flex-col gap-1">
        <Label htmlFor={id}>{label}</Label>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </div>
      <div className="md:max-w-xl">{children}</div>
    </div>
  );
}

function ThemePreview({ dark, selected }: { dark: boolean; selected: boolean }) {
  return (
    <div
      aria-hidden
      className={cn(
        "h-28 w-full overflow-hidden",
        dark ? "bg-[#151922]" : "bg-[#f4f5f7]"
      )}
    >
      <div className="flex h-full gap-2 p-2.5">
        <div className={cn("flex w-16 shrink-0 flex-col gap-1.5 border p-1.5", dark ? "border-white/10 bg-[#0b0e14]" : "border-black/5 bg-white")}>
          <span className={cn("h-1.5 w-full", selected ? "bg-[#6366f1]" : dark ? "bg-white/40" : "bg-zinc-300")} />
          <span className={cn("h-1.5 w-4/5", dark ? "bg-white/25" : "bg-zinc-300")} />
          <span className={cn("h-1.5 w-4/5", dark ? "bg-white/25" : "bg-zinc-300")} />
          <span className={cn("h-1.5 w-4/5", dark ? "bg-white/25" : "bg-zinc-300")} />
        </div>
        <div className="flex flex-1 flex-col gap-1.5">
          <div className="flex items-center gap-1.5">
            <span className={cn("h-1.5 w-8 shrink-0 rounded-full", selected ? "bg-[#6366f1]" : dark ? "bg-white/40" : "bg-zinc-400")} />
            <span className={cn("h-1.5 w-10", dark ? "bg-white/20" : "bg-zinc-200")} />
            <span className="ml-auto flex gap-1">
              <span className={cn("size-1.5 rounded-full", dark ? "bg-white/25" : "bg-zinc-300")} />
              <span className={cn("size-1.5 rounded-full", dark ? "bg-white/25" : "bg-zinc-300")} />
            </span>
          </div>
          <div className={cn("flex gap-1.5 border p-1.5", dark ? "border-white/10 bg-[#0b0e14]" : "border-black/5 bg-white")}>
            <span className={cn("h-4 w-8 shrink-0", selected ? "bg-[#6366f1]" : dark ? "bg-white/30" : "bg-zinc-300")} />
            <span className="flex flex-1 flex-col gap-1">
              <span className={cn("h-1.5 w-3/4", dark ? "bg-white/30" : "bg-zinc-300")} />
              <span className={cn("h-1.5 w-1/2", dark ? "bg-white/15" : "bg-zinc-200")} />
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function LogoPreview({ branding }: { branding: Branding }) {
  const t = useTranslations("settings.branding");

  if (!branding.logo_url) {
    return (
      <div className="flex size-24 flex-col items-center justify-center gap-1 border border-dashed text-muted-foreground">
        <ImageOff className="size-5" />
        <span className="text-xs">{t("noLogo")}</span>
      </div>
    );
  }

  return (
    <div className="relative size-24 shrink-0 border bg-white p-2">
      <Image
        src={branding.logo_url}
        alt={t("logoAlt")}
        width={80}
        height={80}
        unoptimized
        className="size-20 object-contain"
      />
      <span className="absolute -right-1.5 -bottom-1.5 flex size-5 items-center justify-center bg-primary text-primary-foreground">
        <BadgeCheck className="size-3.5" />
      </span>
    </div>
  );
}

function TimezonePicker({
  value,
  disabled,
  onChange,
}: {
  value: string;
  disabled?: boolean;
  onChange: (value: string) => void;
}) {
  const t = useTranslations("settings.branding");
  const [search, setSearch] = React.useState("");
  const options = React.useMemo(() => timezoneOptions(), []);

  const term = search.trim().toLowerCase();
  const visible = term
    ? options.filter((option) => option.label.toLowerCase().includes(term) || option.value.toLowerCase().includes(term))
    : options;

  const selected = options.find((option) => option.value === value);
  const selectedLabel = selected?.label ?? value;

  return (
    <div className="flex flex-col gap-2">
      <div className="relative">
        <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          id="timezone-search"
          className="pl-9"
          placeholder={t("timezoneSearch")}
          value={search}
          disabled={disabled}
          onChange={(event) => setSearch(event.target.value)}
        />
      </div>

      <div className="border">
        {selected ? (
          <div className="flex items-center justify-between gap-3 border-b bg-muted/60 px-3 py-2">
            <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
              {t("currentSelection")}
            </span>
            <span className="min-w-0 truncate text-sm font-medium" title={selectedLabel}>
              {selectedLabel}
            </span>
          </div>
        ) : null}

        <div className="max-h-56 overflow-y-auto">
          {visible.length === 0 ? (
            <p className="px-3 py-6 text-center text-sm text-muted-foreground">{t("noMatches")}</p>
          ) : (
            visible.map((option) => (
              <label
                key={option.value}
                className={cn(
                  "flex cursor-pointer items-center gap-2.5 border-b px-3 py-2.5 text-sm last:border-b-0",
                  option.value === value ? "bg-primary/8" : "hover:bg-muted/60"
                )}
              >
                <input
                  type="radio"
                  name="timezone"
                  className="size-4 accent-[var(--primary)]"
                  value={option.value}
                  checked={option.value === value}
                  disabled={disabled}
                  onChange={() => onChange(option.value)}
                />
                <span className="min-w-0 flex-1 truncate">{option.label}</span>
              </label>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function BrandingSkeleton() {
  return (
    <div className="flex animate-pulse flex-col gap-6">
      <div className="h-36 w-full border bg-muted/50" />
      <div className="h-60 w-full border bg-muted/50" />
      <div className="h-48 w-full border bg-muted/50" />
    </div>
  );
}
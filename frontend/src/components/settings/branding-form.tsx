"use client";

import { ImageOff, Loader2, Search, Trash2, Upload } from "lucide-react";
import Image from "next/image";
import * as React from "react";
import { toast } from "sonner";

import { useAuth } from "@/components/auth-provider";
import { Field, FormError } from "@/components/auth/auth-form";
import { Button } from "@/components/ui/button";
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

const INVALID_LOGO_MESSAGE = "Logo must be PNG, JPEG, or WebP and 1 MB or less.";

const THEMES: { value: Theme; label: string; hint: string }[] = [
  { value: "LIGHT", label: "Light", hint: "Bright background" },
  { value: "DARK", label: "Dark", hint: "Dim background" },
];

const LANGUAGES: { value: Language; label: string }[] = [
  { value: "ENGLISH", label: "English" },
  { value: "JAPANESE", label: "日本語" },
  { value: "SPANISH", label: "Español" },
];

export function BrandingForm() {
  const { identity, loading, reload } = useAuth();
  const branding = identity?.branding;

  if (loading) return <BrandingSkeleton />;

  if (!branding) {
    return (
      <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
        Branding settings are not available for this workspace.
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
  const [updateBranding, { isLoading: saving }] = useUpdateBrandingMutation();
  const [uploadLogo, { isLoading: uploading }] = useUploadLogoMutation();
  const [removeLogo, { isLoading: removing }] = useRemoveLogoMutation();

  const [theme, setTheme] = React.useState<Theme>(branding.theme);
  const [language, setLanguage] = React.useState<Language>(branding.language);
  const [timezone, setTimezone] = React.useState(branding.timezone);
  const [error, setError] = React.useState<string>();

  const fileInput = React.useRef<HTMLInputElement>(null);

  const busy = saving || uploading || removing;

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(undefined);

    try {
      await updateBranding({ theme, language, timezone }).unwrap();
      await reload();
      toast.success("Branding updated");
    } catch (cause) {
      setError(apiErrorMessage(cause, "Unable to save branding settings."));
    }
  }

  async function onLogoSelected(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";

    if (!file) return;

    setError(undefined);

    if (file.size > MAX_LOGO_BYTES || !ACCEPTED_LOGO_TYPES.includes(file.type)) {
      setError(INVALID_LOGO_MESSAGE);
      return;
    }

    try {
      await uploadLogo(file).unwrap();
      await reload();
      toast.success("Logo updated");
    } catch (cause) {
      setError(apiErrorMessage(cause, "Unable to upload logo."));
    }
  }

  async function onLogoRemoved() {
    setError(undefined);

    try {
      await removeLogo().unwrap();
      await reload();
      toast.success("Logo removed");
    } catch (cause) {
      setError(apiErrorMessage(cause, "Unable to remove logo."));
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex max-w-2xl flex-col gap-6">
      <FormError message={error} />

      <Field id="logo" label="Logo" hint="PNG, JPEG or WebP, up to 1 MB.">
        <div className="flex flex-wrap items-center gap-4">
          <LogoPreview branding={branding} />

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
              {uploading ? "Uploading…" : "Upload logo"}
            </Button>
            {branding.logo_url ? (
              <Button type="button" variant="outline" disabled={busy} onClick={onLogoRemoved}>
                {removing ? <Loader2 className="size-4 animate-spin" /> : <Trash2 className="size-4" />}
                Remove
              </Button>
            ) : null}
          </div>
        </div>
      </Field>

      <fieldset className="flex flex-col gap-2" disabled={busy}>
        <Label>Theme</Label>
        <div className="flex flex-wrap gap-3">
          {THEMES.map((option) => (
            <label
              key={option.value}
              className={cn(
                "flex cursor-pointer items-start gap-2.5 border px-4 py-3 text-sm transition-colors",
                theme === option.value ? "border-primary bg-primary/8" : "hover:bg-muted/60"
              )}
            >
              <input
                type="radio"
                name="theme"
                className="mt-0.5 size-4 accent-[var(--primary)]"
                value={option.value}
                checked={theme === option.value}
                onChange={() => setTheme(option.value)}
              />
              <span className="flex flex-col">
                <span className="font-medium">{option.label}</span>
                <span className="text-xs text-muted-foreground">{option.hint}</span>
              </span>
            </label>
          ))}
        </div>
      </fieldset>

      <Field id="language" label="Language">
        <Select
          id="language"
          value={language}
          disabled={busy}
          options={LANGUAGES}
          onChange={(event) => setLanguage(event.target.value as Language)}
        />
      </Field>

      <TimezonePicker value={timezone} disabled={busy} onChange={setTimezone} />

      <div className="flex justify-end border-t pt-4">
        <Button type="submit" disabled={busy}>
          {saving ? <Loader2 className="size-4 animate-spin" /> : null}
          {saving ? "Saving…" : "Save"}
        </Button>
      </div>
    </form>
  );
}

function LogoPreview({ branding }: { branding: Branding }) {
  if (!branding.logo_url) {
    return (
      <div className="flex size-24 flex-col items-center justify-center gap-1 border border-dashed text-muted-foreground">
        <ImageOff className="size-5" />
        <span className="text-xs">No logo</span>
      </div>
    );
  }

  return (
    <Image
      src={branding.logo_url}
      alt="Customer logo"
      width={96}
      height={96}
      unoptimized
      className="size-24 border object-contain p-2"
    />
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
  const [search, setSearch] = React.useState("");
  const options = React.useMemo(() => timezoneOptions(), []);

  const term = search.trim().toLowerCase();
  const visible = term
    ? options.filter((option) => option.label.toLowerCase().includes(term) || option.value.toLowerCase().includes(term))
    : options;

  const selected = options.find((option) => option.value === value);

  return (
    <Field id="timezone-search" label="Timezone" hint={selected?.label ?? value}>
      <div className="flex flex-col gap-2">
        <div className="relative">
          <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="timezone-search"
            className="h-9 pl-9"
            placeholder="Search timezone…"
            value={search}
            disabled={disabled}
            onChange={(event) => setSearch(event.target.value)}
          />
        </div>

        <div className="max-h-56 overflow-y-auto border">
          {visible.length === 0 ? (
            <p className="px-3 py-6 text-center text-sm text-muted-foreground">No matches</p>
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
    </Field>
  );
}

function BrandingSkeleton() {
  return (
    <div className="flex max-w-2xl animate-pulse flex-col gap-6">
      <div className="h-24 w-24 bg-muted" />
      <div className="h-11 w-full bg-muted" />
      <div className="h-11 w-full bg-muted" />
      <div className="h-56 w-full bg-muted" />
    </div>
  );
}

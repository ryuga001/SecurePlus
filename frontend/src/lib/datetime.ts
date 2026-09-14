import type { Language } from "@/lib/api";

export const LOCALE_BY_LANGUAGE: Record<Language, string> = {
  ENGLISH: "en",
  JAPANESE: "ja",
  SPANISH: "es",
};

export type TimezoneOption = { label: string; value: string };

function supportedTimezones(): string[] {
  const values = Intl.supportedValuesOf?.("timeZone");
  if (values && values.length > 0) return [...values];

  return [Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"];
}

function offsetLabel(timezone: string, reference: Date): string {
  const parts = new Intl.DateTimeFormat("en", {
    timeZone: timezone,
    timeZoneName: "longOffset",
  }).formatToParts(reference);

  const name = parts.find((part) => part.type === "timeZoneName")?.value ?? "GMT";

  return name === "GMT" ? "UTC+00:00" : name.replace("GMT", "UTC");
}

function zoneName(timezone: string, reference: Date): string {
  const parts = new Intl.DateTimeFormat("en", {
    timeZone: timezone,
    timeZoneName: "long",
  }).formatToParts(reference);

  return parts.find((part) => part.type === "timeZoneName")?.value ?? timezone;
}

export function timezoneOptions(): TimezoneOption[] {
  const reference = new Date();

  return supportedTimezones()
    .map((timezone) => {
      const city = timezone.split("/").pop()?.replace(/_/g, " ") ?? timezone;

      return {
        value: timezone,
        label: `(${offsetLabel(timezone, reference)}) ${zoneName(timezone, reference)} — ${city}`,
      };
    })
    .sort((a, b) => a.label.localeCompare(b.label));
}

export function formatInZone(
  value: string | number | Date,
  timezone: string,
  language: Language = "ENGLISH",
  precision: "date" | "datetime" = "datetime"
): string {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return "";

  const options: Intl.DateTimeFormatOptions =
    precision === "date"
      ? { dateStyle: "medium" }
      : { dateStyle: "medium", timeStyle: "short" };

  const locale = LOCALE_BY_LANGUAGE[language] ?? "en";

  try {
    return new Intl.DateTimeFormat(locale, { ...options, timeZone: timezone }).format(date);
  } catch {
    return new Intl.DateTimeFormat(locale, { ...options, timeZone: "UTC" }).format(date);
  }
}

import type { Language } from "@/lib/api";
import { LOCALE_BY_LANGUAGE } from "@/lib/datetime";
import type { AnalyticsPeriod } from "@/store/api/email-analytics-api";

export const ANALYTICS_PERIODS: AnalyticsPeriod[] = ["day", "week", "month"];

const AXIS_STEPS = 4;

const NICE_STEPS = [1, 2, 5];

export function niceCeiling(value: number) {
  if (value <= AXIS_STEPS) return AXIS_STEPS;

  const rough = value / AXIS_STEPS;
  const magnitude = 10 ** Math.floor(Math.log10(rough));

  for (const step of NICE_STEPS) {
    if (step * magnitude >= rough) return step * magnitude * AXIS_STEPS;
  }

  return 10 * magnitude * AXIS_STEPS;
}

export function axisTicks(max: number) {
  const step = max / AXIS_STEPS;

  return Array.from({ length: AXIS_STEPS + 1 }, (_, index) => max - index * step);
}

export function formatCount(value: number, language: Language = "ENGLISH") {
  return value.toLocaleString(LOCALE_BY_LANGUAGE[language] ?? "en");
}

export function formatCompact(value: number, language: Language = "ENGLISH") {
  return value.toLocaleString(LOCALE_BY_LANGUAGE[language] ?? "en", {
    notation: "compact",
    maximumFractionDigits: 1,
  });
}

export function bucketLabel(
  value: string,
  bucket: AnalyticsPeriod,
  timezone: string,
  language: Language = "ENGLISH",
) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "";

  const locale = LOCALE_BY_LANGUAGE[language] ?? "en";
  const options: Intl.DateTimeFormatOptions =
    bucket === "month" ? { month: "short", year: "2-digit" } : { day: "numeric", month: "short" };

  try {
    return new Intl.DateTimeFormat(locale, { ...options, timeZone: timezone }).format(parsed);
  } catch {
    return new Intl.DateTimeFormat(locale, { ...options, timeZone: "UTC" }).format(parsed);
  }
}

export function relativeTime(value: string, language: Language = "ENGLISH") {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "";

  const locale = LOCALE_BY_LANGUAGE[language] ?? "en";
  const formatter = new Intl.RelativeTimeFormat(locale, { numeric: "auto", style: "narrow" });
  const seconds = Math.round((parsed.getTime() - Date.now()) / 1000);

  const divisions: [number, Intl.RelativeTimeFormatUnit][] = [
    [60, "second"],
    [3600, "minute"],
    [86400, "hour"],
    [604800, "day"],
    [2629800, "week"],
    [31557600, "month"],
    [Number.POSITIVE_INFINITY, "year"],
  ];

  let previous = 1;

  for (const [limit, unit] of divisions) {
    if (Math.abs(seconds) < limit) {
      return formatter.format(Math.trunc(seconds / previous), unit);
    }
    previous = limit;
  }

  return formatter.format(0, "second");
}

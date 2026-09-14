import type { Language } from "@/lib/api";

import en from "../../messages/en.json";
import es from "../../messages/es.json";
import ja from "../../messages/ja.json";

export type Locale = "en" | "ja" | "es";

export type Messages = typeof en;

export const LOCALE_BY_LANGUAGE: Record<Language, Locale> = {
  ENGLISH: "en",
  JAPANESE: "ja",
  SPANISH: "es",
};

export const DEFAULT_LOCALE: Locale = "en";

const MESSAGES: Record<Locale, Messages> = {
  en,
  ja: ja as Messages,
  es: es as Messages,
};

export function localeFor(language: Language | undefined): Locale {
  return language ? LOCALE_BY_LANGUAGE[language] ?? DEFAULT_LOCALE : DEFAULT_LOCALE;
}

export function messagesFor(locale: Locale): Messages {
  return MESSAGES[locale] ?? MESSAGES[DEFAULT_LOCALE];
}

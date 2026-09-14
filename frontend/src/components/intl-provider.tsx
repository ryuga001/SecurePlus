"use client";

import { NextIntlClientProvider } from "next-intl";
import { useEffect } from "react";

import { useAuth } from "@/components/auth-provider";
import { localeFor, messagesFor } from "@/i18n/config";

export function IntlProvider({ children }: { children: React.ReactNode }) {
  const { identity } = useAuth();
  const locale = localeFor(identity?.branding?.language);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  return (
    <NextIntlClientProvider
      locale={locale}
      messages={messagesFor(locale)}
      timeZone={identity?.branding?.timezone ?? "UTC"}
    >
      {children}
    </NextIntlClientProvider>
  );
}

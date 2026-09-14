"use client";

import { ThemeProvider as NextThemesProvider, useTheme } from "next-themes";
import { useEffect } from "react";

import { useAuth } from "@/components/auth-provider";
import { THEME_STORAGE_KEY } from "@/lib/theme";

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  return (
    <NextThemesProvider
      attribute="class"
      defaultTheme="light"
      enableSystem={false}
      storageKey={THEME_STORAGE_KEY}
      disableTransitionOnChange
    >
      <BrandingSync />
      {children}
    </NextThemesProvider>
  );
}

function BrandingSync() {
  const { identity } = useAuth();
  const { setTheme } = useTheme();

  const preference = identity?.branding?.theme;

  useEffect(() => {
    if (!preference) return;

    setTheme(preference === "DARK" ? "dark" : "light");
  }, [preference, setTheme]);

  return null;
}

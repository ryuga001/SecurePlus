import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";

import { AuthProvider } from "@/components/auth-provider";
import { IntlProvider } from "@/components/intl-provider";
import { StoreProvider } from "@/components/store-provider";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";

import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  display: "swap",
  preload: true,
  fallback: ["system-ui", "sans-serif"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
  display: "swap",
  preload: true,
  fallback: ["ui-monospace", "SFMono-Regular", "Menlo", "monospace"],
});

export const metadata: Metadata = {
  title: {
    default: "SecurePlus",
    template: "%s | SecurePlus",
  },
  description: "SecurePlus security dashboard",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${inter.variable} ${jetbrainsMono.variable} h-full`}
    >
      <body className="min-h-full bg-background font-sans antialiased text-foreground">
        <StoreProvider>
          <AuthProvider>
            <IntlProvider>
              <ThemeProvider>{children}</ThemeProvider>
            </IntlProvider>
          </AuthProvider>
        </StoreProvider>

        <Toaster position="top-right" />
      </body>
    </html>
  );
}
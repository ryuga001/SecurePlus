import type { EmailProvider } from "@/store/api/email-configurations-api";

export type ProviderMeta = {
  label: string;
  description: string;
  monogram: string;
};

export const providerMeta: Record<EmailProvider, ProviderMeta> = {
  outlook365: {
    label: "Outlook 365",
    description: "Microsoft 365 tenant mailboxes.",
    monogram: "O",
  },
  gmail: {
    label: "Gmail",
    description: "Google Workspace mailboxes.",
    monogram: "G",
  },
};

export const providerOrder: EmailProvider[] = ["outlook365", "gmail"];

export function providerLabel(provider: string) {
  return providerMeta[provider as EmailProvider]?.label ?? provider;
}

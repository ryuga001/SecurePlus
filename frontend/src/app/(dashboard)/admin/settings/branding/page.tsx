"use client";

import { useTranslations } from "next-intl";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { BrandingForm } from "@/components/settings/branding-form";

export default function BrandingSettingsPage() {
  const t = useTranslations("settings.branding");

  return (
    <Consolepage heading={t("heading")} subheading={t("subheading")} data={<BrandingForm />} />
  );
}

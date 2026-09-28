"use client";

import { useTranslations } from "next-intl";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { ProfileForm } from "@/components/settings/profile-form";

export default function ProfileSettingsPage() {
  const t = useTranslations("settings.profile");

  return <Consolepage heading={t("heading")} subheading={t("subheading")} data={<ProfileForm />} />;
}

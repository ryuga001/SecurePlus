"use client";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";
import { BrandingForm } from "@/components/settings/branding-form";

export default function BrandingSettingsPage() {
  return (
    <Consolepage
      heading="Branding"
      subheading="Logo, theme, language and timezone for everyone in this workspace."
      data={<BrandingForm />}
    />
  );
}

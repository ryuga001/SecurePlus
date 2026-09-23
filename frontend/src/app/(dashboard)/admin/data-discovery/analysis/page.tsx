"use client";

import { useTranslations } from "next-intl";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";

export default function DataDiscoveryAnalysisPage() {
  const t = useTranslations("data-discovery.analysis");

  return (
    <Consolepage
      heading={t("heading")}
      subheading={t("subheading")}
      data={
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          {t("placeholder")}
        </div>
      }
    />
  );
}

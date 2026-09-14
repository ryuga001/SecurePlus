"use client";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";

export default function ProfileSettingsPage() {
  return (
    <Consolepage
      heading="Profile"
      subheading="Your account details."
      data={
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          Profile settings are not available yet.
        </div>
      }
    />
  );
}

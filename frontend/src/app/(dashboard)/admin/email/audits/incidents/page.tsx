"use client";

import { ShieldAlert } from "lucide-react";

import Consolepage from "@/components/dashboard/pageThemes/consolepage";

export default function EmailIncidentsPage() {
  return (
    <Consolepage
      heading="Email audits"
      subheading="Policy violations raised against outbound mail."
      data={
        <div className="flex flex-col items-center gap-3 border bg-card px-4 py-20 text-center">
          <ShieldAlert className="size-7 text-muted-foreground" />
          <p className="text-sm font-medium">No incidents yet</p>
          <p className="max-w-md text-sm text-muted-foreground">
            Policy evaluation is not switched on for outbound mail, so no incidents are being
            raised. Delivery records are on the Delivery Audit tab.
          </p>
        </div>
      }
    />
  );
}

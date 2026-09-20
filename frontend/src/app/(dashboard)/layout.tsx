"use client";

import { useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { useAuth } from "@/components/auth-provider";
import Sidebar from "@/components/dashboard/sidebar/sidebar";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { identity, loading } = useAuth();
  const t = useTranslations("common");

  useEffect(() => {
    if (!loading && !identity) router.replace("/login");
  }, [loading, identity, router]);

  if (loading || !identity) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <p className="text-sm text-muted-foreground">{t("loading")}</p>
      </div>
    );
  }

  return (
    <div className="flex h-svh overflow-hidden">
      <Sidebar />

      <main className="min-w-0 flex-1 overflow-y-auto">
      <div className="mx-auto w-full max-w-[1440px] p-4 sm:p-6 xl:p-8">{children}</div>
    </main>
    </div>
  );
}

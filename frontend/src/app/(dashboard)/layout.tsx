"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { useAuth } from "@/components/auth-provider";
import Sidebar from "@/components/dashboard/sidebar/sidebar";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { identity, loading } = useAuth();

  useEffect(() => {
    if (!loading && !identity) router.replace("/login");
  }, [loading, identity, router]);

  if (loading || !identity) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <p className="text-sm text-muted-foreground">Loading...</p>
      </div>
    );
  }

  return (
    <div className="flex h-svh overflow-hidden">
      <Sidebar />

      <main className="min-w-0 flex-1 overflow-y-auto p-6">{children}</main>
    </div>
  );
}

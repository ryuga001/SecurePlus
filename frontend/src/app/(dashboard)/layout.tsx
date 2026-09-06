"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { useAuth } from "@/components/auth-provider";
import { Button } from "@/components/ui/button";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { identity, loading, signOut } = useAuth();

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

  async function onSignOut() {
    await signOut();
    router.replace("/login");
  }

  return (
    <div className="flex min-h-full flex-1 flex-col">
      <header className="flex items-center justify-between border-b px-6 py-4">
        <div>
          <p className="text-sm font-medium">{identity.customer.org_name}</p>
          <p className="text-xs text-muted-foreground">
            {identity.user.email}
            {identity.user.role ? ` · ${identity.user.role}` : ""}
          </p>
        </div>

        <Button variant="outline" size="sm" onClick={onSignOut}>
          Sign out
        </Button>
      </header>

      <main className="flex-1 p-6">{children}</main>
    </div>
  );
}

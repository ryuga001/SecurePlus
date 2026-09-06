"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { useAuth } from "@/components/auth-provider";

export default function Home() {
  const router = useRouter();
  const { identity, loading } = useAuth();

  useEffect(() => {
    if (loading) return;
    router.replace(identity ? "/dashboard" : "/login");
  }, [loading, identity, router]);

  return (
    <div className="flex flex-1 items-center justify-center">
      <p className="text-sm text-muted-foreground">Loading...</p>
    </div>
  );
}

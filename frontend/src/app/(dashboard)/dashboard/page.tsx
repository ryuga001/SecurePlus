"use client";

import { useAuth } from "@/components/auth-provider";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function DashboardPage() {
  const { identity } = useAuth();

  if (!identity) return null;

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">
          Welcome, {identity.user.first_name}
        </h1>
        <p className="text-sm text-muted-foreground">
          You are signed in to {identity.customer.org_name}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Organization</CardTitle>
            <CardDescription>Tenant details</CardDescription>
          </CardHeader>
          <CardContent className="text-sm">
            <p className="font-medium">{identity.customer.org_name}</p>
            <p className="text-muted-foreground">ID {identity.customer.id}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Account</CardTitle>
            <CardDescription>Your dashboard user</CardDescription>
          </CardHeader>
          <CardContent className="text-sm">
            <p className="font-medium">
              {identity.user.first_name} {identity.user.last_name}
            </p>
            <p className="text-muted-foreground">{identity.user.email}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Role</CardTitle>
            <CardDescription>Access level</CardDescription>
          </CardHeader>
          <CardContent className="text-sm">
            <p className="font-medium">{identity.user.role ?? "None"}</p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

"use client";

import { useTranslations } from "next-intl";

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
  const t = useTranslations("dashboard");
  const common = useTranslations("common");

  if (!identity) return null;

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="font-heading text-[28px] leading-9 font-semibold tracking-[-0.01em]">
          {t("welcome", { name: identity.user.first_name })}
        </h1>
        <p className="mt-1 text-sm leading-5 text-muted-foreground">
          {t("signedInTo", { org: identity.customer.org_name })}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("organization")}</CardTitle>
            <CardDescription>{t("tenantDetails")}</CardDescription>
          </CardHeader>
          <CardContent className="text-sm">
            <p className="font-medium">{identity.customer.org_name}</p>
            <p className="text-muted-foreground">{t("id", { id: identity.customer.id })}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("account")}</CardTitle>
            <CardDescription>{t("yourDashboardUser")}</CardDescription>
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
            <CardTitle className="text-base">{t("role")}</CardTitle>
            <CardDescription>{t("accessLevel")}</CardDescription>
          </CardHeader>
          <CardContent className="text-sm">
            <p className="font-medium">{identity.user.role ?? common("none")}</p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

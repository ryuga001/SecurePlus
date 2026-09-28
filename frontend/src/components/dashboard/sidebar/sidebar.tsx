"use client";

import { ChevronDown, LogOut, ShieldCheck } from "lucide-react";
import { useTranslations } from "next-intl";
import Image from "next/image";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";

import { useAuth } from "@/components/auth-provider";
import { Button } from "@/components/ui/button";
import { useRouteAccess } from "@/hooks/use-route-access";
import { isRouteVisible, routes, type RouteAccess, type RouteMap, type RouteNode } from "@/lib/routes";
import { cn } from "@/lib/utils";

function initials(firstName: string, lastName: string): string {
  const first = firstName.trim().charAt(0);
  const last = lastName.trim().charAt(0);

  return (first + last).toUpperCase() || "?";
}

function isActive(pathname: string, node: RouteNode): boolean {
  if (node.routePath && (pathname === node.routePath || pathname.startsWith(`${node.routePath}/`))) {
    return true;
  }

  return Object.values(node.children ?? {}).some((child) => isActive(pathname, child));
}

function isGroup(node: RouteNode): boolean {
  return !node.routePath && Object.keys(node.children ?? {}).length > 0;
}

function NavLink({
  node,
  pathname,
  nested,
}: {
  node: RouteNode;
  pathname: string;
  nested?: boolean;
}) {
  const active = isActive(pathname, node);
  const Icon = node.icon;
  const t = useTranslations("nav");

  return (
    <Link
      href={node.routePath ?? "#"}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex min-h-10 items-center gap-3 rounded-md px-3 text-sm font-medium transition-colors",
        nested && "min-h-9 pl-10 text-[0.8125rem] font-normal",
        active
          ? "bg-primary-container text-on-primary-container"
          : "text-text-tertiary hover:bg-surface-container-high hover:text-foreground"
      )}
    >
      {Icon ? (
        <Icon
          className={cn("size-[18px] shrink-0", active ? "text-primary" : "")}
          strokeWidth={1.75}
        />
      ) : null}
      <span className="truncate">{t(node.labelKey)}</span>
    </Link>
  );
}

function NavGroup({ node, pathname, access }: { node: RouteNode; pathname: string; access: RouteAccess }) {
  const active = isActive(pathname, node);
  const [open, setOpen] = useState(active || node.defaultOpen === true);
  const Icon = node.icon;
  const t = useTranslations("nav");

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-expanded={open}
        className={cn(
          "flex min-h-10 w-full items-center gap-3 rounded-md px-3 text-sm font-medium transition-colors",
          active
            ? "bg-primary-container text-on-primary-container"
            : "text-text-tertiary hover:bg-surface-container-high hover:text-foreground"
        )}
      >
        {Icon ? <Icon className="size-[18px] shrink-0" strokeWidth={1.75} /> : null}
        <span className="flex-1 truncate text-left">{t(node.labelKey)}</span>
        <ChevronDown
          className={cn("size-4 transition-transform", open && "rotate-180")}
          strokeWidth={1.75}
        />
      </button>

      {open ? (
        <div className="mt-1 flex flex-col">
          {Object.entries(node.children ?? {})
            .filter(([, child]) => isRouteVisible(child, access))
            .map(([key, child]) =>
            isGroup(child) ? (
              <NavGroup key={key} node={child} pathname={pathname} access={access} />
            ) : (
              <NavLink key={key} node={child} pathname={pathname} nested />
            )
          )}
        </div>
      ) : null}
    </div>
  );
}

const Sidebar = () => {
  const pathname = usePathname();
  const router = useRouter();
  const { identity, signOut } = useAuth();
  const t = useTranslations("common");
  const [failedLogo, setFailedLogo] = useState<string | null>(null);

  const orgName = identity?.customer.org_name ?? "SecurePlus";
  const brandingLogo = identity?.branding?.logo_url ?? null;
  const logoUrl = brandingLogo !== failedLogo ? brandingLogo : null;

  async function onSignOut() {
    await signOut();
    router.replace("/login");
  }

  const access = useRouteAccess();
  const items = Object.entries(routes as RouteMap).filter(([, node]) => isRouteVisible(node, access));

  return (
    <aside className="flex h-svh w-60 shrink-0 flex-col overflow-hidden border-r bg-sidebar">
      <Link
        href="/dashboard"
        className="flex h-16 shrink-0 items-center gap-3 border-b border-sidebar-border px-4"
      >
        {logoUrl ? (
          <Image
            key={logoUrl}
            src={logoUrl}
            alt={orgName}
            width={32}
            height={32}
            unoptimized
            className="size-8 shrink-0 object-contain"
            onError={() => setFailedLogo(brandingLogo)}
          />
        ) : (
          <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-primary-container">
            <ShieldCheck className="size-[18px] text-primary" strokeWidth={1.75} />
          </span>
        )}
        <span className="truncate text-base font-semibold tracking-tight">{orgName}</span>
      </Link>

      <nav className="scrollbar-thin flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto px-3 py-4">
        {items.map(([key, node]) =>
          isGroup(node) ? (
            <NavGroup key={key} node={node} pathname={pathname} access={access} />
          ) : (
            <NavLink key={key} node={node} pathname={pathname} />
          )
        )}
      </nav>

      <div className="shrink-0 border-t border-sidebar-border bg-sidebar-accent/40 p-3 pb-4">
        {identity ? (
          <div className="mb-3 flex items-center gap-2.5 px-2 pt-2">
            <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary-container text-xs font-semibold text-on-primary-container">
              {initials(identity.user.first_name, identity.user.last_name)}
            </span>
            <div className="min-w-0">
              <p className="truncate text-sm font-medium leading-5">
                {identity.user.first_name} {identity.user.last_name}
              </p>
              <p className="truncate text-xs leading-4 text-text-tertiary">{identity.user.email}</p>
            </div>
          </div>
        ) : null}

        <Button
          variant="outline"
          className="w-full justify-start text-text-secondary hover:border-error-border hover:bg-error-container hover:text-error-text"
          onClick={onSignOut}
        >
          <LogOut strokeWidth={1.75} />
          {t("signOut")}
        </Button>
      </div>
    </aside>
  );
};

export default Sidebar;
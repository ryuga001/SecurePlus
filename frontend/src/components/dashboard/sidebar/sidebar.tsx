"use client";

import { ChevronDown, LogOut, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";

import { useAuth } from "@/components/auth-provider";
import { Button } from "@/components/ui/button";
import { routes, type RouteMap, type RouteNode } from "@/lib/routes";
import { cn } from "@/lib/utils";

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

  return (
    <Link
      href={node.routePath ?? "#"}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex items-center gap-2.5 border-l-2 px-4 py-2.5 text-sm transition-colors",
        nested && "py-2 pl-11 text-[0.85rem]",
        active
          ? "border-l-primary bg-primary/8 font-medium text-primary"
          : "border-l-transparent text-muted-foreground hover:bg-muted hover:text-foreground"
      )}
    >
      {Icon ? <Icon className="size-4.5 shrink-0" /> : null}
      <span className="truncate">{node.label}</span>
    </Link>
  );
}

function NavGroup({ node, pathname }: { node: RouteNode; pathname: string }) {
  const active = isActive(pathname, node);
  const [open, setOpen] = useState(active);
  const Icon = node.icon;

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-expanded={open}
        className={cn(
          "flex w-full items-center gap-2.5 border-l-2 border-l-transparent px-4 py-2.5 text-sm transition-colors",
          active
            ? "font-medium text-foreground"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        )}
      >
        {Icon ? <Icon className="size-4.5 shrink-0" /> : null}
        <span className="flex-1 truncate text-left">{node.label}</span>
        <ChevronDown className={cn("size-4 transition-transform", open && "rotate-180")} />
      </button>

      {open ? (
        <div className="flex flex-col">
          {Object.entries(node.children ?? {}).map(([key, child]) =>
            isGroup(child) ? (
              <NavGroup key={key} node={child} pathname={pathname} />
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

  async function onSignOut() {
    await signOut();
    router.replace("/login");
  }

  return (
    <aside className="flex h-svh w-64 shrink-0 flex-col overflow-hidden border-r bg-sidebar">
      <Link href="/dashboard" className="flex shrink-0 items-center gap-3 border-b px-4 py-4">
        <span className="flex size-9 items-center justify-center bg-primary">
          <ShieldCheck className="size-5 text-primary-foreground" />
        </span>
        <span className="text-lg font-semibold tracking-tight">SecurePlus</span>
      </Link>

      <nav className="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto py-3">
        {Object.entries(routes as RouteMap).map(([key, node]) =>
          node.hidden ? null : isGroup(node) ? (
            <NavGroup key={key} node={node} pathname={pathname} />
          ) : (
            <NavLink key={key} node={node} pathname={pathname} />
          )
        )}
      </nav>

      <div className="shrink-0 border-t p-3">
        {identity ? (
          <div className="mb-2 px-1">
            <p className="truncate text-sm font-medium">
              {identity.user.first_name} {identity.user.last_name}
            </p>
            <p className="truncate text-xs text-muted-foreground">{identity.customer.org_name}</p>
          </div>
        ) : null}

        <Button variant="outline" className="w-full justify-start" onClick={onSignOut}>
          <LogOut />
          Sign out
        </Button>
      </div>
    </aside>
  );
};

export default Sidebar;

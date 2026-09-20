"use client";

import { useTranslations } from "next-intl";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { routes, type RouteMap, type RouteNode } from "@/lib/routes";
import { cn } from "@/lib/utils";

function owns(pathname: string, node: RouteNode) {
  if (!node.routePath) return false;
  return pathname === node.routePath || pathname.startsWith(`${node.routePath}/`);
}

function findTabParent(pathname: string, map: RouteMap): RouteNode | undefined {
  for (const node of Object.values(map)) {
    const children = Object.values(node.children ?? {});

    if (node.routePath && children.length > 0) {
      if (owns(pathname, node) || children.some((child) => owns(pathname, child))) return node;
    }

    if (node.children) {
      const nested = findTabParent(pathname, node.children);
      if (nested) return nested;
    }
  }

  return undefined;
}

const PageTabs = () => {
  const pathname = usePathname();
  const t = useTranslations("nav");
  const parent = findTabParent(pathname, routes as RouteMap);
  const tabs = Object.entries(parent?.children ?? {}).filter(
    ([, child]) => child.routePath && !child.hidden
  );

  if (tabs.length === 0) return null;

  return (
    <div className="flex gap-8 border-b">
      {tabs.map(([key, tab]) => {
        const active = owns(pathname, tab);
        const Icon = tab.icon;

        return (
          <Link
            key={key}
            href={tab.routePath as string}
            aria-current={active ? "page" : undefined}
            className={cn(
              "-mb-px flex items-center gap-2 border-b-2 px-1 py-3 text-sm leading-5 transition-colors",
              active
                ? "border-b-primary font-semibold text-primary"
                : "border-b-transparent font-medium text-text-secondary hover:text-foreground"
            )}
          >
            {Icon ? <Icon className="size-4" /> : null}
            {t(tab.labelKey)}
          </Link>
        );
      })}
    </div>
  );
};

export default PageTabs;

import { describe, expect, it } from "vitest";

import { isRouteVisible, type RouteAccess, type RouteNode } from "@/lib/routes";

function node(overrides: Partial<RouteNode> = {}): RouteNode {
  return {
    labelKey: "test",
    privileges: [],
    ...overrides,
  };
}

const openAccess: RouteAccess = { isAdmin: true, granted: [] };

describe("isRouteVisible", () => {
  it("hides a node explicitly marked hidden, regardless of anything else", () => {
    const hidden = node({ routePath: "/x", hidden: true });
    expect(isRouteVisible(hidden, openAccess)).toBe(false);
  });

  it("shows a routed leaf with no required privileges", () => {
    const leaf = node({ routePath: "/x" });
    expect(isRouteVisible(leaf, openAccess)).toBe(true);
  });

  it("hides an admin-only node for a non-admin", () => {
    const adminOnly = node({ routePath: "/console", adminOnly: true });
    expect(isRouteVisible(adminOnly, { isAdmin: false, granted: [] })).toBe(false);
    expect(isRouteVisible(adminOnly, { isAdmin: true, granted: [] })).toBe(true);
  });

  it("does not gate on adminOnly while privileges are still loading (granted === null)", () => {
    const routed = node({ routePath: "/x", privileges: ["admin.rule.view"] });
    expect(isRouteVisible(routed, { isAdmin: true, granted: null })).toBe(true);
  });

  it("hides a routed leaf when the user lacks every required privilege", () => {
    const leaf = node({ routePath: "/x", privileges: ["admin.rule.view"] });
    expect(isRouteVisible(leaf, { isAdmin: true, granted: ["admin.policy.view"] })).toBe(false);
  });

  it("shows a routed leaf when the user holds at least one required privilege", () => {
    const leaf = node({ routePath: "/x", privileges: ["admin.rule.view", "admin.rule.edit"] });
    expect(isRouteVisible(leaf, { isAdmin: true, granted: ["admin.rule.edit"] })).toBe(true);
  });

  it("hides a routeless group with no visible children", () => {
    const group = node({
      children: {
        a: node({ routePath: "/a", privileges: ["admin.a.view"] }),
        b: node({ routePath: "/b", privileges: ["admin.b.view"] }),
      },
    });

    expect(isRouteVisible(group, { isAdmin: true, granted: [] })).toBe(false);
  });

  it("shows a routeless group when at least one child is visible", () => {
    const group = node({
      children: {
        a: node({ routePath: "/a", privileges: ["admin.a.view"] }),
        b: node({ routePath: "/b", privileges: ["admin.b.view"] }),
      },
    });

    expect(isRouteVisible(group, { isAdmin: true, granted: ["admin.b.view"] })).toBe(true);
  });

  it("shows a routed parent that also has children, purely on its own privileges", () => {
    const parentWithOwnRoute = node({
      routePath: "/data-discovery",
      privileges: ["admin.discovery.configuration.view"],
      children: {
        scans: node({ routePath: "/data-discovery/scans", privileges: ["admin.discovery.scan.view"] }),
      },
    });

    expect(isRouteVisible(parentWithOwnRoute, { isAdmin: true, granted: [] })).toBe(false);
    expect(
      isRouteVisible(parentWithOwnRoute, { isAdmin: true, granted: ["admin.discovery.configuration.view"] }),
    ).toBe(true);
  });

  it("shows a routed parent lacking its own privilege if a child is still visible", () => {
    const parentWithOwnRoute = node({
      routePath: "/data-discovery",
      privileges: ["admin.discovery.configuration.view"],
      children: {
        scans: node({ routePath: "/data-discovery/scans", privileges: ["admin.discovery.scan.view"] }),
      },
    });

    expect(
      isRouteVisible(parentWithOwnRoute, { isAdmin: true, granted: ["admin.discovery.scan.view"] }),
    ).toBe(true);
  });

  it("filters out hidden children when deciding whether a group has anything visible", () => {
    const group = node({
      children: {
        a: node({ routePath: "/a", hidden: true }),
      },
    });

    expect(isRouteVisible(group, openAccess)).toBe(false);
  });

  it("hides an admin-only group from a non-admin even if a child would otherwise be visible", () => {
    const group = node({
      adminOnly: true,
      children: {
        a: node({ routePath: "/a" }),
      },
    });

    expect(isRouteVisible(group, { isAdmin: false, granted: [] })).toBe(false);
  });
});

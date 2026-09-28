import { describe, expect, it } from "vitest";

import { groupPrivileges, humanize, messageKey, parsePrivilege } from "@/lib/privilege-catalog";

describe("parsePrivilege", () => {
  it("splits a three-segment admin privilege into group and action", () => {
    expect(parsePrivilege("admin.rule.view")).toEqual({
      name: "admin.rule.view",
      group: "rule",
      action: "view",
    });
  });

  it("keeps multi-part groups together", () => {
    expect(parsePrivilege("admin.discovery.scan.view")).toEqual({
      name: "admin.discovery.scan.view",
      group: "discovery.scan",
      action: "view",
    });
  });

  it("falls back to the whole name when there is no admin prefix", () => {
    expect(parsePrivilege("view")).toEqual({ name: "view", group: "view", action: "view" });
  });
});

describe("groupPrivileges", () => {
  it("groups privileges by their namespace and orders actions view, create, edit, delete", () => {
    const groups = groupPrivileges([
      "admin.rule.delete",
      "admin.rule.view",
      "admin.rule.create",
      "admin.rule.edit",
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0].entries.map((entry) => entry.action)).toEqual(["view", "create", "edit", "delete"]);
  });

  it("orders known groups before unknown ones alphabetically", () => {
    const groups = groupPrivileges(["admin.zzz.view", "admin.rule.view", "admin.policy.view"]);

    expect(groups.map((group) => group.key)).toEqual(["rule", "policy", "zzz"]);
  });

  it("returns an empty list for no privileges", () => {
    expect(groupPrivileges([])).toEqual([]);
  });
});

describe("messageKey", () => {
  it("replaces dots with underscores for i18n lookups", () => {
    expect(messageKey("discovery.scan")).toBe("discovery_scan");
  });
});

describe("humanize", () => {
  it("capitalizes the first letter and strips separators", () => {
    expect(humanize("email_provider")).toBe("Email provider");
    expect(humanize("discovery-scan")).toBe("Discovery scan");
  });
});

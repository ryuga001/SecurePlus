import { describe, expect, it } from "vitest";

import { axisTicks, bucketLabel, formatCompact, formatCount, niceCeiling, relativeTime } from "@/lib/dashboard";

describe("niceCeiling", () => {
  it("returns the four-step floor for small values", () => {
    expect(niceCeiling(0)).toBe(4);
    expect(niceCeiling(3)).toBe(4);
    expect(niceCeiling(4)).toBe(4);
  });

  it("rounds up to a clean multiple of a nice step", () => {
    expect(niceCeiling(9)).toBe(20);
    expect(niceCeiling(23)).toBe(40);
    expect(niceCeiling(41)).toBe(80);
  });

  it("is always at least as large as the input", () => {
    for (const value of [1, 5, 17, 99, 1000, 12345]) {
      expect(niceCeiling(value)).toBeGreaterThanOrEqual(value);
    }
  });
});

describe("axisTicks", () => {
  it("returns five evenly spaced ticks from max down to zero", () => {
    expect(axisTicks(40)).toEqual([40, 30, 20, 10, 0]);
  });

  it("always ends at zero and starts at max", () => {
    const ticks = axisTicks(17);
    expect(ticks[0]).toBe(17);
    expect(ticks.at(-1)).toBe(0);
    expect(ticks).toHaveLength(5);
  });
});

describe("formatCount", () => {
  it("uses the locale matching the given language", () => {
    expect(formatCount(1234, "ENGLISH")).toBe("1,234");
  });

  it("defaults to English when no language is given", () => {
    expect(formatCount(1234)).toBe("1,234");
  });
});

describe("formatCompact", () => {
  it("compacts large numbers", () => {
    expect(formatCompact(1200)).toBe("1.2K");
    expect(formatCompact(999)).toBe("999");
  });
});

describe("bucketLabel", () => {
  it("formats a day bucket as a short date", () => {
    const label = bucketLabel("2026-09-29T00:00:00Z", "day", "UTC", "ENGLISH");
    expect(label).toBe("Sep 29");
  });

  it("formats a month bucket with month and two-digit year", () => {
    const label = bucketLabel("2026-04-01T00:00:00Z", "month", "UTC", "ENGLISH");
    expect(label).toContain("Apr");
    expect(label).toContain("26");
  });

  it("returns an empty string for an unparseable date", () => {
    expect(bucketLabel("not-a-date", "day", "UTC", "ENGLISH")).toBe("");
  });

  it("falls back to UTC when the timezone is invalid", () => {
    const label = bucketLabel("2026-09-29T00:00:00Z", "day", "Not/AZone", "ENGLISH");
    expect(label).toBe("Sep 29");
  });
});

describe("relativeTime", () => {
  it("returns an empty string for an unparseable date", () => {
    expect(relativeTime("garbage")).toBe("");
  });

  it("describes a moment a few minutes in the past", () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    const label = relativeTime(fiveMinutesAgo, "ENGLISH");
    expect(label.toLowerCase()).toBe("5m ago");
  });

  it("describes a moment a few hours in the past", () => {
    const threeHoursAgo = new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString();
    const label = relativeTime(threeHoursAgo, "ENGLISH");
    expect(label.toLowerCase()).toBe("3h ago");
  });
});

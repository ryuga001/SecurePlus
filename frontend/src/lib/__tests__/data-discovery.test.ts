import { describe, expect, it } from "vitest";

import { formatBytes, formatDuration, isScanActive } from "@/lib/data-discovery";

describe("formatBytes", () => {
  it("shows raw bytes under 1024", () => {
    expect(formatBytes(512)).toBe("512 B");
  });

  it("steps through KB/MB/GB/TB with locale-free fixed precision", () => {
    expect(formatBytes(1024)).toBe("1.00 KB");
    expect(formatBytes(9.44 * 1024 * 1024)).toBe("9.44 MB");
    expect(formatBytes(12.3 * 1024 * 1024 * 1024)).toBe("12.3 GB");
  });

  it("returns an em dash for invalid input", () => {
    expect(formatBytes(-1)).toBe("—");
    expect(formatBytes(Number.NaN)).toBe("—");
    expect(formatBytes(Number.POSITIVE_INFINITY)).toBe("—");
  });
});

describe("formatDuration", () => {
  it("shows milliseconds under one second", () => {
    expect(formatDuration(250)).toBe("250 ms");
  });

  it("shows seconds under one minute", () => {
    expect(formatDuration(12300)).toBe("12.3 s");
  });

  it("shows minutes and seconds under one hour", () => {
    expect(formatDuration(5 * 60 * 1000 + 30 * 1000)).toBe("5m 30s");
  });

  it("shows hours and minutes at or above one hour", () => {
    expect(formatDuration(2 * 60 * 60 * 1000 + 15 * 60 * 1000)).toBe("2h 15m");
  });

  it("returns an em dash for invalid input", () => {
    expect(formatDuration(-5)).toBe("—");
    expect(formatDuration(Number.NaN)).toBe("—");
  });
});

describe("isScanActive", () => {
  it("treats PENDING and RUNNING as active", () => {
    expect(isScanActive("PENDING")).toBe(true);
    expect(isScanActive("RUNNING")).toBe(true);
  });

  it("treats every other status, and undefined, as inactive", () => {
    expect(isScanActive("COMPLETED")).toBe(false);
    expect(isScanActive("FAILED")).toBe(false);
    expect(isScanActive("PARTIAL")).toBe(false);
    expect(isScanActive(undefined)).toBe(false);
    expect(isScanActive("not-a-real-status")).toBe(false);
  });
});

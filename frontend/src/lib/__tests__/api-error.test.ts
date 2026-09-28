import { describe, expect, it } from "vitest";

import { apiErrorCode, apiErrorMessage, apiErrorStatus } from "@/lib/api-error";

function rtkError(status: number, error?: string, message?: string) {
  return { status, data: { error, message } };
}

describe("apiErrorMessage", () => {
  it("extracts the message from an RTK Query error shape", () => {
    expect(apiErrorMessage(rtkError(404, "not_found", "policy not found"))).toBe("policy not found");
  });

  it("falls back to the provided default when there is no message", () => {
    expect(apiErrorMessage({}, "fallback text")).toBe("fallback text");
    expect(apiErrorMessage(null, "fallback text")).toBe("fallback text");
    expect(apiErrorMessage(undefined, "fallback text")).toBe("fallback text");
  });

  it("uses the built-in default when none is given", () => {
    expect(apiErrorMessage({})).toBe("Something went wrong");
  });
});

describe("apiErrorCode", () => {
  it("extracts the machine-readable error code", () => {
    expect(apiErrorCode(rtkError(409, "role_name_taken", "taken"))).toBe("role_name_taken");
  });

  it("returns undefined when there is no code", () => {
    expect(apiErrorCode({})).toBeUndefined();
    expect(apiErrorCode("a plain string")).toBeUndefined();
  });
});

describe("apiErrorStatus", () => {
  it("extracts a numeric status", () => {
    expect(apiErrorStatus(rtkError(403))).toBe(403);
  });

  it("returns undefined when status is missing or not a number", () => {
    expect(apiErrorStatus({})).toBeUndefined();
    expect(apiErrorStatus({ status: "FETCH_ERROR" })).toBeUndefined();
    expect(apiErrorStatus(null)).toBeUndefined();
  });
});

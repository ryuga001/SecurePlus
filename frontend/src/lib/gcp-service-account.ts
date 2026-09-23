export type ServiceAccountKey = {
  fileName: string;
  raw: string;
  projectId: string;
  clientEmail: string;
};

export type ServiceAccountKeyError =
  | "tooLarge"
  | "wrongType"
  | "unreadable"
  | "notJson"
  | "notServiceAccount"
  | "missingFields";

export type ParseResult =
  | { ok: true; value: ServiceAccountKey }
  | { ok: false; error: ServiceAccountKeyError };

const PRIVATE_KEY_PREFIX = "-----BEGIN PRIVATE KEY-----";

function nonEmptyString(value: unknown): value is string {
  return typeof value === "string" && value.trim() !== "";
}

export function parseServiceAccountKey(fileName: string, text: string): ParseResult {
  const cleaned = text.replace(/^﻿/, "");

  let parsed: unknown;

  try {
    parsed = JSON.parse(cleaned);
  } catch {
    return { ok: false, error: "notJson" };
  }

  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    return { ok: false, error: "notJson" };
  }

  const key = parsed as Record<string, unknown>;

  if (key.type !== "service_account") {
    return { ok: false, error: "notServiceAccount" };
  }

  if (
    !nonEmptyString(key.project_id) ||
    !nonEmptyString(key.client_email) ||
    !nonEmptyString(key.client_id) ||
    !nonEmptyString(key.private_key_id) ||
    !nonEmptyString(key.private_key)
  ) {
    return { ok: false, error: "missingFields" };
  }

  if (!key.private_key.includes(PRIVATE_KEY_PREFIX)) {
    return { ok: false, error: "missingFields" };
  }

  return {
    ok: true,
    value: {
      fileName,
      raw: cleaned,
      projectId: key.project_id,
      clientEmail: key.client_email,
    },
  };
}

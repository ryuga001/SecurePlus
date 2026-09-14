const BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";

export type Theme = "LIGHT" | "DARK";

export type Language = "ENGLISH" | "JAPANESE" | "SPANISH";

export type Branding = {
  logo_url: string | null;
  theme: Theme;
  language: Language;
  timezone: string;
};

export type Identity = {
  user: {
    id: number;
    email: string;
    first_name: string;
    last_name: string;
    role?: string;
  };
  customer: { id: number; org_name: string };
  branding?: Branding;
  csrf_token: string;
};

export type PrivilegesResponse = { privileges: string[] };

export type MessageResponse = { message: string; expires_in?: number };

export type VerifiedResponse = { registration_token: string; expires_in: number };

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

let csrfToken: string | null = null;
let refreshInflight: Promise<boolean> | null = null;

const NO_REFRESH = new Set([
  "/auth/login",
  "/auth/refresh",
  "/auth/csrf",
  "/auth/register/start",
  "/auth/register/verify",
  "/auth/register/resend",
  "/auth/register/complete",
  "/auth/password/forgot",
  "/auth/password/reset",
]);

export function setCsrfToken(token: string | null) {
  csrfToken = token;
}

export function getCsrfToken() {
  return csrfToken;
}

export const API_BASE_URL = BASE;

async function ensureCsrfToken() {
  if (csrfToken) return;

  const res = await fetch(`${BASE}/auth/csrf`, { credentials: "include" });
  if (!res.ok) return;

  const data = (await res.json()) as { token: string };
  csrfToken = data.token;
}

function refreshOnce(): Promise<boolean> {
  refreshInflight ??= fetch(`${BASE}/auth/refresh`, {
    method: "POST",
    credentials: "include",
    headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
  })
    .then(async (res) => {
      if (!res.ok) return false;
      const data = (await res.json()) as Identity;
      csrfToken = data.csrf_token;
      return true;
    })
    .catch(() => false)
    .finally(() => {
      queueMicrotask(() => {
        refreshInflight = null;
      });
    });

  return refreshInflight;
}

async function toError(res: Response) {
  let code = "unknown";
  let message = res.statusText || "Request failed";

  try {
    const body = (await res.json()) as { error?: string; message?: string };
    if (body.error) code = body.error;
    if (body.message) message = body.message;
  } catch {
    code = "unknown";
  }

  return new ApiError(res.status, code, message);
}

type Options = { method?: string; body?: unknown };

async function request<T>(path: string, options: Options = {}, allowRetry = true): Promise<T> {
  const method = options.method ?? "GET";

  if (method !== "GET") {
    await ensureCsrfToken();
  }

  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (csrfToken) headers["X-CSRF-Token"] = csrfToken;

  const res = await fetch(BASE + path, {
    method,
    credentials: "include",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });

  if (res.status === 401 && allowRetry && !NO_REFRESH.has(path)) {
    if (await refreshOnce()) {
      return request<T>(path, options, false);
    }
  }

  if (!res.ok) throw await toError(res);
  if (res.status === 204) return undefined as T;

  const data = (await res.json()) as T;

  if (data && typeof data === "object" && "csrf_token" in data) {
    const token = (data as { csrf_token?: unknown }).csrf_token;
    if (typeof token === "string") csrfToken = token;
  }

  return data;
}

export const api = {
  login: (email: string, password: string) =>
    request<Identity>("/auth/login", { method: "POST", body: { email, password } }),

  startRegistration: (email: string) =>
    request<MessageResponse>("/auth/register/start", { method: "POST", body: { email } }),

  verifyEmail: (email: string, code: string) =>
    request<VerifiedResponse>("/auth/register/verify", { method: "POST", body: { email, code } }),

  completeRegistration: (body: {
    registration_token: string;
    org_name: string;
    first_name: string;
    last_name: string;
    password: string;
  }) => request<Identity>("/auth/register/complete", { method: "POST", body }),

  resendCode: (email: string) =>
    request<MessageResponse>("/auth/register/resend", { method: "POST", body: { email } }),

  forgotPassword: (email: string) =>
    request<MessageResponse>("/auth/password/forgot", { method: "POST", body: { email } }),

  resetPassword: (email: string, reset_code: string, new_password: string) =>
    request<MessageResponse>("/auth/password/reset", {
      method: "POST",
      body: { email, reset_code, new_password },
    }),

  me: () => request<Identity>("/me"),

  privileges: () => request<PrivilegesResponse>("/me/privileges"),

  logout: () => request<void>("/auth/logout", { method: "POST" }),
};

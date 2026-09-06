type ApiErrorBody = { error?: string; message?: string };

function errorBody(error: unknown): ApiErrorBody | undefined {
  if (typeof error !== "object" || error === null) return undefined;

  const data = (error as { data?: unknown }).data;
  if (typeof data !== "object" || data === null) return undefined;

  return data as ApiErrorBody;
}

export function apiErrorMessage(error: unknown, fallback = "Something went wrong") {
  return errorBody(error)?.message ?? fallback;
}

export function apiErrorCode(error: unknown) {
  return errorBody(error)?.error;
}

export function apiErrorStatus(error: unknown) {
  if (typeof error !== "object" || error === null) return undefined;

  const status = (error as { status?: unknown }).status;

  return typeof status === "number" ? status : undefined;
}

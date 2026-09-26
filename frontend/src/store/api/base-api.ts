import {
  createApi,
  fetchBaseQuery,
  type BaseQueryFn,
  type FetchArgs,
  type FetchBaseQueryError,
} from "@reduxjs/toolkit/query/react";

import {
  API_BASE_URL,
  getCsrfToken,
  isCsrfFailure,
  recoverCsrfToken,
  recoverSession,
} from "@/lib/api";

const rawBaseQuery = fetchBaseQuery({
  baseUrl: API_BASE_URL,
  credentials: "include",
  prepareHeaders: (headers) => {
    const token = getCsrfToken();
    if (token) headers.set("X-CSRF-Token", token);
    return headers;
  },
});

const baseQueryWithReauth: BaseQueryFn<string | FetchArgs, unknown, FetchBaseQueryError> = async (
  args,
  api,
  extraOptions,
) => {
  const result = await rawBaseQuery(args, api, extraOptions);

  if (result.error?.status === 401) {
    return (await recoverSession()) ? rawBaseQuery(args, api, extraOptions) : result;
  }

  if (result.error?.status === 403 && isCsrfFailure(result.error.data)) {
    return (await recoverCsrfToken()) ? rawBaseQuery(args, api, extraOptions) : result;
  }

  return result;
};

export const baseApi = createApi({
  reducerPath: "api",
  baseQuery: baseQueryWithReauth,
  tagTypes: [
    "Privileges",
    "User",
    "Group",
    "EmailUser",
    "Rule",
    "EmailPolicy",
    "EmailConfiguration",
    "AuditLog",
    "EmailIncident",
    "Alert",
    "DiscoveryConfiguration",
    "DiscoveryPolicy",
    "DiscoveryScan",
    "Branding",
  ],
  endpoints: () => ({}),
});

import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

import { API_BASE_URL, getCsrfToken } from "@/lib/api";

export const baseApi = createApi({
  reducerPath: "api",
  baseQuery: fetchBaseQuery({
    baseUrl: API_BASE_URL,
    credentials: "include",
    prepareHeaders: (headers) => {
      const token = getCsrfToken();
      if (token) headers.set("X-CSRF-Token", token);
      return headers;
    },
  }),
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
    "Branding",
  ],
  endpoints: () => ({}),
});

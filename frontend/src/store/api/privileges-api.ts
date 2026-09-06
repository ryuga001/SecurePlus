import type { PrivilegesResponse } from "@/lib/api";
import { baseApi } from "@/store/api/base-api";

export const privilegesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getPrivileges: builder.query<string[], void>({
      query: () => "/me/privileges",
      transformResponse: (response: PrivilegesResponse | string[]) =>
        Array.isArray(response) ? response : (response?.privileges ?? []),
      providesTags: ["Privileges"],
    }),
  }),
});

export const { useGetPrivilegesQuery } = privilegesApi;

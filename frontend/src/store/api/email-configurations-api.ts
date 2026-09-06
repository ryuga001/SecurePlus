import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";

export type EmailProvider = "outlook365" | "gmail";

export type EmailConfiguration = {
  id: number;
  name: string;
  domain: string;
  provider: EmailProvider;
  dkim_public_key?: string | null;
  has_access_token: boolean;
  access_token_expires_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type EmailConfigurationInput = {
  name: string;
  domain: string;
  provider: EmailProvider;
};

export type UpdateEmailConfigurationArgs = EmailConfigurationInput & { id: number };

export type DkimKeyResponse = {
  selector: string;
  record_name: string;
  dkim_public_key: string;
};

export type AccessTokenResponse = {
  access_token: string;
  expires_at: string;
};

const BASE_URL = "/admin/email/configurations";

export const emailConfigurationsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listEmailConfigurations: builder.query<PaginatedResponse<EmailConfiguration>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "EmailConfiguration" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({
          type: "EmailConfiguration" as const,
          id: item.id,
        })),
      ],
    }),

    getEmailConfiguration: builder.query<EmailConfiguration, number>({
      query: (id) => `${BASE_URL}/${id}`,
      providesTags: (result, error, id) => [{ type: "EmailConfiguration", id }],
    }),

    createEmailConfiguration: builder.mutation<EmailConfiguration, EmailConfigurationInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [{ type: "EmailConfiguration", id: "LIST" }],
    }),

    updateEmailConfiguration: builder.mutation<EmailConfiguration, UpdateEmailConfigurationArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "EmailConfiguration", id: arg.id },
        { type: "EmailConfiguration", id: "LIST" },
      ],
    }),

    deleteEmailConfiguration: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "EmailConfiguration", id },
        { type: "EmailConfiguration", id: "LIST" },
      ],
    }),

    generateDkimKey: builder.mutation<DkimKeyResponse, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}/dkim`, method: "POST" }),
      invalidatesTags: (result, error, id) => [{ type: "EmailConfiguration", id }],
    }),

    generateAccessToken: builder.mutation<AccessTokenResponse, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}/access-token`, method: "POST" }),
      invalidatesTags: (result, error, id) => [{ type: "EmailConfiguration", id }],
    }),
  }),
});

export const {
  useListEmailConfigurationsQuery,
  useGetEmailConfigurationQuery,
  useCreateEmailConfigurationMutation,
  useUpdateEmailConfigurationMutation,
  useDeleteEmailConfigurationMutation,
  useGenerateDkimKeyMutation,
  useGenerateAccessTokenMutation,
} = emailConfigurationsApi;

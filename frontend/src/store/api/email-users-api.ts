import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";
import type { EmailGroup } from "@/store/api/email-groups-api";

export type EmailUser = {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  created_at: string;
  updated_at: string;
};

export type EmailUserInput = {
  email: string;
  first_name: string;
  last_name: string;
};

export type UpdateEmailUserArgs = EmailUserInput & { id: number };

const BASE_URL = "/admin/email/users";

export const emailUsersApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listEmailUsers: builder.query<PaginatedResponse<EmailUser>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "EmailUser" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({ type: "EmailUser" as const, id: item.id })),
      ],
    }),

    listUserGroups: builder.query<PaginatedResponse<EmailGroup>, number>({
      query: (id) => `${BASE_URL}/${id}/groups`,
      providesTags: (result, error, id) => [{ type: "EmailUser", id }],
    }),

    createEmailUser: builder.mutation<EmailUser, EmailUserInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [{ type: "EmailUser", id: "LIST" }],
    }),

    updateEmailUser: builder.mutation<EmailUser, UpdateEmailUserArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "EmailUser", id: arg.id },
        { type: "EmailUser", id: "LIST" },
      ],
    }),

    deleteEmailUser: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "EmailUser", id },
        { type: "EmailUser", id: "LIST" },
        { type: "Group", id: "LIST" },
      ],
    }),
  }),
});

export const {
  useListEmailUsersQuery,
  useListUserGroupsQuery,
  useCreateEmailUserMutation,
  useUpdateEmailUserMutation,
  useDeleteEmailUserMutation,
} = emailUsersApi;

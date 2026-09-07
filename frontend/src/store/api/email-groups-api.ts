import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";
import type { EmailUser } from "@/store/api/email-users-api";

export type EmailGroup = {
  id: number;
  name: string;
  type: "USER";
  member_count: number;
  created_at: string;
  updated_at: string;
};

export type EmailGroupInput = {
  name: string;
};

export type UpdateEmailGroupArgs = EmailGroupInput & { id: number };

export type MembershipArgs = { groupId: number; emailUserId: number };

const BASE_URL = "/admin/email/groups";

export const emailGroupsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listEmailGroups: builder.query<PaginatedResponse<EmailGroup>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "Group" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({ type: "Group" as const, id: item.id })),
      ],
    }),

    listGroupMembers: builder.query<PaginatedResponse<EmailUser>, number>({
      query: (id) => `${BASE_URL}/${id}/users`,
      providesTags: (result, error, id) => [{ type: "Group", id }],
    }),

    createEmailGroup: builder.mutation<EmailGroup, EmailGroupInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [{ type: "Group", id: "LIST" }],
    }),

    updateEmailGroup: builder.mutation<EmailGroup, UpdateEmailGroupArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "Group", id: arg.id },
        { type: "Group", id: "LIST" },
      ],
    }),

    deleteEmailGroup: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "Group", id },
        { type: "Group", id: "LIST" },
        { type: "EmailPolicy", id: "LIST" },
      ],
    }),

    addGroupMember: builder.mutation<void, MembershipArgs>({
      query: ({ groupId, emailUserId }) => ({
        url: `${BASE_URL}/${groupId}/users`,
        method: "POST",
        body: { email_user_id: emailUserId },
      }),
      invalidatesTags: (result, error, arg) => [
        { type: "Group", id: arg.groupId },
        { type: "Group", id: "LIST" },
        { type: "EmailUser", id: arg.emailUserId },
      ],
    }),

    removeGroupMember: builder.mutation<void, MembershipArgs>({
      query: ({ groupId, emailUserId }) => ({
        url: `${BASE_URL}/${groupId}/users/${emailUserId}`,
        method: "DELETE",
      }),
      invalidatesTags: (result, error, arg) => [
        { type: "Group", id: arg.groupId },
        { type: "Group", id: "LIST" },
        { type: "EmailUser", id: arg.emailUserId },
      ],
    }),
  }),
});

export const {
  useListEmailGroupsQuery,
  useListGroupMembersQuery,
  useCreateEmailGroupMutation,
  useUpdateEmailGroupMutation,
  useDeleteEmailGroupMutation,
  useAddGroupMemberMutation,
  useRemoveGroupMemberMutation,
} = emailGroupsApi;

import type {
  DataTableQueryArgs,
  PaginatedResponse,
} from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";
import type { EmailUser } from "@/store/api/email-users-api";

type EmailGroupMember = {
  id: number;
  name: string;
  email: string;
};

export type EmailGroup = {
  id: number;
  name: string;
  type: "USER";
  member_count: number;
  members: EmailGroupMember[];
  created_at: string;
  updated_at: string;
};

export type EmailGroupInput = {
  name: string;
  member_ids: number[];
};

export type UpdateEmailGroupArgs = EmailGroupInput & {
  id: number;
};

const BASE_URL = "/admin/email/groups";

export const emailGroupsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listEmailGroups: builder.query<
      PaginatedResponse<EmailGroup>,
      DataTableQueryArgs
    >({
      query: (args) => ({
        url: BASE_URL,
        params: listParams(args),
      }),
      providesTags: (result) => [
        { type: "Group", id: "LIST" },
        ...(result?.items ?? []).map((item) => ({
          type: "Group" as const,
          id: item.id,
        })),
      ],
    }),

    listGroupMembers: builder.query<
      PaginatedResponse<EmailUser>,
      {
        groupId: number;
        page?: number;
        pageSize?: number;
        search?: string;
      }
    >({
      query: ({ groupId, page = 1, pageSize = 25, search }) => ({
        url: `${BASE_URL}/${groupId}/users`,
        params: {
          page,
          page_size: pageSize,
          ...(search ? { search } : {}),
        },
      }),
      providesTags: (result, error, { groupId }) => [
        { type: "Group", id: groupId },
      ],
    }),

    createEmailGroup: builder.mutation<EmailGroup, EmailGroupInput>({
      query: (body) => ({
        url: BASE_URL,
        method: "POST",
        body,
      }),
      invalidatesTags: (result, error, arg) => [
        { type: "Group", id: "LIST" },
        ...arg.member_ids.map((id) => ({
          type: "EmailUser" as const,
          id,
        })),
      ],
    }),

    updateEmailGroup: builder.mutation<EmailGroup, UpdateEmailGroupArgs>({
      query: ({ id, ...body }) => ({
        url: `${BASE_URL}/${id}`,
        method: "PUT",
        body,
      }),
      invalidatesTags: (result, error, arg) => [
        { type: "Group", id: arg.id },
        { type: "Group", id: "LIST" },
        ...arg.member_ids.map((id) => ({
          type: "EmailUser" as const,
          id,
        })),
      ],
    }),

    deleteEmailGroup: builder.mutation<void, number>({
      query: (id) => ({
        url: `${BASE_URL}/${id}`,
        method: "DELETE",
      }),
      invalidatesTags: (result, error, id) => [
        { type: "Group", id },
        { type: "Group", id: "LIST" },
        { type: "EmailPolicy", id: "LIST" },
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
} = emailGroupsApi;

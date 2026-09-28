import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";

export type UserRoleReference = {
  id: number;
  name: string;
  system: boolean;
};

export type User = {
  id: number;
  first_name: string;
  last_name: string;
  email: string;
  role: UserRoleReference | null;
  created_at: string;
};

export type CreateUserInput = {
  first_name: string;
  last_name: string;
  email: string;
  role_id: number;
};

export type UpdateUserArgs = {
  id: number;
  first_name: string;
  last_name: string;
  role_id: number;
};

export type CreateUserResult = {
  user: User;
  invitation_sent: boolean;
};

const BASE_URL = "/admin/users";

export const usersApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listUsers: builder.query<PaginatedResponse<User>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "User" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({ type: "User" as const, id: item.id })),
      ],
    }),

    getUser: builder.query<User, number>({
      query: (id) => `${BASE_URL}/${id}`,
      providesTags: (result, error, id) => [{ type: "User", id }],
    }),

    createUser: builder.mutation<CreateUserResult, CreateUserInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [
        { type: "User", id: "LIST" },
        { type: "Role", id: "LIST" },
      ],
    }),

    updateUser: builder.mutation<User, UpdateUserArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "User", id: arg.id },
        { type: "User", id: "LIST" },
        { type: "Role", id: "LIST" },
      ],
    }),

    deleteUser: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "User", id },
        { type: "User", id: "LIST" },
        { type: "Role", id: "LIST" },
      ],
    }),
  }),
});

export const {
  useListUsersQuery,
  useGetUserQuery,
  useCreateUserMutation,
  useUpdateUserMutation,
  useDeleteUserMutation,
} = usersApi;

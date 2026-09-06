import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";

export type User = {
  id: number;
  first_name: string;
  last_name: string;
  email: string;
  role: string;
  status: string;
  last_login_at?: string;
  created_at: string;
};

export type UserRole = {
  id: number;
  name: string;
  description?: string;
  users_count: number;
  privileges_count: number;
  updated_at: string;
};

function listParams({ page, pageSize, sortBy, sortDir, filters }: DataTableQueryArgs) {
  return {
    page,
    page_size: pageSize,
    ...(sortBy ? { sort_by: sortBy, sort_dir: sortDir } : {}),
    ...filters,
  };
}

export const usersApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listUsers: builder.query<PaginatedResponse<User>, DataTableQueryArgs>({
      query: (args) => ({ url: "/admin/users", params: listParams(args) }),
      providesTags: ["User"],
    }),

    listUserRoles: builder.query<PaginatedResponse<UserRole>, DataTableQueryArgs>({
      query: (args) => ({ url: "/admin/roles", params: listParams(args) }),
      providesTags: ["Group"],
    }),

    deleteUser: builder.mutation<void, number>({
      query: (id) => ({ url: `/admin/users/${id}`, method: "DELETE" }),
      invalidatesTags: ["User"],
    }),
  }),
});

export const { useListUsersQuery, useListUserRolesQuery, useDeleteUserMutation } = usersApi;

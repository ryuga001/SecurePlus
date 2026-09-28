import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";

export type RoleUserPreview = {
  id: number;
  name: string;
  email: string;
};

export type RoleListItem = {
  id: number;
  name: string;
  description: string;
  system: boolean;
  privileges: string[];
  privilege_count: number;
  users: RoleUserPreview[];
  user_count: number;
  updated_at: string;
};

export type Role = {
  id: number;
  name: string;
  description: string;
  system: boolean;
  privileges: string[];
  users: RoleUserPreview[];
  user_count: number;
  created_at: string;
  updated_at: string;
};

export type RoleOption = {
  id: number;
  name: string;
  system: boolean;
};

export type RoleInput = {
  name: string;
  description: string;
  privileges: string[];
};

export type UpdateRoleArgs = RoleInput & { id: number };

const BASE_URL = "/admin/roles";

export const rolesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listRoles: builder.query<PaginatedResponse<RoleListItem>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "Role" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({ type: "Role" as const, id: item.id })),
      ],
    }),

    listRoleOptions: builder.query<RoleOption[], void>({
      query: () => `${BASE_URL}/options`,
      transformResponse: (response: { items?: RoleOption[] }) => response?.items ?? [],
      providesTags: [{ type: "Role", id: "OPTIONS" }],
    }),

    listAssignablePrivileges: builder.query<string[], void>({
      query: () => "/admin/privileges",
      transformResponse: (response: { privileges?: string[] }) => response?.privileges ?? [],
    }),

    getRole: builder.query<Role, number>({
      query: (id) => `${BASE_URL}/${id}`,
      providesTags: (result, error, id) => [{ type: "Role", id }],
    }),

    createRole: builder.mutation<Role, RoleInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [
        { type: "Role", id: "LIST" },
        { type: "Role", id: "OPTIONS" },
      ],
    }),

    updateRole: builder.mutation<Role, UpdateRoleArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "Role", id: arg.id },
        { type: "Role", id: "LIST" },
        { type: "Role", id: "OPTIONS" },
        { type: "User", id: "LIST" },
      ],
    }),

    deleteRole: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "Role", id },
        { type: "Role", id: "LIST" },
        { type: "Role", id: "OPTIONS" },
      ],
    }),
  }),
});

export const {
  useListRolesQuery,
  useListRoleOptionsQuery,
  useListAssignablePrivilegesQuery,
  useGetRoleQuery,
  useCreateRoleMutation,
  useUpdateRoleMutation,
  useDeleteRoleMutation,
} = rolesApi;

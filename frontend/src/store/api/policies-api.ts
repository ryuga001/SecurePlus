import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";

export type PolicyReference = {
  id: number;
  name: string;
};

export type PolicyAction = "BLOCK" | "AUDIT" | "QUARANTINE" | "REDACT";

export type RestrictionMode = "NONE" | "BLOCK" | "ALLOW";

export type Restriction = {
  mode: RestrictionMode;
  values: string[];
};

export type FileType = {
  id: number;
  extension: string;
  label: string;
};

export type Policy = {
  id: number;
  policy_name: string;
  type: "EMAIL";
  action: PolicyAction;
  active: boolean;
  domain_restriction: Restriction;
  attachment_restriction: Restriction;
  groups: PolicyReference[];
  rules: PolicyReference[];
  created_at: string;
  updated_at: string;
};

export type PolicyListItem = {
  id: number;
  policy_name: string;
  type: "EMAIL";
  action: PolicyAction;
  active: boolean;
  domain_restriction_mode: RestrictionMode;
  attachment_restriction_mode: RestrictionMode;
  group_count: number;
  rule_count: number;
  created_at: string;
  updated_at: string;
};

export type PolicyInput = {
  policy_name: string;
  action: PolicyAction;
  active: boolean;
  group_ids: number[];
  rule_ids: number[];
  domain_restriction: Restriction;
  attachment_restriction: Restriction;
};

export type UpdatePolicyArgs = PolicyInput & { id: number };

export type PolicyStatusArgs = { id: number; active: boolean };

const BASE_URL = "/admin/policies";

export const policiesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listPolicies: builder.query<PaginatedResponse<PolicyListItem>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "EmailPolicy" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({ type: "EmailPolicy" as const, id: item.id })),
      ],
    }),

    getPolicy: builder.query<Policy, number>({
      query: (id) => `${BASE_URL}/${id}`,
      providesTags: (result, error, id) => [{ type: "EmailPolicy", id }],
    }),

    listFileTypes: builder.query<PaginatedResponse<FileType>, void>({
      query: () => "/admin/file-types",
    }),

    createPolicy: builder.mutation<Policy, PolicyInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [{ type: "EmailPolicy", id: "LIST" }],
    }),

    updatePolicy: builder.mutation<Policy, UpdatePolicyArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "EmailPolicy", id: arg.id },
        { type: "EmailPolicy", id: "LIST" },
      ],
    }),

    setPolicyStatus: builder.mutation<Policy, PolicyStatusArgs>({
      query: ({ id, active }) => ({
        url: `${BASE_URL}/${id}/status`,
        method: "PATCH",
        body: { active },
      }),
      invalidatesTags: (result, error, arg) => [
        { type: "EmailPolicy", id: arg.id },
        { type: "EmailPolicy", id: "LIST" },
      ],
    }),

    deletePolicy: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "EmailPolicy", id },
        { type: "EmailPolicy", id: "LIST" },
      ],
    }),
  }),
});

export const {
  useListPoliciesQuery,
  useGetPolicyQuery,
  useListFileTypesQuery,
  useCreatePolicyMutation,
  useUpdatePolicyMutation,
  useSetPolicyStatusMutation,
  useDeletePolicyMutation,
} = policiesApi;

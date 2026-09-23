import type {
  DataTableQueryArgs,
  PaginatedResponse,
} from "@/components/data-table/types";
import type {
  ConfigurationType,
  DiscoveryStatus,
  SourceType,
} from "@/lib/data-discovery";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";
import type { PolicyReference } from "@/store/api/policies-api";

export type DiscoveryPolicyListItem = {
  id: number;
  name: string;
  description: string;
  configuration_id: number;
  configuration_name: string;
  configuration_type: ConfigurationType;
  source_type: SourceType;
  status: DiscoveryStatus;
  target_count: number;
  file_type_count: number;
  rule_count: number;
  created_at: string;
  updated_at: string;
};

export type DiscoveryPolicy = {
  id: number;
  name: string;
  description: string;
  configuration_id: number;
  configuration_name: string;
  configuration_type: ConfigurationType;
  source_type: SourceType;
  status: DiscoveryStatus;
  target_list: string[];
  file_types: string[];
  rules: PolicyReference[];
  created_at: string;
  updated_at: string;
};

export type DiscoveryPolicyInput = {
  name: string;
  description: string;
  configuration_id: number;
  source_type: SourceType;
  status: DiscoveryStatus;
  target_list: string[];
  file_types: string[];
  rule_ids: number[];
};

export type UpdateDiscoveryPolicyArgs = DiscoveryPolicyInput & { id: number };

const BASE_URL = "/admin/data-discovery/policies";

export const dataDiscoveryPoliciesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listDiscoveryPolicies: builder.query<
      PaginatedResponse<DiscoveryPolicyListItem>,
      DataTableQueryArgs
    >({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "DiscoveryPolicy" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({
          type: "DiscoveryPolicy" as const,
          id: item.id,
        })),
      ],
    }),

    getDiscoveryPolicy: builder.query<DiscoveryPolicy, number>({
      query: (id) => `${BASE_URL}/${id}`,
      providesTags: (result, error, id) => [{ type: "DiscoveryPolicy", id }],
    }),

    createDiscoveryPolicy: builder.mutation<DiscoveryPolicy, DiscoveryPolicyInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [
        { type: "DiscoveryPolicy", id: "LIST" },
        { type: "DiscoveryConfiguration", id: "LIST" },
      ],
    }),

    updateDiscoveryPolicy: builder.mutation<DiscoveryPolicy, UpdateDiscoveryPolicyArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "DiscoveryPolicy", id: arg.id },
        { type: "DiscoveryPolicy", id: "LIST" },
        { type: "DiscoveryConfiguration", id: "LIST" },
      ],
    }),

    deleteDiscoveryPolicy: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "DiscoveryPolicy", id },
        { type: "DiscoveryPolicy", id: "LIST" },
        { type: "DiscoveryConfiguration", id: "LIST" },
      ],
    }),
  }),
});

export const {
  useListDiscoveryPoliciesQuery,
  useGetDiscoveryPolicyQuery,
  useCreateDiscoveryPolicyMutation,
  useUpdateDiscoveryPolicyMutation,
  useDeleteDiscoveryPolicyMutation,
} = dataDiscoveryPoliciesApi;

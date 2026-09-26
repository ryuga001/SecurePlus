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

export type DiscoveryConfigurationListItem = {
  id: number;
  name: string;
  description: string;
  configuration_type: ConfigurationType;
  status: DiscoveryStatus;
  policy_count: number;
  last_tested_at: string;
  created_at: string;
  updated_at: string;
};

export type DiscoveryConfiguration = DiscoveryConfigurationListItem & {
  config: Record<string, string>;
  has_credential: boolean;
};

export type DiscoveryConfigurationInput = {
  name: string;
  description: string;
  configuration_type: ConfigurationType;
  status: DiscoveryStatus;
  config: Record<string, string>;
  secret: Record<string, string>;
};

export type UpdateDiscoveryConfigurationArgs = Omit<
  DiscoveryConfigurationInput,
  "configuration_type"
> & { id: number };

export type TestConfigurationArgs = {
  configuration_id?: number;
  configuration_type?: ConfigurationType;
  config: Record<string, string>;
  secret: Record<string, string>;
};

export type SourceCapability = {
  configuration_type: ConfigurationType;
  source_type: SourceType;
  label: string;
};

export type TargetOption = {
  value: string;
  label: string;
  description: string;
  kind: string;
  expandable: boolean;
};

export type TargetPage = {
  items: TargetOption[];
  next_cursor: string;
};

export type BrowseTargetsArgs = {
  configurationId: number;
  sourceType: SourceType;
  search: string;
  parent: string;
};

export const TARGET_PAGE_SIZE = 50;

const BASE_URL = "/admin/data-discovery/configurations";

export const dataDiscoveryConfigurationsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listDiscoveryConfigurations: builder.query<
      PaginatedResponse<DiscoveryConfigurationListItem>,
      DataTableQueryArgs
    >({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "DiscoveryConfiguration" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({
          type: "DiscoveryConfiguration" as const,
          id: item.id,
        })),
      ],
    }),

    getDiscoveryConfiguration: builder.query<DiscoveryConfiguration, number>({
      query: (id) => `${BASE_URL}/${id}`,
      providesTags: (result, error, id) => [{ type: "DiscoveryConfiguration", id }],
    }),

    createDiscoveryConfiguration: builder.mutation<
      DiscoveryConfiguration,
      DiscoveryConfigurationInput
    >({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [{ type: "DiscoveryConfiguration", id: "LIST" }],
    }),

    updateDiscoveryConfiguration: builder.mutation<
      DiscoveryConfiguration,
      UpdateDiscoveryConfigurationArgs
    >({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "DiscoveryConfiguration", id: arg.id },
        { type: "DiscoveryConfiguration", id: "LIST" },
      ],
    }),

    deleteDiscoveryConfiguration: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "DiscoveryConfiguration", id },
        { type: "DiscoveryConfiguration", id: "LIST" },
      ],
    }),

    testDiscoveryConfiguration: builder.mutation<{ status: string }, TestConfigurationArgs>({
      query: (body) => ({ url: `${BASE_URL}/test`, method: "POST", body }),
    }),

    listSourceCapabilities: builder.query<PaginatedResponse<SourceCapability>, void>({
      query: () => "/admin/data-discovery/source-capabilities",
    }),

    browseDiscoveryTargets: builder.infiniteQuery<TargetPage, BrowseTargetsArgs, string>({
      infiniteQueryOptions: {
        initialPageParam: "",
        getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
      },
      query: ({ queryArg, pageParam }) => ({
        url: `${BASE_URL}/${queryArg.configurationId}/targets`,
        params: {
          source_type: queryArg.sourceType,
          limit: TARGET_PAGE_SIZE,
          ...(queryArg.search ? { search: queryArg.search } : {}),
          ...(queryArg.parent ? { parent: queryArg.parent } : {}),
          ...(pageParam ? { cursor: pageParam } : {}),
        },
      }),
    }),
  }),
});

export const {
  useListDiscoveryConfigurationsQuery,
  useGetDiscoveryConfigurationQuery,
  useCreateDiscoveryConfigurationMutation,
  useUpdateDiscoveryConfigurationMutation,
  useDeleteDiscoveryConfigurationMutation,
  useTestDiscoveryConfigurationMutation,
  useListSourceCapabilitiesQuery,
  useBrowseDiscoveryTargetsInfiniteQuery,
} = dataDiscoveryConfigurationsApi;

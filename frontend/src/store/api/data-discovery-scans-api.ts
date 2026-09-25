import type {
  DataTableQueryArgs,
  PaginatedResponse,
} from "@/components/data-table/types";
import type { FileResultStatus, ScanStatus } from "@/lib/data-discovery";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";

export type ScanCounters = {
  total_targets: number;
  completed_targets: number;
  failed_targets: number;
  files_discovered: number;
  files_supported: number;
  files_skipped: number;
  files_processed: number;
  files_succeeded: number;
  files_failed: number;
  findings_total: number;
  bytes_processed: number;
};

export type DiscoveryScanListItem = {
  id: number;
  policy_id: number;
  policy_name: string;
  status: ScanStatus;
  error_code: string;
  counters: ScanCounters;
  started_at: string | null;
  finished_at: string | null;
  created_at: string;
  updated_at: string;
};

export type DiscoveryScanTarget = {
  position: number;
  target: string;
  status: ScanStatus;
  error_code: string;
  files_discovered: number;
  files_skipped: number;
  files_succeeded: number;
  files_failed: number;
  findings_total: number;
  started_at: string | null;
  finished_at: string | null;
};

export type DiscoveryScan = DiscoveryScanListItem & {
  targets: DiscoveryScanTarget[];
};

export type FileFinding = {
  rule_id: number;
  rule_name: string;
  rule_type: string;
  count: number;
  offsets: number[];
};

export type DiscoveryFileResult = {
  id: number;
  target_position: number;
  file_key: string;
  file_name: string;
  extension: string;
  mime_type: string;
  size_bytes: number;
  modified_at: string | null;
  status: FileResultStatus;
  error_code: string;
  findings_total: number;
  findings: FileFinding[];
  bytes_processed: number;
  duration_ms: number;
  processed_at: string;
};

export type ListScanFilesArgs = DataTableQueryArgs & { scanId: number };

const BASE_URL = "/admin/data-discovery/scans";

export const dataDiscoveryScansApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listDiscoveryScans: builder.query<PaginatedResponse<DiscoveryScanListItem>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "DiscoveryScan" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({
          type: "DiscoveryScan" as const,
          id: item.id,
        })),
      ],
    }),

    getDiscoveryScan: builder.query<DiscoveryScan, number>({
      query: (id) => `${BASE_URL}/${id}`,
      providesTags: (result, error, id) => [{ type: "DiscoveryScan", id }],
    }),

    listDiscoveryScanFiles: builder.query<PaginatedResponse<DiscoveryFileResult>, ListScanFilesArgs>({
      query: ({ scanId, ...args }) => ({ url: `${BASE_URL}/${scanId}/files`, params: listParams(args) }),
      providesTags: (result, error, arg) => [{ type: "DiscoveryScan", id: `FILES-${arg.scanId}` }],
    }),

    createDiscoveryScan: builder.mutation<DiscoveryScan, { policy_id: number }>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [{ type: "DiscoveryScan", id: "LIST" }],
    }),
  }),
});

export const {
  useListDiscoveryScansQuery,
  useGetDiscoveryScanQuery,
  useListDiscoveryScanFilesQuery,
  useCreateDiscoveryScanMutation,
} = dataDiscoveryScansApi;

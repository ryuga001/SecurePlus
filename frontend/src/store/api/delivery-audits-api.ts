import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";

export type DeliveryStatus = "PROCESSING" | "SUCCESS" | "FAILED";

export type DeliveryFailureType = "RULE" | "PROCESSING" | "DKIM" | "RELAY" | "UNKNOWN";

export type DeliveryAuditListItem = {
  correlation_id: string;
  message_id: string;
  from: string;
  sender_domain: string;
  recipients: string[];
  recipient_count: number;
  status: DeliveryStatus;
  failure_type: DeliveryFailureType | "";
  attempt_count: number;
  size: number;
  created_at: string;
  updated_at: string;
};

export type DeliveryAuditRecipient = {
  email: string;
  domain: string;
  status: string;
  smtp_code: number;
  error: string;
};

export type DeliveryAuditAttempt = {
  number: number;
  started_at: string;
  finished_at: string;
  mx_host: string;
  tls: string;
  smtp_code: number;
  error: string;
};

export type DeliveryAudit = {
  correlation_id: string;
  message_id: string;
  config_id: number;
  from: string;
  sender_domain: string;
  recipients: DeliveryAuditRecipient[];
  status: DeliveryStatus;
  failure: { type: DeliveryFailureType; reason: string; smtp_code: number } | null;
  dkim: { domain: string; selector: string; signed: boolean };
  attempts: DeliveryAuditAttempt[];
  size: number;
  created_at: string;
  updated_at: string;
};

const BASE_URL = "/admin/email/audits";

export const deliveryAuditsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listDeliveryAudits: builder.query<PaginatedResponse<DeliveryAuditListItem>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "AuditLog" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({
          type: "AuditLog" as const,
          id: item.correlation_id,
        })),
      ],
    }),

    getDeliveryAudit: builder.query<DeliveryAudit, string>({
      query: (correlationId) => ({ url: `${BASE_URL}/${correlationId}` }),
      providesTags: (result, error, correlationId) => [{ type: "AuditLog", id: correlationId }],
    }),
  }),
});

export const {
  useListDeliveryAuditsQuery,
  useGetDeliveryAuditQuery,
  useLazyGetDeliveryAuditQuery,
} = deliveryAuditsApi;

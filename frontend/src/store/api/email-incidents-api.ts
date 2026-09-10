import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";

export type IncidentTrigger = "RESTRICTION" | "CONTENT";

export type IncidentAction = "BLOCK" | "QUARANTINE" | "REDACT" | "AUDIT";

export type IncidentActionStatus = "PENDING" | "INVOKED" | "FAILED";

export type RestrictionKind = "DOMAIN" | "ATTACHMENT";

export type EmailIncidentListItem = {
  correlation_id: string;
  message_id: string;
  from: string;
  sender_domain: string;
  recipients: string[];
  recipient_count: number;
  decision: string;
  trigger: IncidentTrigger;
  effective_action: IncidentAction;
  action_status: IncidentActionStatus;
  withheld_count: number;
  violation_count: number;
  match_count: number;
  created_at: string;
};

export type IncidentRecipient = {
  email: string;
  domain: string;
};

export type IncidentWithheldRecipient = {
  email: string;
  domain: string;
  policy_id: number;
  policy_name: string;
};

export type IncidentViolation = {
  kind: RestrictionKind;
  mode: "BLOCK" | "ALLOW";
  value: string;
  filename: string;
  content_type: string;
  policy_id: number;
  policy_name: string;
};

export type IncidentMatch = {
  policy_id: number;
  policy_name: string;
  rule_id: number;
  rule_name: string;
  rule_type: "KEYWORD" | "REGEX";
  configured_value: string;
  occurrences: number;
  locations: string[];
};

export type EmailIncident = {
  correlation_id: string;
  message_id: string;
  config_id: number;
  from: string;
  sender_domain: string;
  recipients: IncidentRecipient[];
  email_user_id: number;
  evaluated_policy_count: number;
  triggered_policy_ids: number[];
  decision: string;
  trigger: IncidentTrigger;
  effective_action: IncidentAction;
  action_invoked: string;
  action_status: IncidentActionStatus;
  action_error: string;
  withheld_recipients: IncidentWithheldRecipient[];
  restriction_violations: IncidentViolation[];
  matches: IncidentMatch[];
  created_at: string;
  updated_at: string;
};

const BASE_URL = "/admin/email/incidents";

export const emailIncidentsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listEmailIncidents: builder.query<PaginatedResponse<EmailIncidentListItem>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "EmailIncident" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({
          type: "EmailIncident" as const,
          id: item.correlation_id,
        })),
      ],
    }),

    getEmailIncident: builder.query<EmailIncident, string>({
      query: (correlationId) => ({ url: `${BASE_URL}/${correlationId}` }),
      providesTags: (result, error, correlationId) => [
        { type: "EmailIncident", id: correlationId },
      ],
    }),
  }),
});

export const {
  useListEmailIncidentsQuery,
  useGetEmailIncidentQuery,
  useLazyGetEmailIncidentQuery,
} = emailIncidentsApi;

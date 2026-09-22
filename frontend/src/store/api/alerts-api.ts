import type {
  DataTableQueryArgs,
  PaginatedResponse,
} from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";
import type { PolicyReference } from "@/store/api/policies-api";

export type NotificationType = "EMAIL" | "SMS";

export type ScheduleType = "REAL_TIME" | "CUSTOM";

export type AlertType = "SYSTEM" | "APPLICATION";

export type AlertListItem = {
  id: string;
  name: string;
  schedule_type: ScheduleType;
  notification_type: NotificationType;
  alert_type: AlertType;
  policies: PolicyReference[];
  policy_count: number;
  target_count: number;
  created_at: string;
  updated_at: string;
};

export type Alert = {
  id: string;
  name: string;
  schedule_type: ScheduleType;
  notification_type: NotificationType;
  target: string[];
  alert_type: AlertType;
  policies: PolicyReference[];
  created_at: string;
  updated_at: string;
};

export type AlertInput = {
  name: string;
  schedule_type: ScheduleType;
  notification_type: NotificationType;
  target: string[];
  policy_ids: number[];
};

export type UpdateAlertArgs = AlertInput & { id: string };

const BASE_URL = "/admin/email/alerts";

export const alertsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listAlerts: builder.query<PaginatedResponse<AlertListItem>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "Alert" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({ type: "Alert" as const, id: item.id })),
      ],
    }),

    getAlert: builder.query<Alert, string>({
      query: (id) => `${BASE_URL}/${id}`,
      providesTags: (result, error, id) => [{ type: "Alert", id }],
    }),

    createAlert: builder.mutation<Alert, AlertInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [{ type: "Alert", id: "LIST" }],
    }),

    updateAlert: builder.mutation<Alert, UpdateAlertArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "Alert", id: arg.id },
        { type: "Alert", id: "LIST" },
      ],
    }),

    deleteAlert: builder.mutation<void, string>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "Alert", id },
        { type: "Alert", id: "LIST" },
      ],
    }),
  }),
});

export const {
  useListAlertsQuery,
  useGetAlertQuery,
  useCreateAlertMutation,
  useUpdateAlertMutation,
  useDeleteAlertMutation,
} = alertsApi;

import { baseApi } from "@/store/api/base-api";

export type AnalyticsPeriod = "day" | "week" | "month";

export type AnalyticsWindow = {
  from: string;
  to: string;
};

export type TrendPoint = {
  bucket_start: string;
  total: number;
  blocked: number;
  flagged: number;
};

export type AnalyticsSummary = {
  total: number;
  blocked: number;
  flagged: number;
};

export type SenderRanking = {
  email: string;
  total: number;
  blocked: number;
  flagged: number;
};

export type PolicyRanking = {
  policy_id: number;
  policy_name: string;
  count: number;
};

export type RuleRanking = {
  rule_id: number;
  rule_name: string;
  rule_type: string;
  count: number;
};

export type EmailAnalytics = {
  period: AnalyticsPeriod;
  trend_window: AnalyticsWindow;
  top_window: AnalyticsWindow;
  summary: AnalyticsSummary;
  trend: {
    bucket: AnalyticsPeriod;
    points: TrendPoint[];
  };
  top_users: SenderRanking[];
  top_policies: PolicyRanking[];
  top_rules: RuleRanking[];
};

export const emailAnalyticsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getEmailAnalytics: builder.query<EmailAnalytics, AnalyticsPeriod>({
      query: (period) => ({ url: "/admin/email/analytics", params: { period } }),
      providesTags: (result, error, period) => [{ type: "EmailAnalytics", id: period }],
    }),
  }),
});

export const { useGetEmailAnalyticsQuery } = emailAnalyticsApi;

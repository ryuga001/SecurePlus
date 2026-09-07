import type { DataTableQueryArgs, PaginatedResponse } from "@/components/data-table/types";
import { baseApi } from "@/store/api/base-api";
import { listParams } from "@/store/api/list-params";

export type RuleType = "REGEX" | "KEYWORD";

export type Rule = {
  id: number;
  rule_name: string;
  type: RuleType;
  value: string;
  created_at: string;
  updated_at: string;
};

export type RuleInput = {
  rule_name: string;
  type: RuleType;
  value: string;
};

export type UpdateRuleArgs = RuleInput & { id: number };

const BASE_URL = "/admin/rules";

export const rulesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    listRules: builder.query<PaginatedResponse<Rule>, DataTableQueryArgs>({
      query: (args) => ({ url: BASE_URL, params: listParams(args) }),
      providesTags: (result) => [
        { type: "Rule" as const, id: "LIST" },
        ...(result?.items ?? []).map((item) => ({ type: "Rule" as const, id: item.id })),
      ],
    }),

    createRule: builder.mutation<Rule, RuleInput>({
      query: (body) => ({ url: BASE_URL, method: "POST", body }),
      invalidatesTags: [{ type: "Rule", id: "LIST" }],
    }),

    updateRule: builder.mutation<Rule, UpdateRuleArgs>({
      query: ({ id, ...body }) => ({ url: `${BASE_URL}/${id}`, method: "PUT", body }),
      invalidatesTags: (result, error, arg) => [
        { type: "Rule", id: arg.id },
        { type: "Rule", id: "LIST" },
        { type: "EmailPolicy", id: "LIST" },
      ],
    }),

    deleteRule: builder.mutation<void, number>({
      query: (id) => ({ url: `${BASE_URL}/${id}`, method: "DELETE" }),
      invalidatesTags: (result, error, id) => [
        { type: "Rule", id },
        { type: "Rule", id: "LIST" },
        { type: "EmailPolicy", id: "LIST" },
      ],
    }),
  }),
});

export const {
  useListRulesQuery,
  useLazyListRulesQuery,
  useCreateRuleMutation,
  useUpdateRuleMutation,
  useDeleteRuleMutation,
} = rulesApi;

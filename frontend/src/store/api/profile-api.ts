import { baseApi } from "@/store/api/base-api";

export type Profile = {
  org_name: string;
  first_name: string;
  last_name: string;
  admin_email: string;
};

export type ProfileUpdate = Partial<Pick<Profile, "org_name" | "first_name" | "last_name">>;

export const profileApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getProfile: builder.query<Profile, void>({
      query: () => "/admin/profile",
      providesTags: ["Profile"],
    }),
    updateProfile: builder.mutation<Profile, ProfileUpdate>({
      query: (body) => ({
        url: "/admin/profile",
        method: "PATCH",
        body,
      }),
      async onQueryStarted(_, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled;
          dispatch(profileApi.util.upsertQueryData("getProfile", undefined, data));
        } catch {
          return;
        }
      },
    }),
  }),
});

export const { useGetProfileQuery, useUpdateProfileMutation } = profileApi;

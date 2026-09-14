import type { Language, Theme } from "@/lib/api";
import { baseApi } from "@/store/api/base-api";

export type BrandingUpdate = {
  theme?: Theme;
  language?: Language;
  timezone?: string;
};

export const brandingApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    updateBranding: builder.mutation<void, BrandingUpdate>({
      query: (body) => ({
        url: "/admin/branding",
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["Branding"],
    }),
    uploadLogo: builder.mutation<void, File>({
      query: (file) => {
        const body = new FormData();
        body.append("logo", file);

        return {
          url: "/admin/branding/logo",
          method: "POST",
          body,
        };
      },
      invalidatesTags: ["Branding"],
    }),
    removeLogo: builder.mutation<void, void>({
      query: () => ({
        url: "/admin/branding/logo",
        method: "DELETE",
      }),
      invalidatesTags: ["Branding"],
    }),
  }),
});

export const {
  useUpdateBrandingMutation,
  useUploadLogoMutation,
  useRemoveLogoMutation,
} = brandingApi;

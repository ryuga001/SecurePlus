import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async redirects() {
    return [
      {
        source: "/admin/rules",
        destination: "/admin/content-inspection/rules",
        permanent: true,
      },
    ];
  },
};

export default nextConfig;

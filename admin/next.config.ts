import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Every page depends on the signed-in user, whose permissions the API
  // reloads on each request, so pages render per request and nothing is
  // worth prerendering or caching (Cache Components stays off).
  output: "standalone",
  poweredByHeader: false,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;

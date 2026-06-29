import path from "path";

const libsodiumCjs = path.resolve(
  import.meta.dirname,
  "node_modules/libsodium-wrappers/dist/modules/libsodium-wrappers.js",
);

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  serverExternalPackages: ["isomorphic-dompurify"],
  webpack: (config) => {
    // The published ESM build of libsodium-wrappers imports a sibling
    // libsodium.mjs that is not shipped; point the bundler at the
    // self-contained CommonJS build instead.
    config.resolve.alias = {
      ...config.resolve.alias,
      "libsodium-wrappers$": libsodiumCjs,
    };
    return config;
  },
  async rewrites() {
    return [
      {
        source: "/api/v3/:path*",
        destination: `${process.env.API_URL || "http://localhost:8080"}/api/v3/:path*`,
      },
      {
        source: "/api/:path*",
        destination: `${process.env.API_URL || "http://localhost:8080"}/:path*`,
      },
    ];
  },
};

export default nextConfig;

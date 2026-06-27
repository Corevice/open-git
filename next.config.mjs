/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
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

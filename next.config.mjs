/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  // isomorphic-dompurify pulls in jsdom, which loads resource files (e.g.
  // default-stylesheet.css) at runtime. Keep it out of the server bundle so
  // those files resolve from node_modules instead of being webpack-inlined.
  serverExternalPackages: ['isomorphic-dompurify'],
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.API_URL || 'http://localhost:8080'}/:path*`,
      },
    ];
  },
};

export default nextConfig;

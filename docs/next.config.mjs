import nextra from 'nextra';

const withNextra = nextra({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
});

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  // Static export (required so the Pagefind postbuild can index `out/`) is not
  // compatible with the default next/image optimization loader.
  images: {
    unoptimized: true,
  },
};

export default withNextra(nextConfig);

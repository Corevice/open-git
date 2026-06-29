import nextra from 'nextra';

const withNextra = nextra({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
});

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  // Static export to `out/` (so the Pagefind postbuild and the docs CI
  // workflow can index it) is not compatible with the default next/image
  // optimization loader.
  images: {
    unoptimized: true,
  },
};

export default withNextra(nextConfig);

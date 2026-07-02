// The docs site (Nextra) does not use the root project's Tailwind pipeline.
// Without this local config, Next.js walks up the directory tree and loads the
// repository-root postcss.config.mjs, which requires @tailwindcss/postcss — a
// dependency that is not installed in docs/. An empty plugin set keeps the docs
// build self-contained.
const config = {
  plugins: {},
};

export default config;

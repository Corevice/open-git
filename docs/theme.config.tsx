import type { DocsThemeConfig } from 'nextra-theme-docs';

const config: DocsThemeConfig = {
  logo: <span>open-git</span>,
  docsRepositoryBase: 'https://github.com/Corevice/open-git/blob/main/docs',
  footer: {
    text: (
      <>
        MIT Licensed. © {new Date().getFullYear()} Corevice. open-git is open source
        software.
      </>
    ),
  },
  useNextSeoProps() {
    return {
      titleTemplate: '%s – open-git Docs',
    };
  },
};

export default config;

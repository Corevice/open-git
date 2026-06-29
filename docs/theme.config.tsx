import type { DocsThemeConfig } from 'nextra-theme-docs';
import { DocHeader } from './components/DocHeader';
import { EditPageLink } from './components/EditPageLink';

const config: DocsThemeConfig = {
  logo: <span>open-git</span>,
  docsRepositoryBase: 'https://github.com/Corevice/open-git/blob/main/docs',
  navbar: {
    component: <DocHeader />,
  },
  editLink: {
    component: EditPageLink,
  },
  i18n: [
    { locale: 'ja', text: '日本語' },
    { locale: 'en', text: 'English' },
  ],
  defaultLocale: 'ja',
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

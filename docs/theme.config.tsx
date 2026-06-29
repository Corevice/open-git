import type { DocsThemeConfig } from 'nextra-theme-docs';
import { useConfig } from 'nextra-theme-docs';
import { usePathname } from 'next/navigation';
import { useRouter } from 'next/router';
import type { ReactNode } from 'react';
import Search from './components/Search';
import { DocHeader } from './components/DocHeader';
import { EditPageLink } from './components/EditPageLink';
import { FeedbackWidget } from './components/FeedbackWidget';
import { UntranslatedBanner } from './components/UntranslatedBanner';

function DocsMain({ children }: { children: ReactNode }) {
  const { frontMatter } = useConfig();
  const pathname = usePathname();
  const { locale } = useRouter();
  const pageLang =
    typeof frontMatter.lang === 'string' ? frontMatter.lang : undefined;
  const version =
    typeof frontMatter.version === 'string' ? frontMatter.version : 'latest';

  return (
    <>
      <UntranslatedBanner locale={locale} pageLang={pageLang} />
      {children}
      <FeedbackWidget path={pathname ?? ''} version={version} />
    </>
  );
}

const config: DocsThemeConfig = {
  logo: <span>open-git</span>,
  main: DocsMain,
  docsRepositoryBase: 'https://github.com/Corevice/open-git/blob/main/docs',
  search: {
    component: Search,
  },
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

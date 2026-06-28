import type { DocsThemeConfig } from "nextra-theme-docs";
import { useConfig } from "nextra-theme-docs";
import { usePathname } from "next/navigation";
import { useRouter } from "next/router";
import type { ReactNode } from "react";
import { FeedbackWidget } from "./components/FeedbackWidget";
import { UntranslatedBanner } from "./components/UntranslatedBanner";

function DocsMain({ children }: { children: ReactNode }) {
  const { frontMatter } = useConfig();
  const pathname = usePathname();
  const { locale } = useRouter();
  const pageLang =
    typeof frontMatter.lang === "string" ? frontMatter.lang : undefined;
  const version =
    typeof frontMatter.version === "string"
      ? frontMatter.version
      : "latest";

  return (
    <>
      <UntranslatedBanner locale={locale} pageLang={pageLang} />
      {children}
      <FeedbackWidget path={pathname ?? ""} version={version} />
    </>
  );
}

const config: DocsThemeConfig = {
  logo: <span>Open Git Docs</span>,
  main: DocsMain,
};

export default config;

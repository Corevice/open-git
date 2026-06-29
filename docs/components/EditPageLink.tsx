import { useConfig } from 'nextra-theme-docs';

type EditPageLinkProps = {
  filePath: string;
};

export function EditPageLink({ filePath }: EditPageLinkProps) {
  const { docsRepositoryBase } = useConfig();
  const href = `${docsRepositoryBase}/tree/main/${filePath}`;

  return <a href={href}>このページを編集</a>;
}

type EditPageLinkProps = {
  filePath: string;
};

const DOCS_REPOSITORY_BASE =
  'https://github.com/Corevice/open-git/blob/main/docs';

export function EditPageLink({ filePath }: EditPageLinkProps) {
  const href = `${DOCS_REPOSITORY_BASE}/${filePath}`;

  return <a href={href}>このページを編集</a>;
}

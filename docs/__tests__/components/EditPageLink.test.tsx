import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

const { docsRepositoryBase } = vi.hoisted(() => ({
  docsRepositoryBase: 'https://github.com/Corevice/open-git',
}));

vi.mock('nextra-theme-docs', () => ({
  useConfig: () => ({
    docsRepositoryBase,
  }),
}));

import { EditPageLink } from '../../components/EditPageLink';

describe('EditPageLink', () => {
  it('renders edit link with docsRepositoryBase in href', () => {
    render(<EditPageLink filePath="pages/getting-started.mdx" />);

    const link = screen.getByRole('link', { name: 'このページを編集' });
    expect(link.getAttribute('href')).toContain(docsRepositoryBase);
  });
});

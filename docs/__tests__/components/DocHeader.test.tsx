import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

vi.mock('../../components/VersionSelector', () => ({
  VersionSelector: () => <div data-testid="version-selector" />,
}));

vi.mock('../../components/LanguageSelector', () => ({
  LanguageSelector: () => <div data-testid="language-selector" />,
}));

vi.mock('../../components/ThemeToggle', () => ({
  ThemeToggle: () => <div data-testid="theme-toggle" />,
}));

import { DocHeader } from '../../components/DocHeader';

describe('DocHeader', () => {
  it('renders logo text and GitHub link', () => {
    render(<DocHeader />);

    expect(screen.getByText('open-git')).toBeInTheDocument();

    const githubLink = screen.getByRole('link', { name: 'GitHub' });
    expect(githubLink.getAttribute('href')).toContain('github.com/Corevice/open-git');
  });

  it('renders mocked child components', () => {
    render(<DocHeader />);

    expect(screen.getByTestId('version-selector')).toBeInTheDocument();
    expect(screen.getByTestId('language-selector')).toBeInTheDocument();
    expect(screen.getByTestId('theme-toggle')).toBeInTheDocument();
  });
});

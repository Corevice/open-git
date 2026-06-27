import { expect, test } from '@playwright/test';

test('homepage renders', async ({ page }) => {
  await page.goto('/');
  expect(await page.title()).toBeTruthy();
});

test('theme toggle changes html class', async ({ page }) => {
  await page.goto('/');

  const themeToggle = page.getByTestId('theme-toggle');
  await themeToggle.click();

  const htmlClass = (await page.locator('html').getAttribute('class')) ?? '';
  expect(htmlClass.includes('dark') || htmlClass.includes('light')).toBe(true);
});

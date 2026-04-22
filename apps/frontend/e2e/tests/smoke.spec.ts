import { test, expect } from '@playwright/test';

test('homepage loads', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveTitle(/.+/);
});

test('scans page — primary action buttons are present', async ({ page }) => {
  await page.goto('/scans');
  await expect(page.getByTestId('scan-all-sources-btn')).toBeVisible();
  await expect(page.getByTestId('new-scan-btn')).toBeVisible();
});

test('findings page — filters and search inputs are present', async ({ page }) => {
  await page.goto('/findings');
  await expect(page.getByTestId('pii-type-filter')).toBeVisible();
  await expect(page.getByTestId('severity-filter')).toBeVisible();
  await expect(page.getByTestId('findings-search-input')).toBeVisible();
});

test('connectors page — add connector button is present', async ({ page }) => {
  await page.goto('/connectors');
  await expect(page.getByRole('button', { name: /add connector/i })).toBeVisible();
});

test('global layout — add source quick action is present on desktop', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto('/');
  await expect(page.getByTestId('add-source-quick-action-btn')).toBeVisible();
});

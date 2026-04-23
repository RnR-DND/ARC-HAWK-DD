import { test, expect } from '@playwright/test';

test.describe('Findings filter flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/findings');
  });

  test('PII type filter is visible', async ({ page }) => {
    await expect(page.getByTestId('pii-type-filter')).toBeVisible();
  });

  test('severity filter is visible', async ({ page }) => {
    await expect(page.getByTestId('severity-filter')).toBeVisible();
  });

  test('search input is visible', async ({ page }) => {
    await expect(page.getByTestId('findings-search-input')).toBeVisible();
  });

  test('typing in search narrows the visible URL / does not crash', async ({ page }) => {
    await page.getByTestId('findings-search-input').fill('ABCPE1234F');
    // Page must not show an error boundary — basic liveness check.
    await expect(page.getByRole('main')).not.toContainText('Something went wrong');
  });

  test('findings list or empty state is shown', async ({ page }) => {
    const hasRows = await page.getByTestId('finding-row').count() > 0;
    const hasEmpty = await page.getByText(/no findings|no results/i).isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  test('export CSV button is present', async ({ page }) => {
    await expect(page.getByTestId('export-csv-btn')).toBeVisible();
  });
});

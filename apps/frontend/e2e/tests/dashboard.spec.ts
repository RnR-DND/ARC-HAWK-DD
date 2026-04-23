import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('renders risk score metric card', async ({ page }) => {
    await expect(page.getByTestId('metric-risk-score')).toBeVisible();
  });

  test('renders findings metric card', async ({ page }) => {
    await expect(page.getByTestId('metric-findings')).toBeVisible();
  });

  test('renders assets metric card', async ({ page }) => {
    await expect(page.getByTestId('metric-assets')).toBeVisible();
  });

  test('renders scans metric card', async ({ page }) => {
    await expect(page.getByTestId('metric-scans')).toBeVisible();
  });

  test('risk score trend chart is present', async ({ page }) => {
    await expect(page.getByTestId('risk-score-trend-chart')).toBeVisible();
  });

  test('quick action add-source button navigates to connectors', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.getByTestId('add-source-quick-action-btn').click();
    await expect(page).toHaveURL(/\/connectors/);
  });
});

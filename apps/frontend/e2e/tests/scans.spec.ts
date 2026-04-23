import { test, expect } from '@playwright/test';

test.describe('Scan trigger flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/scans');
  });

  test('scan all sources button is present', async ({ page }) => {
    await expect(page.getByTestId('scan-all-sources-btn')).toBeVisible();
  });

  test('new scan button is present', async ({ page }) => {
    await expect(page.getByTestId('new-scan-btn')).toBeVisible();
  });

  test('new scan modal opens on button click', async ({ page }) => {
    await page.getByTestId('new-scan-btn').click();
    await expect(page.getByRole('dialog')).toBeVisible();
  });

  test('new scan modal has a submit button', async ({ page }) => {
    await page.getByTestId('new-scan-btn').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByRole('button', { name: /start scan|trigger|run/i })).toBeVisible();
  });

  test('new scan modal closes on cancel', async ({ page }) => {
    await page.getByTestId('new-scan-btn').click();
    await page.getByRole('dialog').getByRole('button', { name: /cancel/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible();
  });

  test('scans list or empty state is shown', async ({ page }) => {
    const hasRows = await page.getByTestId('scan-row').count() > 0;
    const hasEmpty = await page.getByText(/no scans|run your first/i).isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });
});

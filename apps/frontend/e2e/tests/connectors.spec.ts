import { test, expect } from '@playwright/test';

test.describe('Connectors page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/connectors');
  });

  test('add connector button is visible', async ({ page }) => {
    await expect(page.getByRole('button', { name: /add connector/i })).toBeVisible();
  });

  test('connector list renders (empty or populated)', async ({ page }) => {
    // Either an empty-state message or a list of connector cards must appear.
    const hasList = await page.getByTestId('connector-card').count() > 0;
    const hasEmpty = await page.getByText(/no connectors|add your first/i).isVisible().catch(() => false);
    expect(hasList || hasEmpty).toBeTruthy();
  });

  test('connector health section is visible', async ({ page }) => {
    await expect(page.getByTestId('connector-health-section')).toBeVisible();
  });

  test('add connector modal opens on button click', async ({ page }) => {
    await page.getByRole('button', { name: /add connector/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible();
  });

  test('add connector modal has source type selector', async ({ page }) => {
    await page.getByRole('button', { name: /add connector/i }).click();
    await expect(page.getByRole('dialog').getByRole('combobox')).toBeVisible();
  });

  test('add connector modal closes on cancel', async ({ page }) => {
    await page.getByRole('button', { name: /add connector/i }).click();
    await page.getByRole('dialog').getByRole('button', { name: /cancel/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible();
  });
});
